"use client";

import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api";
import type { AggregatedMetric, MetricType } from "@/lib/types";

export function useMetrics(
  type: MetricType,
  from: string,
  to: string,
  interval = "1d",
) {
  return useQuery({
    queryKey: ["metrics", type, from, to, interval],
    queryFn: () =>
      apiFetch<AggregatedMetric[]>("/api/v1/metrics", {
        type,
        start_date: from,
        end_date: to,
        interval,
      }),
  });
}
