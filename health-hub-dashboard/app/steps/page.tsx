"use client";

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
} from "recharts";
import { useMetrics } from "@/lib/hooks/use-metrics";
import { StepsHeatmap } from "@/components/charts/steps-heatmap";
import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Skeleton } from "@/components/ui/skeleton";
import { daysAgo, today, formatDate, formatNumber } from "@/lib/format";
import { parseAsString, useQueryState } from "nuqs";

const intervals: Record<string, { from: string; interval: string }> = {
  "7d": { from: daysAgo(7), interval: "1d" },
  "30d": { from: daysAgo(30), interval: "1d" },
  "90d": { from: daysAgo(90), interval: "1d" },
};

export default function StepsPage() {
  const [range, setRange] = useQueryState(
    "range",
    parseAsString.withDefault("30d"),
  );

  const cfg = intervals[range] ?? intervals["30d"];
  const { data: barData, isLoading: barLoading } = useMetrics(
    "steps",
    cfg.from,
    today(),
    cfg.interval,
  );
  const { data: heatData, isLoading: heatLoading } = useMetrics(
    "steps",
    daysAgo(365),
    today(),
    "1d",
  );

  const chartData = (barData ?? []).map((d) => ({
    date: formatDate(d.time),
    steps: Math.round(d.sum),
  }));

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold">걸음수</h2>

      {/* Heatmap */}
      <Card>
        <CardContent className="p-4">
          <h3 className="text-sm font-medium text-muted-foreground mb-3">
            연간 걸음수
          </h3>
          {heatLoading ? (
            <Skeleton className="h-32 w-full" />
          ) : (
            <StepsHeatmap data={heatData ?? []} />
          )}
        </CardContent>
      </Card>

      {/* Bar chart */}
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-medium text-muted-foreground">
              일별 걸음수
            </h3>
            <Tabs value={range} onValueChange={(v) => setRange(v)}>
              <TabsList>
                <TabsTrigger value="7d">7일</TabsTrigger>
                <TabsTrigger value="30d">30일</TabsTrigger>
                <TabsTrigger value="90d">90일</TabsTrigger>
              </TabsList>
            </Tabs>
          </div>

          {barLoading ? (
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
                  tickFormatter={(v) => formatNumber(Number(v))}
                />
                <Tooltip
                  contentStyle={{
                    backgroundColor: "hsl(var(--card))",
                    border: "1px solid hsl(var(--border))",
                    borderRadius: 8,
                    fontSize: 12,
                  }}
                  formatter={(v) => [
                    `${formatNumber(Number(v))} 걸음`,
                    "걸음수",
                  ]}
                />
                <Bar dataKey="steps" fill="#39d353" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
