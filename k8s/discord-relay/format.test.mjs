import { test } from 'node:test';
import assert from 'node:assert/strict';
import { bar, displayWidth, formatLifehubWeekly } from './format.mjs';

const payload = {
  type: 'lifehub_weekly',
  generatedAt: '2026-11-01T00:00:00Z',
  timezone: 'Asia/Seoul',
  range: { start: '2026-10-25', end: '2026-10-31' },
  link: 'https://example.test/stats',
  habits: {
    rate: 82,
    prevRate: 78,
    best: [
      { name: '금주', o: 7, total: 7 },
      { name: 'Workout', o: 5, total: 6 },
    ],
    worst: [{ name: '수면 프로토콜', o: 2, total: 7 }],
    others: 14,
    weeklyOpen: ['가족 연락', '베갯잇 빨래'],
  },
  sleep: { avgMinutes: 395, prevAvgMinutes: 407, nightsOver7h: 2, nights: 7 },
  steps: { avg: 7820, prevAvg: 7220 },
  screen: { avgSeconds: 31200, prevAvgSeconds: 29400, topApps: ['chrome', 'Code', 'Discord'] },
  weight: { first: 75.9, last: 75.5 },
};

test('displayWidth counts Hangul as 2 columns', () => {
  assert.equal(displayWidth('abc'), 3);
  assert.equal(displayWidth('금주'), 4);
  assert.equal(displayWidth('수면 프로토콜'), 13);
});

test('bar is always 10 columns with 1/8 partials', () => {
  assert.equal(bar(7, 7), '██████████');
  assert.equal(bar(0, 7), '          ');
  assert.equal(bar(5, 6), '████████▍ ');
  assert.equal(displayWidth(bar(2, 7)), 10);
});

test('weekly message has header, aligned habit bars, and one-line metrics', () => {
  const text = formatLifehubWeekly(payload);
  const lines = text.split('\n');
  assert.equal(lines[0], '📊 **지난주 (10/25 ~ 10/31)** · 습관 82% (+4%p)');
  const block = lines.slice(lines.indexOf('```') + 1, lines.lastIndexOf('```'));
  const rows = block.filter((l) => l !== '···');
  const barEnds = rows.map((l) => displayWidth(l.replace(/ \d+\/\d+$/, '')));
  assert.equal(new Set(barEnds).size, 1);
  assert.ok(block.includes('···'));
  assert.ok(block[0].startsWith('금주'));
  assert.ok(block[0].endsWith('7/7'));
  assert.ok(text.includes('외 14개 · 남은 주간 습관: 가족 연락 · 베갯잇 빨래'));
  assert.ok(text.includes('😴 수면 평균 6h 35m (−12m) · 7시간 이상 2/7일'));
  assert.ok(text.includes('👟 걸음 평균 7,820 (+600)'));
  assert.ok(text.includes('📱 스크린타임 하루 평균 8h 40m (+30m) · chrome · Code · Discord'));
  assert.ok(text.includes('⚖️ 체중 75.9 → 75.5kg (−0.4)'));
  assert.ok(text.includes('<https://example.test/stats>'));
});

test('null sections are omitted, no fake zeros', () => {
  const text = formatLifehubWeekly({ ...payload, habits: null, sleep: null, steps: null, screen: null, weight: null });
  assert.equal(text.split('\n')[0], '📊 **지난주 (10/25 ~ 10/31)**');
  assert.ok(!text.includes('```'));
  assert.ok(!text.includes('수면'));
  assert.ok(!text.includes('체중'));
});

test('missing previous values drop the delta', () => {
  const text = formatLifehubWeekly({
    ...payload,
    habits: { ...payload.habits, prevRate: null },
    steps: { avg: 7820, prevAvg: null },
  });
  assert.ok(text.split('\n')[0].endsWith('습관 82%'));
  assert.ok(text.includes('👟 걸음 평균 7,820\n') || text.endsWith('👟 걸음 평균 7,820'));
});
