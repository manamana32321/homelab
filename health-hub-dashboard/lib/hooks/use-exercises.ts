"use client";

import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api";
import type { ExerciseSession } from "@/lib/types";

export function useExercises(from: string, to: string) {
  return useQuery({
    queryKey: ["exercises", from, to],
    queryFn: () =>
      apiFetch<ExerciseSession[]>("/api/v1/exercises", { start_date: from, end_date: to }),
  });
}
