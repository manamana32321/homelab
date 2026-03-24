const API_TOKEN = process.env.NEXT_PUBLIC_API_TOKEN ?? "";
const API_BASE = process.env.NEXT_PUBLIC_API_BASE ?? "";

export async function apiFetch<T>(
  path: string,
  params?: Record<string, string>,
): Promise<T> {
  const url = new URL(`${API_BASE}${path}`, window.location.origin);
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      if (v) url.searchParams.set(k, v);
    }
  }

  const res = await fetch(url.toString(), {
    headers: API_TOKEN
      ? { Authorization: `Bearer ${API_TOKEN}` }
      : {},
  });

  if (!res.ok) {
    const body = await res.text();
    throw new Error(`API ${res.status}: ${body}`);
  }

  return res.json();
}
