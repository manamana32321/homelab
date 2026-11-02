import { test } from 'node:test';
import assert from 'node:assert/strict';
import { formatLifehubWeekly } from './format.mjs';

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
    daily: [
      { date: '2026-10-25', value: 100 },
      { date: '2026-10-26', value: 50 },
      { date: '2026-10-27', value: null },
      { date: '2026-10-28', value: 0 },
      { date: '2026-10-29', value: 75 },
      { date: '2026-10-30', value: 88 },
      { date: '2026-10-31', value: 63 },
    ],
  },
  sleep: { avgMinutes: 395, prevAvgMinutes: 407, nightsOver7h: 2, nights: 7 },
  steps: { avg: 7820, prevAvg: 7220 },
  screen: {
    avgSeconds: 31200,
    prevAvgSeconds: 29400,
    topApps: ['chrome', 'Code', 'Discord'],
    daily: [
      { date: '2026-10-25', value: 36000 },
      { date: '2026-10-26', value: 18000 },
      { date: '2026-10-27', value: null },
      { date: '2026-10-28', value: 9000 },
      { date: '2026-10-29', value: 27000 },
      { date: '2026-10-30', value: 36000 },
      { date: '2026-10-31', value: 30000 },
    ],
  },
  weight: { first: 75.9, last: 75.5 },
};

test('weekly message is text only: header, best/worst, one-line metrics', () => {
  const text = formatLifehubWeekly(payload);
  const lines = text.split('\n');
  assert.equal(lines[0], '📊 **지난주 (10/25 ~ 10/31)** · 습관 82% (+4%p)');
  assert.ok(!text.includes('```'));
  assert.ok(text.includes('잘함: 금주 7/7 · Workout 5/6'));
  assert.ok(text.includes('약함: 수면 프로토콜 2/7'));
  assert.ok(text.includes('외 14개 · 남은 주간 습관: 가족 연락 · 베갯잇 빨래'));
  assert.ok(text.includes('😴 수면 평균 6h 35m (−12m) · 7시간 이상 2/7일'));
  assert.ok(text.includes('👟 걸음 평균 7,820 (+600)'));
  assert.ok(text.includes('📱 스크린타임 하루 평균 8h 40m (+30m) · chrome · Code · Discord'));
  assert.ok(text.includes('⚖️ 체중 75.9 → 75.5kg (−0.4)'));
  assert.ok(text.includes('<https://example.test/stats>'));
});

test('payload without daily arrays still renders without charts', () => {
  const { daily: _h, ...habits } = payload.habits;
  const { daily: _s, ...screen } = payload.screen;
  const text = formatLifehubWeekly({ ...payload, habits, screen });
  assert.ok(!text.includes('```'));
  assert.ok(text.includes('잘함: 금주 7/7'));
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

test('big minute deltas use h/m, .exe is dropped, single weight has no arrow', () => {
  const text = formatLifehubWeekly({
    ...payload,
    screen: { ...payload.screen, avgSeconds: 17580, prevAvgSeconds: 31980, topApps: ['chrome.exe', 'Code.exe'] },
    sleep: { avgMinutes: 376, prevAvgMinutes: 308, nightsOver7h: 3, nights: 7 },
    weight: { first: 75.1, last: 75.1 },
  });
  assert.ok(text.includes('📱 스크린타임 하루 평균 4h 53m (−4h 00m) · chrome · Code'));
  assert.ok(text.includes('😴 수면 평균 6h 16m (+1h 08m)'));
  assert.ok(text.includes('⚖️ 체중 75.1kg\n'));
});
