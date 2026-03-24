"use client";

import { useMemo } from "react";
import { format, subDays, getDay, startOfWeek, addDays } from "date-fns";
import { ko } from "date-fns/locale";
import type { AggregatedMetric } from "@/lib/types";

interface StepsHeatmapProps {
  data: AggregatedMetric[];
  days?: number;
}

const CELL_SIZE = 13;
const GAP = 3;
const LEVELS = [0, 2000, 5000, 8000, 12000];
const COLORS = [
  "var(--color-muted)",
  "#0e4429",
  "#006d32",
  "#26a641",
  "#39d353",
];

function getLevel(steps: number): number {
  for (let i = LEVELS.length - 1; i >= 0; i--) {
    if (steps >= LEVELS[i]) return i;
  }
  return 0;
}

export function StepsHeatmap({ data, days = 365 }: StepsHeatmapProps) {
  const { grid, months, maxSteps } = useMemo(() => {
    const map = new Map<string, number>();
    for (const d of data) {
      const key = format(new Date(d.time), "yyyy-MM-dd");
      map.set(key, (map.get(key) ?? 0) + d.sum);
    }

    const today = new Date();
    const start = subDays(today, days - 1);
    const weekStart = startOfWeek(start, { weekStartsOn: 0 });

    const cells: {
      date: string;
      steps: number;
      col: number;
      row: number;
    }[] = [];
    const monthLabels: { label: string; col: number }[] = [];
    let prevMonth = -1;
    let maxS = 0;

    let current = weekStart;
    while (current <= today) {
      const dateStr = format(current, "yyyy-MM-dd");
      const dayOfWeek = getDay(current);
      const weekNum = Math.floor(
        (current.getTime() - weekStart.getTime()) / (7 * 24 * 60 * 60 * 1000),
      );

      const steps = map.get(dateStr) ?? 0;
      if (steps > maxS) maxS = steps;

      if (current.getMonth() !== prevMonth) {
        monthLabels.push({
          label: format(current, "MMM", { locale: ko }),
          col: weekNum,
        });
        prevMonth = current.getMonth();
      }

      if (current >= start) {
        cells.push({ date: dateStr, steps, col: weekNum, row: dayOfWeek });
      }

      current = addDays(current, 1);
    }

    return { grid: cells, months: monthLabels, maxSteps: maxS };
  }, [data, days]);

  const totalWeeks = grid.length > 0 ? grid[grid.length - 1].col + 1 : 53;
  const svgWidth = totalWeeks * (CELL_SIZE + GAP) + 30;
  const svgHeight = 7 * (CELL_SIZE + GAP) + 30;

  return (
    <div className="overflow-x-auto">
      <svg width={svgWidth} height={svgHeight} className="block">
        {/* Month labels */}
        {months.map((m, i) => (
          <text
            key={i}
            x={m.col * (CELL_SIZE + GAP) + 30}
            y={10}
            className="fill-muted-foreground"
            fontSize={10}
          >
            {m.label}
          </text>
        ))}

        {/* Day labels */}
        {["일", "", "화", "", "목", "", "토"].map((d, i) =>
          d ? (
            <text
              key={i}
              x={0}
              y={i * (CELL_SIZE + GAP) + 20 + CELL_SIZE - 2}
              className="fill-muted-foreground"
              fontSize={9}
            >
              {d}
            </text>
          ) : null,
        )}

        {/* Cells */}
        {grid.map((cell) => (
          <rect
            key={cell.date}
            x={cell.col * (CELL_SIZE + GAP) + 30}
            y={cell.row * (CELL_SIZE + GAP) + 20}
            width={CELL_SIZE}
            height={CELL_SIZE}
            rx={2}
            fill={COLORS[getLevel(cell.steps)]}
            className="transition-colors"
          >
            <title>
              {format(new Date(cell.date), "M월 d일", { locale: ko })}:{" "}
              {cell.steps.toLocaleString()}걸음
            </title>
          </rect>
        ))}
      </svg>

      {/* Legend */}
      <div className="flex items-center gap-1 mt-2 text-xs text-muted-foreground">
        <span>적음</span>
        {COLORS.map((c, i) => (
          <div
            key={i}
            className="w-3 h-3 rounded-sm"
            style={{ backgroundColor: c }}
          />
        ))}
        <span>많음</span>
        <span className="ml-4">
          최대: {maxSteps.toLocaleString()}걸음
        </span>
      </div>
    </div>
  );
}
