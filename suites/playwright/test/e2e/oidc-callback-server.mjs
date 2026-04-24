import https from 'node:https';

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

const key = `-----BEGIN PRIVATE KEY-----
MIIEuwIBADANBgkqhkiG9w0BAQEFAASCBKUwggShAgEAAoIBAQCnryQ6mvD9vykz
OfJo51oJ9Ao8jVhDcRY/9hIpQM7WyxfoFj8POKVP0X1UOlWFinEnHPssYiJYPSH2
Nyxl5sNKsrjY9u53/JxEMEl4TaUXgN6Da+7kQIAbZFoZzjWLsDi84qYqoJMmNucr
XMFzjPM9Rm9wYxYaDr9H0u0kkypL64ikLVG4ucfmGFBUZapi/LhFwcs5rMJ+4Im3
6kZvFNAZDgubg3jXF4fEkEcM6SYv0+Qm+wqLC/bR7nvZI2hyKrv4Kqai3FtvNxW9
HLk2+3gxCV2r1IfhXw2RjGI7V+XJz+tIs6kaFrVMkirZWYIlWTq0RYQnKFqogbkU
oGyZERkdAgMBAAECggEAL9QH7GNnW6kb0k2z8/IRP4eJJ+5U/5+Q7ht84KFoneF9
5yf5QpkwpcymB9E/tYBgd/yPNACltS9ysWzZUBN7HqJNkS0Vpcm6tMRlIFhdP4/1
Z9zwXdB7+dQs0vF7WmWgOVgYd04nyp2cYETrtM6+Tnr5rD/G/RW5v33NQEJtrQuB
ldfpAYSpLk1zIcbjoMvJ0MhSfHqFcs4+FDo0BAbKHW2U2w5bJmdIT/TeXTNwxENu
pJN0rLtXr8DnRHfAlV4C3pqFI3kvmxBrJ07LWewB8phA0BX32pvL5MNEsGlTbGvj
HB8YFmetAI53KphFIeFHc9l1njNM2tPqqb7dViIBLQKBgQDnterRuq7wd7g2j1SO
gqCAW5pGr9XEFl6mz1FhX04+4LDh6gqZg7WBT1EHpuXQPikIxleAOajSp6o0PWPK
xY7zu85CtZtGImxkA7+skt7hxyEp0KUUiZ8DNq0YKhbOmKSZgmKE8Xcolaq5ME4H
09eCd7SnAfAV7ZayrDXSBi2RYwKBgQC5QwncEkm/IwRdpxMW0EbRiXdNCRRjqhHE
asqaqtwwjuERottSmXtE2VdRqqel+0BMNdA1uLcueeGUbpwQJhokcwMdy4FhRrX8
BeRDst6zqGv7z87OVoI6aqkJkYjoxQQMdt6igGlFXEqx+JfJVhbB50nc3MnKWaVW
nDedx0fzfwKBgBaNq3SMkjiPvpt46gcRCeRUhji5Jrp2XvInnck3iJswLadfq3Zz
znfuq3luMlJJqp7TB3NQqXEPps585zi2cAqjThlKKfnyodA+WSrIBO++/ShfyaGt
H5Alg0Wl2yBy1RqoCUTdZ/bIUpzB6eZzJTfqxOe4lZDc1l0/y+FMfqT3AoGBAK43
qsf3sr458dsYSM1FY7OcsEITbccjob2yJ4E3eAV595GcMuAEUXW3ZXP5Jdri4d5J
JNnAMRNVrprlQYG2MxNfzOhx/eM6mdy8taIsTV1p3tJY48QKekDxGLFU2Qj8bQhD
qK3sUBLX7a5bdnHxsUj7dexq/KB7mQ5PrcWEJ9eFAn8yLqTvyVjLhT4o41X7DPAZ
gCMKfwynhg8UAxjHQLQ8R6/jL7VIDicuWTBLt3L+ovG+XVnsS+PSoa9wprq6De1F
nd2dkjoaSngbczNnGLmpehDLUxR62mMixfY2LXyBcSM+TRF4SSnpv3iUzFQBmI7R
BUfI0arQz5vRZY54MonO
-----END PRIVATE KEY-----`;

const cert = `-----BEGIN CERTIFICATE-----
MIIDCTCCAfGgAwIBAgIUbKD85J49vO4zNaphx2AUg81vTCswDQYJKoZIhvcNAQEL
BQAwFDESMBAGA1UEAwwJbG9jYWxob3N0MB4XDTI2MDQyNDAzMzQ1MVoXDTM2MDQy
MTAzMzQ1MVowFDESMBAGA1UEAwwJbG9jYWxob3N0MIIBIjANBgkqhkiG9w0BAQEF
AAOCAQ8AMIIBCgKCAQEAp68kOprw/b8pMznyaOdaCfQKPI1YQ3EWP/YSKUDO1ssX
6BY/DzilT9F9VDpVhYpxJxz7LGIiWD0h9jcsZebDSrK42Pbud/ycRDBJeE2lF4De
g2vu5ECAG2RaGc41i7A4vOKmKqCTJjbnK1zBc4zzPUZvcGMWGg6/R9LtJJMqS+uI
pC1RuLnH5hhQVGWqYvy4RcHLOazCfuCJt+pGbxTQGQ4Lm4N41xeHxJBHDOkmL9Pk
JvsKiwv20e572SNociq7+CqmotxbbzcVvRy5Nvt4MQldq9SH4V8NkYxiO1flyc/r
SLOpGha1TJIq2VmCJVk6tEWEJyhaqIG5FKBsmREZHQIDAQABo1MwUTAdBgNVHQ4E
FgQURgoNhzw2K1aryfIDYboE0AEmOqgwHwYDVR0jBBgwFoAURgoNhzw2K1aryfID
YboE0AEmOqgwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAjm8u
GLRxIJqvj1C21NYuYOwdAgiNm40yBc8vAoO3kpR5kdHOcH6FzgZNDql6qD0KvRt2
WlopssbdVJRIe1dHQ5nvTcIaJAPpZFy8a+KTBHJTKl3MaOptLXmURcBO9zDvYsq5
FdW85Zt9HTyoNu+LcIN/cawdxA/Q0Gien3nBCxkVNuxWDPvj61hMF+K1K1SBrs5n
n0enCM18cnzXGF+Z6hRKxguePHXTJbLJhRPzX6Mg43IABXqBG96j+875N0sQgCe8
MCSyObSddH+q7dHvz4PasMqhryjK9OruLbAjFRMyLGCugzHa/ia70QVZdhrdcahi
lAGwj/yidSfuyHS5ig==
-----END CERTIFICATE-----`;

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
