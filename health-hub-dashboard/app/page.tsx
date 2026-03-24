"use client";

import { useState, useEffect } from "react";
import {
  Footprints,
  Moon,
  Heart,
  Flame,
  MapPin,
  Activity,
  Scale,
  Dumbbell,
} from "lucide-react";
import { useSummary } from "@/lib/hooks/use-summary";
import { SummaryCard } from "@/components/charts/summary-card";
import {
  formatNumber,
  formatDuration,
  formatDistance,
  today,
  formatDateFull,
} from "@/lib/format";

export default function HomePage() {
  const [date, setDate] = useState("");
  useEffect(() => setDate(today()), []);
  const { data, isLoading } = useSummary(date || today());

  return (
    <div>
      <h2 className="text-xl font-bold mb-1">오늘의 건강</h2>
      <p className="text-sm text-muted-foreground mb-6">
        {date && formatDateFull(date + "T00:00:00")}
      </p>

      <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
        <SummaryCard
          icon={Footprints}
          label="걸음수"
          value={data?.total_steps ? formatNumber(data.total_steps) : "—"}
          unit="걸음"
          iconColor="text-green-500"
          loading={isLoading}
        />
        <SummaryCard
          icon={Moon}
          label="수면"
          value={data?.sleep ? formatDuration(data.sleep.duration_m) : "—"}
          sub={
            data?.sleep
              ? `${data.sleep.stages?.filter((s) => s.stage === "deep" || s.stage === "3").reduce((a, s) => a + s.duration_m, 0) ?? 0}분 깊은잠`
              : undefined
          }
          iconColor="text-indigo-500"
          loading={isLoading}
        />
        <SummaryCard
          icon={Heart}
          label="평균 심박수"
          value={
            data?.avg_heart_rate
              ? formatNumber(data.avg_heart_rate, 0)
              : "—"
          }
          unit="bpm"
          iconColor="text-red-500"
          loading={isLoading}
        />
        <SummaryCard
          icon={Flame}
          label="소모 칼로리"
          value={
            data?.total_calories
              ? formatNumber(data.total_calories, 0)
              : "—"
          }
          unit="kcal"
          iconColor="text-orange-500"
          loading={isLoading}
        />
        <SummaryCard
          icon={MapPin}
          label="이동 거리"
          value={
            data?.total_distance_m
              ? formatDistance(data.total_distance_m)
              : "—"
          }
          iconColor="text-blue-500"
          loading={isLoading}
        />
        <SummaryCard
          icon={Activity}
          label="산소포화도"
          value={data?.avg_spo2 ? formatNumber(data.avg_spo2, 1) : "—"}
          unit="%"
          iconColor="text-cyan-500"
          loading={isLoading}
        />
      </div>

      {data?.weight?.weight_kg && (
        <div className="mt-4">
          <SummaryCard
            icon={Scale}
            label="체중"
            value={formatNumber(data.weight.weight_kg, 1)}
            unit="kg"
            sub={
              data.weight.body_fat_pct
                ? `체지방 ${formatNumber(data.weight.body_fat_pct, 1)}%`
                : undefined
            }
            iconColor="text-purple-500"
          />
        </div>
      )}

      {data?.exercises && data.exercises.length > 0 && (
        <div className="mt-6">
          <h3 className="text-sm font-medium text-muted-foreground mb-3">
            오늘의 운동
          </h3>
          <div className="space-y-2">
            {data.exercises.map((ex) => (
              <div
                key={ex.id}
                className="flex items-center gap-3 rounded-lg border border-border p-3"
              >
                <Dumbbell className="h-4 w-4 text-emerald-500" />
                <div className="flex-1">
                  <span className="text-sm font-medium">
                    {ex.exercise_type}
                  </span>
                  <span className="text-xs text-muted-foreground ml-2">
                    {formatDuration(ex.duration_m)}
                  </span>
                </div>
                {ex.calories_kcal && (
                  <span className="text-xs text-muted-foreground">
                    {formatNumber(ex.calories_kcal, 0)} kcal
                  </span>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
