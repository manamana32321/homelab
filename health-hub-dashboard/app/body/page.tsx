"use client";

import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
} from "recharts";
import { useBody } from "@/lib/hooks/use-body";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { daysAgo, today, formatDate, formatNumber } from "@/lib/format";

export default function BodyPage() {
  const { data, isLoading } = useBody(daysAgo(90), today());

  const chartData = (data ?? [])
    .slice()
    .reverse()
    .map((d) => ({
      date: formatDate(d.time),
      weight: d.weight_kg,
      fat: d.body_fat_pct,
    }));

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold">체성분</h2>

      {isLoading ? (
        <Skeleton className="h-80 w-full" />
      ) : chartData.length === 0 ? (
        <Card>
          <CardContent className="p-8 text-center text-muted-foreground">
            최근 90일 체성분 기록이 없습니다
          </CardContent>
        </Card>
      ) : (
        <>
          <Card>
            <CardContent className="p-4">
              <h3 className="text-sm font-medium text-muted-foreground mb-4">
                체중 추이
              </h3>
              <ResponsiveContainer width="100%" height={280}>
                <LineChart data={chartData}>
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
                    domain={["dataMin - 2", "dataMax + 2"]}
                    tick={{ fontSize: 11 }}
                    className="fill-muted-foreground"
                    unit="kg"
                  />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: "hsl(var(--card))",
                      border: "1px solid hsl(var(--border))",
                      borderRadius: 8,
                      fontSize: 12,
                    }}
                    formatter={(v) => [
                      `${formatNumber(Number(v), 1)} kg`,
                      "체중",
                    ]}
                  />
                  <Line
                    type="monotone"
                    dataKey="weight"
                    stroke="#a855f7"
                    strokeWidth={2}
                    dot={{ r: 3 }}
                    connectNulls
                  />
                </LineChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          {chartData.some((d) => d.fat != null) && (
            <Card>
              <CardContent className="p-4">
                <h3 className="text-sm font-medium text-muted-foreground mb-4">
                  체지방률 추이
                </h3>
                <ResponsiveContainer width="100%" height={200}>
                  <LineChart data={chartData}>
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
                      unit="%"
                    />
                    <Tooltip
                      contentStyle={{
                        backgroundColor: "hsl(var(--card))",
                        border: "1px solid hsl(var(--border))",
                        borderRadius: 8,
                        fontSize: 12,
                      }}
                      formatter={(v) => [
                        `${formatNumber(Number(v), 1)}%`,
                        "체지방률",
                      ]}
                    />
                    <Line
                      type="monotone"
                      dataKey="fat"
                      stroke="#f59e0b"
                      strokeWidth={2}
                      dot={{ r: 3 }}
                      connectNulls
                    />
                  </LineChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>
          )}
        </>
      )}
    </div>
  );
}
