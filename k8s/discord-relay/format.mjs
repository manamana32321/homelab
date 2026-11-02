const MAX_CONTENT = 2000;

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

export function formatDigest(p) {
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
export function chunk(text) {
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


function md(date) {
  const [, m, d] = date.split('-');
  return `${Number(m)}/${Number(d)}`;
}

function hm(minutes) {
  const t = Math.round(minutes);
  return `${Math.floor(t / 60)}h ${String(t % 60).padStart(2, '0')}m`;
}

function signed(n, unit, fmt = (v) => String(v)) {
  if (n === 0) return `±0${unit}`;
  return `${n > 0 ? '+' : '−'}${fmt(Math.abs(n))}${unit}`;
}

function minutesDelta(m) {
  if (Math.abs(m) < 60) return signed(m, 'm');
  return signed(m, '', (v) => hm(v));
}

function delta(text) {
  return text ? ` (${text})` : '';
}

export function formatLifehubWeekly(p) {
  const lines = [];
  const h = p.habits;
  let header = `📊 **지난주 (${md(p.range.start)} ~ ${md(p.range.end)})**`;
  if (h && h.rate !== null) {
    header += ` · 습관 ${h.rate}%${h.prevRate !== null ? delta(signed(h.rate - h.prevRate, '%p')) : ''}`;
  }
  lines.push(header);

  if (h && h.weeklyOpen.length > 0) lines.push(`남은 주간 습관: ${h.weeklyOpen.map(escapeMd).join(' · ')}`);

  if (p.sleep) {
    const s = p.sleep;
    const d = s.prevAvgMinutes !== null ? delta(minutesDelta(Math.round(s.avgMinutes - s.prevAvgMinutes))) : '';
    lines.push(`😴 수면 평균 ${hm(s.avgMinutes)}${d} · 7시간 이상 ${s.nightsOver7h}/${s.nights}일`);
  }
  if (p.steps) {
    const s = p.steps;
    const n = (v) => Math.round(v).toLocaleString('ko-KR');
    const d = s.prevAvg !== null ? delta(signed(Math.round(s.avg - s.prevAvg), '', n)) : '';
    lines.push(`👟 걸음 평균 ${n(s.avg)}${d}`);
  }
  if (p.screen) {
    const s = p.screen;
    const d = s.prevAvgSeconds !== null ? delta(minutesDelta(Math.round((s.avgSeconds - s.prevAvgSeconds) / 60))) : '';
    lines.push(`📱 스크린타임 하루 평균 ${hm(s.avgSeconds / 60)}${d}`);
  }
  if (p.weight) {
    const w = p.weight;
    if (w.first === w.last) {
      lines.push(`⚖️ 체중 ${w.last.toFixed(1)}kg`);
    } else {
      const diff = Math.round((w.last - w.first) * 10) / 10;
      lines.push(`⚖️ 체중 ${w.first.toFixed(1)} → ${w.last.toFixed(1)}kg (${signed(diff, '', (v) => v.toFixed(1))})`);
    }
  }
  lines.push(`→ 통계 보기${link(p.link)}`);
  return lines.join('\n');
}
