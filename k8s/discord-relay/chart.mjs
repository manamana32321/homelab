import { Resvg } from '@resvg/resvg-js';

const FONT_FILE = process.env.CHART_FONT ?? '/usr/share/fonts/dejavu/DejaVuSans.ttf';
const WIDTH = 720;
const PANEL_HEIGHT = 230;
const PAD = { top: 44, right: 24, bottom: 34, left: 24 };
const COLORS = { bg: '#2b2d31', text: '#dbdee1', muted: '#949ba4', grid: '#3f4147' };

const PANELS = [
  { key: 'habits', title: 'Habits (%)', color: '#57f287', max: () => 100, label: (v) => String(Math.round(v)) },
  {
    key: 'screen',
    title: 'Screen time (h)',
    color: '#5865f2',
    max: (vals) => Math.max(...vals),
    label: (v) => (v / 3600).toFixed(1),
  },
];

function escapeXml(s) {
  return String(s).replace(/[<>&"]/g, (c) => ({ '<': '&lt;', '>': '&gt;', '&': '&amp;', '"': '&quot;' })[c]);
}

function md(date) {
  return `${Number(date.slice(5, 7))}/${Number(date.slice(8, 10))}`;
}

function panel(def, days, offsetY) {
  const values = days.map((d) => d.value).filter((v) => v !== null);
  const max = def.max(values);
  const plotW = WIDTH - PAD.left - PAD.right;
  const plotH = PANEL_HEIGHT - PAD.top - PAD.bottom;
  const slot = plotW / days.length;
  const barW = slot * 0.6;
  const baseY = offsetY + PAD.top + plotH;
  const parts = [
    `<text x="${PAD.left}" y="${offsetY + 26}" fill="${COLORS.text}" font-size="16" font-weight="bold">${escapeXml(def.title)}</text>`,
    `<line x1="${PAD.left}" y1="${baseY}" x2="${WIDTH - PAD.right}" y2="${baseY}" stroke="${COLORS.grid}" stroke-width="1"/>`,
  ];
  days.forEach((d, i) => {
    const cx = PAD.left + slot * i + slot / 2;
    parts.push(
      `<text x="${cx}" y="${baseY + 22}" fill="${COLORS.muted}" font-size="13" text-anchor="middle">${md(d.date)}</text>`,
    );
    if (d.value === null) {
      parts.push(`<text x="${cx}" y="${baseY - 6}" fill="${COLORS.muted}" font-size="13" text-anchor="middle">-</text>`);
      return;
    }
    const h = max > 0 ? (Math.min(d.value, max) / max) * plotH : 0;
    if (h > 0) {
      parts.push(
        `<rect class="bar" x="${(cx - barW / 2).toFixed(2)}" y="${(baseY - h).toFixed(2)}" width="${barW.toFixed(2)}" height="${h.toFixed(2)}" rx="3" fill="${def.color}"/>`,
      );
    }
    parts.push(
      `<text x="${cx}" y="${(baseY - h - 6).toFixed(2)}" fill="${COLORS.text}" font-size="13" text-anchor="middle">${escapeXml(def.label(d.value))}</text>`,
    );
  });
  return parts.join('');
}

export function weeklyChartSvg(payload) {
  const series = PANELS.map((def) => ({ def, days: payload[def.key]?.daily ?? [] })).filter(({ days }) =>
    days.some((d) => d.value !== null),
  );
  if (series.length === 0) return null;
  const height = PANEL_HEIGHT * series.length;
  const body = series.map(({ def, days }, i) => panel(def, days, i * PANEL_HEIGHT)).join('');
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${WIDTH}" height="${height}" viewBox="0 0 ${WIDTH} ${height}" font-family="DejaVu Sans"><rect width="100%" height="100%" fill="${COLORS.bg}"/>${body}</svg>`;
}

export function svgToPng(svg) {
  const resvg = new Resvg(svg, {
    fitTo: { mode: 'zoom', value: 2 },
    font: { fontFiles: [FONT_FILE], loadSystemFonts: false, defaultFontFamily: 'DejaVu Sans' },
  });
  return resvg.render().asPng();
}
