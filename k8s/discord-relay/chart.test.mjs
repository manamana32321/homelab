import { test } from 'node:test';
import assert from 'node:assert/strict';
import { svgToPng, weeklyChartSvg } from './chart.mjs';

const days = (values) => values.map((value, i) => ({ date: `2026-10-${25 + i}`, value }));

const habits = { daily: days([100, 50, null, 0, 75, 88, 63]) };
const screen = { daily: days([36000, 18000, null, 9000, 27000, 36000, 30000]) };

test('one panel per available series with 7 bars, nulls get no bar', () => {
  const svg = weeklyChartSvg({ habits, screen });
  assert.equal((svg.match(/class="bar"/g) ?? []).length, 11);
  assert.ok(svg.includes('Habits (%)'));
  assert.ok(svg.includes('Screen time (h)'));
  assert.ok(svg.includes('>10/25<'));
  assert.ok(svg.includes('>100<'));
  assert.ok(svg.includes('>10.0<'));
});

test('bar heights are proportional within a panel', () => {
  const svg = weeklyChartSvg({ habits: { daily: days([100, 50, 0, 0, 0, 0, 0]) }, screen: null });
  const heights = [...svg.matchAll(/class="bar"[^>]*height="([\d.]+)"/g)].map((m) => Number(m[1]));
  assert.equal(heights.length, 2);
  assert.ok(Math.abs(heights[0] - 2 * heights[1]) < 0.01);
});

test('returns null when there is nothing to draw', () => {
  assert.equal(weeklyChartSvg({ habits: null, screen: null }), null);
  assert.equal(weeklyChartSvg({ habits: { daily: days([null, null, null, null, null, null, null]) }, screen: {} }), null);
});

test('svgToPng produces a PNG', () => {
  const png = svgToPng(weeklyChartSvg({ habits, screen }));
  assert.deepEqual([...png.subarray(0, 8)], [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
});
