"use client";

import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
  Area,
  AreaChart,
} from "recharts";
import { useMetrics } from "@/lib/hooks/use-metrics";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { daysAgo, today, formatDate, formatTime } from "@/lib/format";
import { parseAsString, useQueryState } from "nuqs";

export default function HeartRatePage() {
  const [range, setRange] = useQueryState(
    "range",
    parseAsString.withDefault("today"),
  );

  const isToday = range === "today";
  const from = isToday ? today() : daysAgo(range === "7d" ? 7 : 30);
  const to = isToday ? daysAgo(-1) : today();
  const interval = isToday ? "15m" : "1d";

  const { data, isLoading } = useMetrics("heart_rate", from, to, interval);

  const chartData = (data ?? []).map((d) => ({
    time: isToday ? formatTime(d.time) : formatDate(d.time),
    avg: Math.round(d.avg),
    min: Math.round(d.min),
    max: Math.round(d.max),
  }));

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold">심박수</h2>

      <Card>
        <CardContent className="p-4">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-medium text-muted-foreground">
              {isToday ? "오늘 심박수" : "평균 심박수 추이"}
            </h3>
            <Tabs value={range} onValueChange={(v) => setRange(v)}>
              <TabsList>
                <TabsTrigger value="today">오늘</TabsTrigger>
                <TabsTrigger value="7d">7일</TabsTrigger>
                <TabsTrigger value="30d">30일</TabsTrigger>
              </TabsList>
            </Tabs>
          </div>

          {isLoading ? (
            <Skeleton className="h-64 w-full" />
          ) : (
            <ResponsiveContainer width="100%" height={300}>
              <AreaChart data={chartData}>
                <CartesianGrid
                  strokeDasharray="3 3"
                  className="stroke-border"
                />
                <XAxis
                  dataKey="time"
                  tick={{ fontSize: 11 }}
                  className="fill-muted-foreground"
                />
                <YAxis
                  domain={["dataMin - 10", "dataMax + 10"]}
                  tick={{ fontSize: 11 }}
                  className="fill-muted-foreground"
                />
                <Tooltip
                  contentStyle={{
                    backgroundColor: "hsl(var(--card))",
                    border: "1px solid hsl(var(--border))",
                    borderRadius: 8,
                    fontSize: 12,
                  }}
                  formatter={(v, name) => {
                    const labels: Record<string, string> = {
                      avg: "평균",
                      min: "최소",
                      max: "최대",
                    };
                    return [`${Number(v)} bpm`, labels[String(name)] ?? name];
                  }}
                />
                <Area
                  type="monotone"
                  dataKey="max"
                  stroke="none"
                  fill="#ef4444"
                  fillOpacity={0.1}
                />
                <Area
                  type="monotone"
                  dataKey="min"
                  stroke="none"
                  fill="hsl(var(--background))"
                  fillOpacity={1}
                />
                <Line
                  type="monotone"
                  dataKey="avg"
                  stroke="#ef4444"
                  strokeWidth={2}
                  dot={false}
                />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
