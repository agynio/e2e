import http from 'node:http';
import https from 'node:https';
import net from 'node:net';

const portRaw = process.env.TRACING_PROXY_PORT ?? '5100';
const port = Number(portRaw);
if (!Number.isInteger(port) || port <= 0) {
  throw new Error(`TRACING_PROXY_PORT must be a positive integer (got ${portRaw}).`);
}

const appTarget = new URL(process.env.TRACING_APP_TARGET ?? 'http://tracing-app.platform.svc.cluster.local:3000');
const gatewayTarget = new URL(
  process.env.TRACING_GATEWAY_TARGET ?? 'http://gateway-gateway.platform.svc.cluster.local:8080',
);

function resolveTargetPort(target) {
  if (target.port) return Number(target.port);
  return target.protocol === 'https:' ? 443 : 80;
}

function resolveProxyTarget(requestUrl) {
  const url = new URL(requestUrl ?? '/', 'http://proxy.local');
  if (url.pathname === '/api' || url.pathname.startsWith('/api/')) {
    const stripped = url.pathname.replace(/^\/api/, '') || '/';
    return { target: gatewayTarget, path: `${stripped}${url.search}` };
  }
  if (url.pathname === '/socket.io' || url.pathname.startsWith('/socket.io/')) {
    return { target: gatewayTarget, path: `${url.pathname}${url.search}` };
  }
  return { target: appTarget, path: `${url.pathname}${url.search}` };
}

function buildProxyHeaders(requestHeaders, target) {
  return {
    ...requestHeaders,
    host: target.host,
  };
}

function proxyHttpRequest(req, res) {
  const { target, path } = resolveProxyTarget(req.url);
  const client = target.protocol === 'https:' ? https : http;
  const proxyReq = client.request(
    {
      hostname: target.hostname,
      port: resolveTargetPort(target),
      method: req.method,
      path,
      headers: buildProxyHeaders(req.headers, target),
    },
    (proxyRes) => {
      res.writeHead(proxyRes.statusCode ?? 502, proxyRes.headers);
      proxyRes.pipe(res);
    },
  );

  proxyReq.on('error', () => {
    res.writeHead(502, { 'Content-Type': 'text/plain' });
    res.end('Bad gateway');
  });

  req.pipe(proxyReq);
}

function proxyWebSocket(req, socket, head) {
  const { target, path } = resolveProxyTarget(req.url);
  const proxySocket = net.connect(resolveTargetPort(target), target.hostname, () => {
    const headers = buildProxyHeaders(req.headers, target);
    let request = `GET ${path} HTTP/1.1\r\n`;
    for (const [key, value] of Object.entries(headers)) {
      if (Array.isArray(value)) {
        for (const entry of value) {
          request += `${key}: ${entry}\r\n`;
        }
        continue;
      }
      if (value !== undefined) {
        request += `${key}: ${value}\r\n`;
      }
    }
    request += '\r\n';
    proxySocket.write(request);
    if (head?.length) {
      proxySocket.write(head);
    }
    proxySocket.pipe(socket);
    socket.pipe(proxySocket);
  });

  proxySocket.on('error', () => {
    socket.destroy();
  });
}

const server = http.createServer(proxyHttpRequest);
server.on('upgrade', proxyWebSocket);

server.listen(port, () => {
  console.log(`[tracing-proxy] listening on ${port}`);
});
