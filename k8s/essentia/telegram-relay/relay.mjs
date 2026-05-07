#!/usr/bin/env node
// Tiny webhook → Telegram relay. Receives essentia-api digest payloads at
// POST /notify, formats them as HTML, and forwards to the Telegram bot API.
// Single-purpose, no framework, no deps beyond Node 22's built-in http + fetch.

import http from 'node:http';

const TOKEN = process.env.TELEGRAM_BOT_TOKEN;
const CHAT_ID = process.env.TELEGRAM_CHAT_ID;
const PORT = parseInt(process.env.PORT ?? '8080', 10);

if (!TOKEN || !CHAT_ID) {
  console.error('TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID env vars required');
  process.exit(1);
}

function escapeHtml(s) {
  return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function formatDate(iso, tz) {
  return new Date(iso).toLocaleString('ko-KR', {
    timeZone: tz,
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function formatDayHeader(iso, tz) {
  return new Date(iso).toLocaleDateString('ko-KR', {
    timeZone: tz,
    weekday: 'short',
    month: 'long',
    day: 'numeric',
  });
}

function formatDigest(p) {
  const lines = [];
  lines.push(`<b>🌅 ${escapeHtml(formatDayHeader(p.generatedAt, p.timezone))}</b>`);
  lines.push('');

  const { deadlines, newActivity, suppressed } = p.sections;

  if (deadlines.length > 0) {
    lines.push(`<b>📌 마감 임박 (${deadlines.length})</b>`);
    for (const d of deadlines) {
      const due = escapeHtml(formatDate(d.dueAt, p.timezone));
      const link = d.deeplink ? ` <a href="${escapeHtml(d.deeplink)}">↗</a>` : '';
      lines.push(`• ${due} <i>${escapeHtml(d.course)}</i> — ${escapeHtml(d.title)}${link}`);
    }
    lines.push('');
  }

  if (newActivity.length > 0) {
    lines.push(`<b>📢 새 알림 (${newActivity.length})</b>`);
    for (const a of newActivity) {
      const link = a.deeplink ? ` <a href="${escapeHtml(a.deeplink)}">↗</a>` : '';
      const courseStr = a.course ? `<i>${escapeHtml(a.course)}</i> — ` : '';
      lines.push(`• <code>${escapeHtml(a.category)}</code> ${courseStr}${escapeHtml(a.title)}${link}`);
    }
    lines.push('');
  }

  if (suppressed.count > 0) {
    lines.push(`<i>🔇 LearningX 알림 ${suppressed.count}개 무시 (off 카테고리)</i>`);
  }

  if (deadlines.length === 0 && newActivity.length === 0 && suppressed.count === 0) {
    lines.push('<i>오늘 새 알림 없음.</i>');
  }

  return lines.join('\n');
}

async function sendTelegram(text) {
  const res = await fetch(`https://api.telegram.org/bot${TOKEN}/sendMessage`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      chat_id: CHAT_ID,
      text,
      parse_mode: 'HTML',
      disable_web_page_preview: true,
    }),
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`Telegram API ${res.status}: ${body}`);
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
        if (payload.type !== 'daily_digest') {
          res.writeHead(400);
          res.end(`unsupported type: ${payload.type ?? '(missing)'}`);
          return;
        }
        const text = formatDigest(payload);
        await sendTelegram(text);
        const d = payload.sections.deadlines.length;
        const a = payload.sections.newActivity.length;
        const s = payload.sections.suppressed.count;
        console.log(`relay ok user=${payload.userId} deadlines=${d} activity=${a} suppressed=${s}`);
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
  console.log(`essentia-telegram-relay listening on :${PORT}`);
});
