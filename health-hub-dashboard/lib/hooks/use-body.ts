"use client";

import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api";
import type { BodyMeasurement } from "@/lib/types";

export function useBody(from: string, to: string) {
  return useQuery({
    queryKey: ["body", from, to],
    queryFn: () => apiFetch<BodyMeasurement[]>("/api/v1/body", { from, to }),
  });
}
