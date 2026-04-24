import { execFileSync } from 'node:child_process';
import { rmSync } from 'node:fs';
import { mkdtemp, readFile } from 'node:fs/promises';
import https from 'node:https';
import os from 'node:os';
import path from 'node:path';

const redirectUriRaw = process.env.E2E_OIDC_REDIRECT_URI;
if (!redirectUriRaw) {
  throw new Error('E2E_OIDC_REDIRECT_URI is required to start the callback server.');
}

let redirectUrl;
try {
  redirectUrl = new URL(redirectUriRaw);
} catch (error) {
  throw new Error(`E2E_OIDC_REDIRECT_URI is invalid: ${redirectUriRaw}`);
}

if (redirectUrl.protocol !== 'https:') {
  throw new Error(`E2E_OIDC_REDIRECT_URI must use https (got ${redirectUrl.protocol}).`);
}

if (!redirectUrl.port) {
  throw new Error('E2E_OIDC_REDIRECT_URI must include a non-privileged port for the callback server.');
}
const portRaw = redirectUrl.port;
const port = Number(portRaw);
if (!Number.isInteger(port) || port < 1024) {
  throw new Error(`E2E_OIDC_REDIRECT_URI must use a non-privileged port (>=1024). Got ${portRaw}.`);
}

const callbackPath = redirectUrl.pathname || '/callback';

const tempDir = await mkdtemp(path.join(os.tmpdir(), 'oidc-callback-'));
const keyPath = path.join(tempDir, 'oidc-key.pem');
const certPath = path.join(tempDir, 'oidc-cert.pem');

try {
  execFileSync(
    'openssl',
    [
      'req',
      '-x509',
      '-newkey',
      'rsa:2048',
      '-nodes',
      '-days',
      '1',
      '-subj',
      '/CN=localhost',
      '-keyout',
      keyPath,
      '-out',
      certPath,
    ],
    { stdio: 'ignore' },
  );
} catch (error) {
  throw new Error('Failed to generate a self-signed cert for the callback server. Ensure openssl is available.');
}

const [key, cert] = await Promise.all([readFile(keyPath, 'utf8'), readFile(certPath, 'utf8')]);

process.on('exit', () => {
  rmSync(tempDir, { recursive: true, force: true });
});

const server = https.createServer({ key, cert }, (req, res) => {
  const url = new URL(req.url ?? '/', `https://${redirectUrl.hostname}`);
  if (url.pathname === callbackPath) {
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end('<html>ok</html>');
    return;
  }
  if (url.pathname === '/healthz') {
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    res.end('ok');
    return;
  }
  res.writeHead(404, { 'Content-Type': 'text/plain' });
  res.end('Not found');
});

server.listen(port, '0.0.0.0', () => {
  console.log(`[oidc-callback] listening on ${port} for ${callbackPath}`);
});
