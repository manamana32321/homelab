import { test } from 'node:test';
import assert from 'node:assert/strict';
import { svgToPng, weeklyChartSvg } from './chart.mjs';

const range = { start: '2026-10-25', end: '2026-10-31' };
const habits = {
  rate: 57,
  grid: [
    { name: '금주', cells: ['O', 'O', 'X', 'skip', null, 'O', 'O'] },
    { name: 'Workout', cells: [null, null, null, null, null, null, null] },
  ],
};
const screen = {
  apps: [
    { name: 'chrome.exe', seconds: 7200 },
    { name: 'Code.exe', seconds: 3600 },
  ],
  otherSeconds: 3600,
};

test('heatmap: one row per habit, 7 cells each, colored by status', () => {
  const svg = weeklyChartSvg({ range, habits, screen: null });
  assert.equal((svg.match(/class="cell"/g) ?? []).length, 14);
  assert.equal((svg.match(/class="cell" data-s="O"/g) ?? []).length, 4);
  assert.equal((svg.match(/class="cell" data-s="X"/g) ?? []).length, 1);
  assert.equal((svg.match(/class="cell" data-s="skip"/g) ?? []).length, 1);
  assert.equal((svg.match(/class="cell" data-s="none"/g) ?? []).length, 8);
  assert.ok(svg.includes('>금주<'));
  assert.ok(svg.includes('>10/25<'));
  assert.ok(svg.includes('>일<'));
});

test('donut: one slice per app plus 기타, legend strips .exe with share', () => {
  const svg = weeklyChartSvg({ range, habits: null, screen });
  assert.equal((svg.match(/class="slice"/g) ?? []).length, 3);
  assert.ok(svg.includes('>chrome<'));
  assert.ok(svg.includes('>Code<'));
  assert.ok(svg.includes('>기타<'));
  assert.ok(svg.includes('50%'));
  assert.ok(svg.includes('25%'));
  assert.ok(!svg.includes('.exe'));
});

test('a single app makes a full ring', () => {
  const svg = weeklyChartSvg({ range, habits: null, screen: { apps: [{ name: 'a', seconds: 10 }], otherSeconds: 0 } });
  assert.equal((svg.match(/class="slice"/g) ?? []).length, 1);
  assert.ok(svg.includes('100%'));
});

test('returns null when there is nothing to draw', () => {
  assert.equal(weeklyChartSvg({ range, habits: null, screen: null }), null);
  assert.equal(weeklyChartSvg({ range, habits: { grid: [] }, screen: { apps: [], otherSeconds: 0 } }), null);
});

test('svgToPng produces a PNG', () => {
  const png = svgToPng(weeklyChartSvg({ range, habits, screen }));
  assert.deepEqual([...png.subarray(0, 8)], [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
});
