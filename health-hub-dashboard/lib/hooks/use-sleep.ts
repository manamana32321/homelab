"use client";

import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api";
import type { SleepSession } from "@/lib/types";

export function useSleep(from: string, to: string) {
  return useQuery({
    queryKey: ["sleep", from, to],
    queryFn: () => apiFetch<SleepSession[]>("/api/v1/sleep", { start_date: from, end_date: to }),
  });
}
