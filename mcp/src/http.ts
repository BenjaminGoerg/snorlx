import { randomUUID, timingSafeEqual } from 'node:crypto';

import { StreamableHTTPServerTransport } from '@modelcontextprotocol/sdk/server/streamableHttp.js';
import express from 'express';

import { SnorlxApiClient } from './client.js';
import { createSnorlxMcpServer } from './server.js';

const DEFAULT_SESSION_TTL_MS = 30 * 60 * 1000;
const SESSION_SWEEP_INTERVAL_MS = 60 * 1000;
const LOOPBACK_HOSTS = ['127.0.0.1', 'localhost', '::1', '[::1]'];

function safeEqual(a: string, b: string): boolean {
  const left = Buffer.from(a);
  const right = Buffer.from(b);
  if (left.length !== right.length) {
    return false;
  }
  return timingSafeEqual(left, right);
}

function requireMcpHttpAuth(httpToken: string): express.RequestHandler {
  return (req, res, next) => {
    const header = req.get('authorization') || '';
    if (!header.startsWith('Bearer ')) {
      res.status(401).json({ error: 'Unauthorized', message: 'Bearer MCP_HTTP_TOKEN required' });
      return;
    }
    const presented = header.slice('Bearer '.length).trim();
    if (!presented || !safeEqual(presented, httpToken)) {
      res.status(401).json({ error: 'Unauthorized', message: 'Invalid MCP HTTP token' });
      return;
    }
    next();
  };
}

/**
 * Builds the Host allowlist used against DNS rebinding. An explicit list wins; a loopback bind
 * defaults to the loopback names; any other bind returns null, which disables the check because
 * a reverse proxy in front of the server is expected to validate the Host header.
 */
export function resolveAllowedHosts(bindHost: string, port: number, explicit?: string): string[] | null {
  const configured = (explicit || '')
    .split(',')
    .map((h) => h.trim().toLowerCase())
    .filter(Boolean);
  if (configured.length > 0) {
    return configured;
  }
  if (!LOOPBACK_HOSTS.includes(bindHost)) {
    return null;
  }
  return LOOPBACK_HOSTS.flatMap((h) => [h, `${h}:${port}`]);
}

/** Rejects requests whose Host header is not in the allowlist (DNS rebinding protection). */
export function requireAllowedHost(allowedHosts: string[] | null): express.RequestHandler {
  const allowed = new Set((allowedHosts || []).map((h) => h.toLowerCase()));
  return (req, res, next) => {
    if (allowedHosts === null) {
      next();
      return;
    }
    const host = (req.get('host') || '').trim().toLowerCase();
    if (!host || !allowed.has(host)) {
      res.status(421).json({ error: 'Misdirected Request', message: 'Host header is not allowed' });
      return;
    }
    next();
  };
}

interface TrackedSession {
  transport: StreamableHTTPServerTransport;
  lastSeen: number;
}

export async function startHttpServer(opts: {
  client: SnorlxApiClient;
  port: number;
  host: string;
  httpToken: string;
  allowedHosts?: string;
  sessionTtlMs?: number;
}): Promise<void> {
  const app = express();
  app.disable('x-powered-by');
  app.use(express.json({ limit: '4mb' }));

  const sessions = new Map<string, TrackedSession>();
  const sessionTtlMs = opts.sessionTtlMs ?? DEFAULT_SESSION_TTL_MS;
  const auth = requireMcpHttpAuth(opts.httpToken);
  const hostCheck = requireAllowedHost(resolveAllowedHosts(opts.host, opts.port, opts.allowedHosts));

  const closeSession = async (id: string) => {
    const tracked = sessions.get(id);
    sessions.delete(id);
    if (tracked) {
      await tracked.transport.close().catch(() => undefined);
    }
  };

  const sweep = setInterval(() => {
    const cutoff = Date.now() - sessionTtlMs;
    for (const [id, tracked] of sessions) {
      if (tracked.lastSeen < cutoff) {
        void closeSession(id);
      }
    }
  }, SESSION_SWEEP_INTERVAL_MS);
  sweep.unref();

  const handle = async (req: express.Request, res: express.Response) => {
    const sessionId = req.headers['mcp-session-id'] as string | undefined;

    try {
      if (sessionId && sessions.has(sessionId)) {
        const tracked = sessions.get(sessionId)!;
        tracked.lastSeen = Date.now();
        await tracked.transport.handleRequest(req, res, req.body);
        return;
      }

      if (req.method === 'POST') {
        const transport = new StreamableHTTPServerTransport({
          sessionIdGenerator: () => randomUUID(),
          onsessioninitialized: (id) => {
            sessions.set(id, { transport, lastSeen: Date.now() });
          },
        });
        transport.onclose = () => {
          if (transport.sessionId) {
            sessions.delete(transport.sessionId);
          }
        };
        const server = createSnorlxMcpServer(opts.client);
        await server.connect(transport);
        await transport.handleRequest(req, res, req.body);
        return;
      }

      res.status(400).json({ error: 'Invalid or missing MCP session' });
    } catch (err) {
      // Details stay in the server log; clients only learn that the request failed.
      console.error('snorlx-mcp request failed:', err);
      if (!res.headersSent) {
        res.status(500).json({ error: 'Internal error' });
      }
    }
  };

  app.post('/mcp', hostCheck, auth, handle);
  app.get('/mcp', hostCheck, auth, handle);
  app.delete('/mcp', hostCheck, auth, handle);

  app.get('/health', (_req, res) => {
    res.json({ status: 'ok', service: 'snorlx-mcp' });
  });

  await new Promise<void>((resolve) => {
    app.listen(opts.port, opts.host, () => {
      console.error(`snorlx-mcp Streamable HTTP listening on http://${opts.host}:${opts.port}/mcp`);
      resolve();
    });
  });
}
