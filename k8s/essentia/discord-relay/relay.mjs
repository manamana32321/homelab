#!/usr/bin/env node
// Tiny webhook → Discord relay. Receives essentia-api digest payloads at
// POST /notify, formats them as Discord markdown, and forwards to a Discord
// channel webhook. Single-purpose, no framework, no deps beyond Node 22's
// built-in http + fetch.

import http from 'node:http';

const WEBHOOK_URL = process.env.DISCORD_WEBHOOK_URL;
const PORT = parseInt(process.env.PORT ?? '8080', 10);

// 디스코드 메시지 상한. 넘으면 줄 경계로 쪼개 보낸다.
const MAX_CONTENT = 2000;

if (!WEBHOOK_URL) {
  console.error('DISCORD_WEBHOOK_URL env var required');
  process.exit(1);
}

function escapeMd(s) {
  return String(s).replace(/([\\`*_~|>])/g, '\\$1');
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

// 링크는 <url> 로 감싸 임베드 미리보기를 막는다 (텔레그램의
// disable_web_page_preview 와 같은 효과).
function link(url) {
  return url ? ` [↗](<${url}>)` : '';
}

function formatDigest(p) {
  const lines = [];
  lines.push(`**🌅 ${escapeMd(formatDayHeader(p.generatedAt, p.timezone))}**`);
  lines.push('');

  const { deadlines, newActivity, suppressed } = p.sections;

  if (deadlines.length > 0) {
    lines.push(`**📌 마감 임박 (${deadlines.length})**`);
    for (const d of deadlines) {
      const due = escapeMd(formatDate(d.dueAt, p.timezone));
      lines.push(`• ${due} *${escapeMd(d.course)}* — ${escapeMd(d.title)}${link(d.deeplink)}`);
    }
    lines.push('');
  }

  if (newActivity.length > 0) {
    lines.push(`**📢 새 알림 (${newActivity.length})**`);
    for (const a of newActivity) {
      const courseStr = a.course ? `*${escapeMd(a.course)}* — ` : '';
      lines.push(`• \`${escapeMd(a.category)}\` ${courseStr}${escapeMd(a.title)}${link(a.deeplink)}`);
    }
    lines.push('');
  }

  if (suppressed.count > 0) {
    lines.push(`*🔇 LearningX 알림 ${suppressed.count}개 무시 (off 카테고리)*`);
  }

  if (deadlines.length === 0 && newActivity.length === 0 && suppressed.count === 0) {
    lines.push('*오늘 새 알림 없음.*');
  }

  return lines.join('\n');
}

// 2000자를 넘으면 줄 단위로 나눈다. 한 줄이 그 자체로 상한을 넘으면 잘라낸다.
function chunk(text) {
  const out = [];
  let cur = '';
  for (const line of text.split('\n')) {
    const piece = line.length > MAX_CONTENT ? line.slice(0, MAX_CONTENT) : line;
    if (cur.length + piece.length + 1 > MAX_CONTENT) {
      if (cur) out.push(cur);
      cur = piece;
    } else {
      cur = cur ? `${cur}\n${piece}` : piece;
    }
  }
  if (cur) out.push(cur);
  return out;
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
        if (payload.type !== 'daily_digest') {
          res.writeHead(400);
          res.end(`unsupported type: ${payload.type ?? '(missing)'}`);
          return;
        }
        const text = formatDigest(payload);
        await sendDiscord(text);
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
  console.log(`essentia-discord-relay listening on :${PORT}`);
});
