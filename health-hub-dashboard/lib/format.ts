import { format, parseISO } from "date-fns";
import { ko } from "date-fns/locale";

export function formatDate(iso: string, fmt = "M월 d일"): string {
  return format(parseISO(iso), fmt, { locale: ko });
}

export function formatDateFull(iso: string): string {
  return format(parseISO(iso), "yyyy년 M월 d일 (EEE)", { locale: ko });
}

export function formatTime(iso: string): string {
  return format(parseISO(iso), "HH:mm");
}

export function formatDuration(minutes: number): string {
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  if (h === 0) return `${m}분`;
  if (m === 0) return `${h}시간`;
  return `${h}시간 ${m}분`;
}

export function formatNumber(n: number, decimals = 0): string {
  return n.toLocaleString("ko-KR", {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  });
}

export function formatDistance(meters: number): string {
  if (meters < 1000) return `${Math.round(meters)}m`;
  return `${(meters / 1000).toFixed(1)}km`;
}

export function today(): string {
  return format(new Date(), "yyyy-MM-dd");
}

export function daysAgo(n: number): string {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return format(d, "yyyy-MM-dd");
}
