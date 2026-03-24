"use client";

import { Dumbbell } from "lucide-react";
import { useExercises } from "@/lib/hooks/use-exercises";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { daysAgo, today, formatDate, formatDuration, formatNumber, formatDistance } from "@/lib/format";

export default function ExercisePage() {
  const { data, isLoading } = useExercises(daysAgo(30), today());

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold">운동</h2>

      {isLoading ? (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-20 w-full" />
          ))}
        </div>
      ) : (data ?? []).length === 0 ? (
        <Card>
          <CardContent className="p-8 text-center text-muted-foreground">
            최근 30일 운동 기록이 없습니다
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {(data ?? []).map((ex) => (
            <Card key={ex.id}>
              <CardContent className="p-4">
                <div className="flex items-start gap-3">
                  <div className="rounded-lg bg-emerald-500/10 p-2">
                    <Dumbbell className="h-5 w-5 text-emerald-500" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-medium">
                        {ex.exercise_type}
                      </span>
                      <Badge variant="secondary">
                        {formatDuration(ex.duration_m)}
                      </Badge>
                    </div>
                    <p className="text-xs text-muted-foreground mt-1">
                      {formatDate(ex.start_time)}
                    </p>
                    <div className="flex gap-4 mt-2 text-sm text-muted-foreground">
                      {ex.calories_kcal && (
                        <span>{formatNumber(ex.calories_kcal, 0)} kcal</span>
                      )}
                      {ex.distance_m && (
                        <span>{formatDistance(ex.distance_m)}</span>
                      )}
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
