#!/usr/bin/env node
// Tiny webhook → Discord relay. Receives typed payloads (essentia daily_digest,
// lifehub_weekly) at POST /notify, formats them as Discord markdown, and
// forwards to a Discord channel webhook. Single-purpose, no framework, no deps beyond Node 22's
// built-in http + fetch.

import http from 'node:http';
import { chunk, formatDigest, formatLifehubWeekly } from './format.mjs';

const FORMATS = { daily_digest: formatDigest, lifehub_weekly: formatLifehubWeekly };

const WEBHOOK_URL = process.env.DISCORD_WEBHOOK_URL;
const PORT = parseInt(process.env.PORT ?? '8080', 10);

if (!WEBHOOK_URL) {
  console.error('DISCORD_WEBHOOK_URL env var required');
  process.exit(1);
}

async function sendDiscord(text) {
  for (const content of chunk(text)) {
    const res = await fetch(WEBHOOK_URL, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content, allowed_mentions: { parse: [] } }),
    });
    if (!res.ok) {
      const body = await res.text();
      throw new Error(`Discord webhook ${res.status}: ${body}`);
    }
  }
}

const server = http.createServer((req, res) => {
  if (req.method === 'GET' && req.url === '/healthz') {
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    res.end('ok');
    return;
  }
  if (req.method !== 'POST' || req.url !== '/notify') {
    res.writeHead(404);
    res.end();
    return;
  }
  let body = '';
  req.on('data', (c) => { body += c; });
  req.on('end', () => {
    void (async () => {
      try {
        const payload = JSON.parse(body);
        const format = FORMATS[payload.type];
        if (!format) {
          res.writeHead(400);
          res.end(`unsupported type: ${payload.type ?? '(missing)'}`);
          return;
        }
        await sendDiscord(format(payload));
        console.log(`relay ok type=${payload.type}`);
        res.writeHead(204);
        res.end();
      } catch (err) {
        console.error('relay error:', err);
        res.writeHead(500);
        res.end(String(err));
      }
    })();
  });
});

server.listen(PORT, () => {
  console.log(`discord-relay listening on :${PORT}`);
});
