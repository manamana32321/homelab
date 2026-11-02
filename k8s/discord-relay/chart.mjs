import { Resvg } from '@resvg/resvg-js';

const FONT_FILE = process.env.CHART_FONT ?? '/usr/share/fonts/chart/NotoSansKR-subset.otf';
const FONT_FAMILY = 'Noto Sans CJK KR';
const WIDTH = 720;
const PAD = 24;
const COLORS = { bg: '#2b2d31', text: '#dbdee1', muted: '#949ba4', empty: '#313338', line: '#3f4147' };
const STATUS = {
  O: { fill: '#059669', mark: 'O' },
  X: { fill: '#dc2626', mark: 'X' },
  skip: { fill: '#71717a', mark: '-' },
  none: { fill: COLORS.empty, mark: '' },
};
const SLICE_COLORS = ['#5865f2', '#57f287', '#fee75c', '#eb459e', '#ed4245', '#3ba55d'];
const OTHER_COLOR = '#6d6f78';
const WEEKDAYS = ['일', '월', '화', '수', '목', '금', '토'];

const HEAT = { top: 34, header: 44, rowH: 26, cellH: 20, nameW: 210 };
const DONUT = { top: 34, r: 92, width: 34 };

function escapeXml(s) {
  return String(s).replace(/[<>&"]/g, (c) => ({ '<': '&lt;', '>': '&gt;', '&': '&amp;', '"': '&quot;' })[c]);
}

function text(x, y, value, { size = 13, fill = COLORS.text, anchor = 'start', weight = 'normal' } = {}) {
  return `<text x="${x}" y="${y}" fill="${fill}" font-size="${size}" font-weight="${weight}" text-anchor="${anchor}">${escapeXml(value)}</text>`;
}

function approxWidth(s, size) {
  let w = 0;
  for (const ch of s) w += ch.codePointAt(0) > 0x2e80 ? size : size * 0.58;
  return w;
}

function fit(s, size, max) {
  if (approxWidth(s, size) <= max) return s;
  let out = '';
  for (const ch of s) {
    if (approxWidth(`${out + ch}…`, size) > max) break;
    out += ch;
  }
  return `${out}…`;
}

function hm(seconds) {
  const m = Math.round(seconds / 60);
  return `${Math.floor(m / 60)}h ${String(m % 60).padStart(2, '0')}m`;
}

function days(range) {
  const out = [];
  const d = new Date(`${range.start}T00:00:00Z`);
  const end = new Date(`${range.end}T00:00:00Z`);
  while (d <= end) {
    out.push({ md: `${d.getUTCMonth() + 1}/${d.getUTCDate()}`, wd: WEEKDAYS[d.getUTCDay()] });
    d.setUTCDate(d.getUTCDate() + 1);
  }
  return out;
}

function heatmap(habits, range, offsetY) {
  const cols = days(range);
  const slot = (WIDTH - PAD * 2 - HEAT.nameW) / cols.length;
  const cellW = slot - 8;
  const x0 = PAD + HEAT.nameW;
  const title = habits.rate === null || habits.rate === undefined ? '습관' : `습관 · ${habits.rate}%`;
  const parts = [text(PAD, offsetY + 24, title, { size: 17, weight: 'bold' })];
  cols.forEach((c, i) => {
    const cx = x0 + slot * i + slot / 2;
    parts.push(text(cx, offsetY + HEAT.top + 16, c.wd, { anchor: 'middle', fill: COLORS.muted }));
    parts.push(text(cx, offsetY + HEAT.top + 32, c.md, { size: 11, anchor: 'middle', fill: COLORS.muted }));
  });
  habits.grid.forEach((row, r) => {
    const y = offsetY + HEAT.top + HEAT.header + r * HEAT.rowH;
    parts.push(text(PAD, y + 15, fit(row.name, 14, HEAT.nameW - 12), { size: 14 }));
    row.cells.forEach((cell, i) => {
      const key = cell ?? 'none';
      const look = STATUS[key];
      const x = x0 + slot * i + (slot - cellW) / 2;
      parts.push(
        `<rect class="cell" data-s="${key}" x="${x.toFixed(1)}" y="${y}" width="${cellW.toFixed(1)}" height="${HEAT.cellH}" rx="4" fill="${look.fill}"${key === 'none' ? ` stroke="${COLORS.line}"` : ''}/>`,
      );
      if (look.mark) parts.push(text(x + cellW / 2, y + 15, look.mark, { size: 12, anchor: 'middle', fill: '#ffffff', weight: 'bold' }));
    });
  });
  return { svg: parts.join(''), height: HEAT.top + HEAT.header + habits.grid.length * HEAT.rowH + 16 };
}

function donut(screen, offsetY) {
  const slices = screen.apps.map((a, i) => ({ name: a.name.replace(/\.exe$/i, ''), seconds: a.seconds, color: SLICE_COLORS[i % SLICE_COLORS.length] }));
  if (screen.otherSeconds > 0) slices.push({ name: '기타', seconds: screen.otherSeconds, color: OTHER_COLOR });
  const total = slices.reduce((s, x) => s + x.seconds, 0);
  const cx = PAD + DONUT.r + DONUT.width / 2 + 8;
  const cy = offsetY + DONUT.top + 18 + DONUT.r + DONUT.width / 2;
  const circ = 2 * Math.PI * DONUT.r;
  const parts = [text(PAD, offsetY + 24, '스크린타임 · 앱별', { size: 17, weight: 'bold' })];
  let start = 0;
  for (const s of slices) {
    const len = (s.seconds / total) * circ;
    parts.push(
      `<circle class="slice" cx="${cx}" cy="${cy}" r="${DONUT.r}" fill="none" stroke="${s.color}" stroke-width="${DONUT.width}" stroke-dasharray="${len.toFixed(2)} ${(circ - len).toFixed(2)}" stroke-dashoffset="${(-start).toFixed(2)}" transform="rotate(-90 ${cx} ${cy})"/>`,
    );
    start += len;
  }
  parts.push(text(cx, cy - 2, hm(total), { size: 18, anchor: 'middle', weight: 'bold' }));
  parts.push(text(cx, cy + 18, '7일 합계', { size: 12, anchor: 'middle', fill: COLORS.muted }));
  const lx = cx + DONUT.r + DONUT.width / 2 + 40;
  const ly = offsetY + DONUT.top + 40;
  slices.forEach((s, i) => {
    const y = ly + i * 28;
    const share = `${Math.round((s.seconds / total) * 100)}%`;
    parts.push(`<rect x="${lx}" y="${y - 12}" width="14" height="14" rx="3" fill="${s.color}"/>`);
    parts.push(text(lx + 24, y, fit(s.name, 14, 170), { size: 14 }));
    parts.push(text(WIDTH - PAD, y, `${hm(s.seconds)} · ${share}`, { size: 13, anchor: 'end', fill: COLORS.muted }));
  });
  const ringBottom = DONUT.top + 18 + DONUT.r * 2 + DONUT.width;
  const legendBottom = DONUT.top + 40 + (slices.length - 1) * 28 + 8;
  return { svg: parts.join(''), height: Math.max(ringBottom, legendBottom) + 24 };
}

export function weeklyChartSvg(payload) {
  const panels = [];
  if (payload.habits?.grid?.length > 0) panels.push((y) => heatmap(payload.habits, payload.range, y));
  const screen = payload.screen;
  if (screen && (screen.apps?.length > 0 || screen.otherSeconds > 0)) panels.push((y) => donut(screen, y));
  if (panels.length === 0) return null;
  let y = 0;
  const body = [];
  for (const draw of panels) {
    const { svg, height } = draw(y);
    body.push(svg);
    y += height;
  }
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${WIDTH}" height="${y}" viewBox="0 0 ${WIDTH} ${y}" font-family="${FONT_FAMILY}"><rect width="100%" height="100%" fill="${COLORS.bg}"/>${body.join('')}</svg>`;
}

export function svgToPng(svg) {
  const resvg = new Resvg(svg, {
    fitTo: { mode: 'zoom', value: 2 },
    font: { fontFiles: [FONT_FILE], loadSystemFonts: false, defaultFontFamily: FONT_FAMILY },
  });
  return resvg.render().asPng();
}
