"use client";

import { useMetrics } from "@/lib/hooks/use-metrics";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { daysAgo, today, formatDate, formatNumber } from "@/lib/format";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
} from "recharts";

export default function NutritionPage() {
  const { data, isLoading } = useMetrics("calories", daysAgo(30), today(), "1d");

  const chartData = (data ?? []).map((d) => ({
    date: formatDate(d.time),
    calories: Math.round(d.sum),
  }));

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold">영양</h2>

      <Card>
        <CardContent className="p-4">
          <h3 className="text-sm font-medium text-muted-foreground mb-4">
            일별 소모 칼로리 (최근 30일)
          </h3>
          {isLoading ? (
            <Skeleton className="h-64 w-full" />
          ) : chartData.length === 0 ? (
            <p className="text-center text-muted-foreground py-8">
              데이터 없음
            </p>
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
                />
                <Tooltip
                  contentStyle={{
                    backgroundColor: "hsl(var(--card))",
                    border: "1px solid hsl(var(--border))",
                    borderRadius: 8,
                    fontSize: 12,
                  }}
                  formatter={(v) => [
                    `${formatNumber(Number(v))} kcal`,
                    "칼로리",
                  ]}
                />
                <Bar
                  dataKey="calories"
                  fill="#f97316"
                  radius={[4, 4, 0, 0]}
                />
              </BarChart>
            </ResponsiveContainer>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
