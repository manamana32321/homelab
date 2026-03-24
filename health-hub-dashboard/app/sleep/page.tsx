"use client";

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
  Legend,
} from "recharts";
import { useSleep } from "@/lib/hooks/use-sleep";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { daysAgo, today, formatDate, formatDuration } from "@/lib/format";

const STAGE_COLORS: Record<string, string> = {
  deep: "#6366f1",
  light: "#a5b4fc",
  rem: "#f59e0b",
  awake: "#ef4444",
};

const STAGE_LABELS: Record<string, string> = {
  deep: "깊은잠",
  light: "얕은잠",
  rem: "REM",
  awake: "깨어남",
};

function normStage(s: string): string {
  if (s === "3" || s === "deep" || s === "90") return "deep";
  if (s === "2" || s === "light" || s === "40" || s === "4") return "light";
  if (s === "5" || s === "rem" || s === "100") return "rem";
  return "awake";
}

export default function SleepPage() {
  const { data, isLoading } = useSleep(daysAgo(30), today());

  const chartData = (data ?? [])
    .slice()
    .reverse()
    .map((s) => {
      const stageSums: Record<string, number> = {
        deep: 0,
        light: 0,
        rem: 0,
        awake: 0,
      };
      for (const st of s.stages ?? []) {
        const key = normStage(st.stage);
        stageSums[key] += st.duration_m;
      }
      return {
        date: formatDate(s.start_time),
        total: s.duration_m,
        ...stageSums,
      };
    });

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold">수면</h2>

      {/* Duration trend */}
      <Card>
        <CardContent className="p-4">
          <h3 className="text-sm font-medium text-muted-foreground mb-4">
            수면 시간 (최근 30일)
          </h3>
          {isLoading ? (
            <Skeleton className="h-64 w-full" />
          ) : (
            <ResponsiveContainer width="100%" height={280}>
              <BarChart data={chartData}>
                <CartesianGrid
                  strokeDasharray="3 3"
                  className="stroke-border"
                />
                <XAxis
                  dataKey="date"
                  tick={{ fontSize: 11 }}
                  className="fill-muted-foreground"
                />
                <YAxis
                  tick={{ fontSize: 11 }}
                  className="fill-muted-foreground"
                  tickFormatter={(v) => `${(v / 60).toFixed(0)}h`}
                />
                <Tooltip
                  contentStyle={{
                    backgroundColor: "hsl(var(--card))",
                    border: "1px solid hsl(var(--border))",
                    borderRadius: 8,
                    fontSize: 12,
                  }}
                  formatter={(v, name) => [
                    formatDuration(Number(v)),
                    STAGE_LABELS[String(name)] ?? String(name),
                  ]}
                />
                <Legend
                  formatter={(value) => STAGE_LABELS[value] ?? value}
                />
                <Bar
                  dataKey="deep"
                  stackId="sleep"
                  fill={STAGE_COLORS.deep}
                />
                <Bar
                  dataKey="light"
                  stackId="sleep"
                  fill={STAGE_COLORS.light}
                />
                <Bar
                  dataKey="rem"
                  stackId="sleep"
                  fill={STAGE_COLORS.rem}
                />
                <Bar
                  dataKey="awake"
                  stackId="sleep"
                  fill={STAGE_COLORS.awake}
                  radius={[4, 4, 0, 0]}
                />
              </BarChart>
            </ResponsiveContainer>
          )}
        </CardContent>
      </Card>

      {/* Session list */}
      <Card>
        <CardContent className="p-4">
          <h3 className="text-sm font-medium text-muted-foreground mb-3">
            수면 기록
          </h3>
          <div className="space-y-2">
            {(data ?? []).slice(0, 7).map((s) => (
              <div
                key={s.id}
                className="flex items-center justify-between rounded-lg border border-border p-3"
              >
                <span className="text-sm">{formatDate(s.start_time)}</span>
                <span className="text-sm font-medium">
                  {formatDuration(s.duration_m)}
                </span>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
