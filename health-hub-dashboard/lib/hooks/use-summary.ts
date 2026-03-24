"use client";

import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api";
import type { DailySummary } from "@/lib/types";

export function useSummary(date: string) {
  return useQuery({
    queryKey: ["summary", date],
    queryFn: () => apiFetch<DailySummary>("/api/v1/summary", { date }),
  });
}
