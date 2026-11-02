#!/usr/bin/env node
// Tiny webhook → Discord relay. Receives typed payloads (essentia daily_digest,
// lifehub_weekly) at POST /notify, formats them as Discord markdown, and
// forwards to a Discord channel webhook. Single-purpose, no framework, no deps beyond Node 22's
// built-in http + fetch.

import http from 'node:http';
import { weeklyChartSvg, svgToPng } from './chart.mjs';
import { chunk, formatDigest, formatLifehubWeekly } from './format.mjs';

const FORMATS = { daily_digest: formatDigest, lifehub_weekly: formatLifehubWeekly };

const WEBHOOK_URL = process.env.DISCORD_WEBHOOK_URL;
const PORT = parseInt(process.env.PORT ?? '8080', 10);

if (!WEBHOOK_URL) {
  console.error('DISCORD_WEBHOOK_URL env var required');
  process.exit(1);
}

async function post(content, file) {
  const json = { content, allowed_mentions: { parse: [] } };
  let init;
  if (file) {
    const form = new FormData();
    form.append('payload_json', JSON.stringify({ ...json, attachments: [{ id: 0, filename: file.name }] }));
    form.append('files[0]', new Blob([file.data], { type: 'image/png' }), file.name);
    init = { method: 'POST', body: form };
  } else {
    init = { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(json) };
  }
  const res = await fetch(WEBHOOK_URL, init);
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`Discord webhook ${res.status}: ${body}`);
  }
}

async function sendDiscord(text, file) {
  const parts = chunk(text);
  for (const [i, content] of parts.entries()) {
    await post(content, i === parts.length - 1 ? file : undefined);
  }
}

function attachmentFor(payload) {
  if (payload.type !== 'lifehub_weekly') return undefined;
  const svg = weeklyChartSvg(payload);
  return svg ? { name: 'weekly.png', data: svgToPng(svg) } : undefined;
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
        await sendDiscord(format(payload), attachmentFor(payload));
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
