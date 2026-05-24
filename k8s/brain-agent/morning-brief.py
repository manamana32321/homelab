#!/usr/bin/env python3
"""
Brain Agent — Morning Brief (v3.0: 완전 결정론)

매일 07:30 KST 실행. 철학: Compiled AI / Crystallization (전체 완성)
  - 데이터 수집·포맷팅 = 결정론 (REST 직접 호출)
  - "오늘 한 줄" 메타 레이어도 결정론 (v2.1 OpenAI gpt-4o-mini → 우선순위 cascade)

Sources:
  1. health-hub REST — https://health.json-server.win/api/v1/summary
  2. Canvas REST — 7일 내 과제 + 최근 24h 공지

LLM 의존 0. 환각/외부 의존/비용 모두 제거.
"""
import datetime
import html
import os
import sys
import traceback
from typing import Any

import httpx

# --- Config ---
KST = datetime.timezone(datetime.timedelta(hours=9))
NOW = datetime.datetime.now(KST)
TODAY = NOW.date()
YESTERDAY = TODAY - datetime.timedelta(days=1)
WEEKDAY_KR = "월화수목금토일"[TODAY.weekday()]

HEALTH_HUB_URL = os.environ.get("HEALTH_HUB_URL", "https://health.json-server.win").rstrip("/")
HEALTH_HUB_TOKEN = os.environ["HEALTH_HUB_TOKEN"]
CANVAS_BASE_URL = os.environ["CANVAS_BASE_URL"].rstrip("/")
CANVAS_ACCESS_TOKEN = os.environ["CANVAS_ACCESS_TOKEN"]
TELEGRAM_BOT_TOKEN = os.environ["TELEGRAM_BOT_TOKEN"]
TELEGRAM_CHAT_ID = os.environ["TELEGRAM_CHAT_ID"]
TELEGRAM_THREAD_ID = os.environ.get("TELEGRAM_THREAD_ID")
DRY_RUN = os.environ.get("BRAIN_AGENT_DRY_RUN") == "1"


# =====================
# Source 1: health-hub REST (direct)
# =====================
def fetch_health() -> dict:
    resp = httpx.get(
        f"{HEALTH_HUB_URL}/api/v1/summary",
        params={"date": YESTERDAY.isoformat(), "tz": "Asia/Seoul"},
        headers={"Authorization": f"Bearer {HEALTH_HUB_TOKEN}"},
        timeout=30.0,
    )
    resp.raise_for_status()
    return resp.json()


# =====================
# Source 2: Canvas REST
# =====================
def canvas_get(path: str, params: dict = None) -> Any:
    headers = {"Authorization": f"Bearer {CANVAS_ACCESS_TOKEN}"}
    with httpx.Client(timeout=30.0) as client:
        resp = client.get(
            f"{CANVAS_BASE_URL}{path}",
            headers=headers,
            params=params or {},
        )
        resp.raise_for_status()
        return resp.json()


def fetch_active_courses() -> list[dict]:
    """현재 학기 강의만 반환.

    Canvas 'active enrollment'는 과거 학기도 포함되어 30+개 반환됨.
    정규 학기(비정규·법정의무교육 제외) 중 start_at 가장 최근인 term만 필터.
    """
    all_courses = canvas_get(
        "/api/v1/courses",
        {"enrollment_state": "active", "per_page": 50, "include[]": ["term"]},
    )

    regular = [
        c for c in all_courses
        if (term := c.get("term") or {}).get("name")
        and "비정규" not in term["name"]
        and term.get("start_at")
    ]

    if not regular:
        print("  [warn] 정규 학기 course 없음, 전체 반환", file=sys.stderr)
        return all_courses

    latest_term_name = max(
        regular, key=lambda c: c["term"]["start_at"]
    )["term"]["name"]

    current = [
        c for c in all_courses
        if (c.get("term") or {}).get("name") == latest_term_name
    ]

    print(
        f"  Current term: '{latest_term_name}' "
        f"({len(current)}/{len(all_courses)} courses)"
    )
    return current


def fetch_upcoming_assignments(course_ids: list[int]) -> list[dict]:
    results = []
    now_iso = NOW.isoformat()
    week_later = (NOW + datetime.timedelta(days=7)).isoformat()
    for cid in course_ids:
        try:
            items = canvas_get(
                f"/api/v1/courses/{cid}/assignments",
                {"order_by": "due_at", "per_page": 50, "bucket": "upcoming"},
            )
            for a in items:
                if not a.get("due_at"):
                    continue
                if a.get("has_submitted_submissions"):
                    continue
                if now_iso <= a["due_at"] <= week_later:
                    results.append(
                        {
                            "course_id": cid,
                            "name": a["name"],
                            "due_at": a["due_at"],
                            "url": a.get("html_url"),
                        }
                    )
        except Exception as e:
            print(f"  [warn] course {cid} assignments: {e}", file=sys.stderr)
    return sorted(results, key=lambda x: x["due_at"])


def fetch_recent_announcements(course_ids: list[int]) -> list[dict]:
    if not course_ids:
        return []
    since = (NOW - datetime.timedelta(days=1)).isoformat()
    try:
        items = canvas_get(
            "/api/v1/announcements",
            {
                "context_codes[]": [f"course_{cid}" for cid in course_ids],
                "start_date": since,
                "per_page": 20,
            },
        )
        return [
            {
                "title": a.get("title", "(no title)"),
                "posted_at": a.get("posted_at", ""),
                "url": a.get("html_url"),
                "context_code": a.get("context_code", ""),
            }
            for a in items
        ]
    except Exception as e:
        print(f"  [warn] announcements: {e}", file=sys.stderr)
        return []


# =====================
# Deterministic formatters
# =====================
def _esc(s: Any) -> str:
    """Telegram HTML mode: escape user-supplied content (<, >, &)."""
    return html.escape(str(s), quote=False)


def format_health(data: dict) -> str:
    lines = [f"<b>어제 건강 ({YESTERDAY})</b>"]

    sleep = data.get("sleep") or {}
    if (dur := sleep.get("duration_m")):
        h, m = dur // 60, dur % 60
        lines.append(f"🛏 수면: {h}h {m}m")

    if (steps := data.get("total_steps")):
        lines.append(f"👟 걸음: {steps:,}보")

    if (hr := data.get("avg_heart_rate")):
        lines.append(f"❤️ 심박 평균: {hr:.0f} bpm")

    if (dist := data.get("total_distance_m")):
        km = dist / 1000
        lines.append(f"🗺 이동: {km:.2f} km")

    if (cal := data.get("total_calories")):
        lines.append(f"🔥 칼로리: {cal} kcal")

    if (spo2 := data.get("avg_spo2")):
        lines.append(f"🫁 SpO2: {spo2}%")

    exercises = data.get("exercises") or []
    if exercises:
        total_min = sum(e.get("duration_m", 0) for e in exercises)
        types = sorted({e.get("exercise_type_name", "?") for e in exercises})
        lines.append(
            f"🏃 운동: {len(exercises)}회 · 총 {total_min}분 "
            f"({_esc(', '.join(types))})"
        )

    if len(lines) == 1:
        lines.append("<i>데이터 없음 (기기 미착용?)</i>")
    return "\n".join(lines)


def format_icampus(
    assignments: list[dict],
    announcements: list[dict],
    courses: list[dict],
) -> str:
    course_map = {
        c["id"]: c.get("course_code") or c.get("name", f"course{c['id']}")
        for c in courses
    }
    lines = ["<b>🎓 아이캠퍼스</b>"]

    if announcements:
        lines.append("")
        lines.append("<i>최근 24h 공지:</i>")
        for a in announcements[:5]:
            code = a.get("context_code", "").replace("course_", "")
            try:
                code_name = course_map.get(int(code), code)
            except ValueError:
                code_name = code
            lines.append(f"• [{_esc(code_name)}] {_esc(a['title'])}")
    else:
        lines.append("<i>최근 공지 없음</i>")

    if assignments:
        lines.append("")
        lines.append("<i>7일 내 과제:</i>")
        for a in assignments[:10]:
            due = datetime.datetime.fromisoformat(
                a["due_at"].replace("Z", "+00:00")
            ).astimezone(KST)
            days_left = (due.date() - TODAY).days
            marker = "🚨" if days_left == 0 else ("⏰" if days_left <= 2 else "📅")
            course_name = course_map.get(a["course_id"], f"c{a['course_id']}")
            lines.append(
                f"{marker} [{_esc(course_name)}] {_esc(a['name'])} — "
                f"{due.strftime('%m/%d %H:%M')} (D-{days_left})"
            )
    else:
        lines.append("")
        lines.append("<i>7일 내 제출 과제 없음</i>")

    return "\n".join(lines)


# =====================
# Highlight layer — 결정론 우선순위 cascade
# (v2.1까지는 LLM이 암묵적으로 하던 sorter 역할. v3에서 명시 룰로 정착)
# =====================
def days_left(assignment: dict) -> int:
    due = datetime.datetime.fromisoformat(
        assignment["due_at"].replace("Z", "+00:00")
    ).astimezone(KST)
    return (due.date() - TODAY).days


def deterministic_highlight(
    health: dict,
    assignments: list[dict],
    announcements: list[dict],
) -> str:
    """오늘 가장 주목할 한 줄을 결정론으로 결정.

    위에서 아래로 평가, 매치된 첫 룰이 이김.
    cascade 순서·임계치는 도메인 지식(개인 패턴) — PR review에서 refine 가능.
    """
    today_due = [a for a in assignments if days_left(a) == 0]
    soon_due  = [a for a in assignments if 1 <= days_left(a) <= 2]
    sleep_min = ((health.get("sleep") or {}).get("duration_m") or 0)
    sleep_h   = sleep_min / 60 if sleep_min else 0
    steps     = health.get("total_steps") or 0

    # 1. 격무 경고 — 마감 + 수면 부족 콤보 (가장 위험)
    if today_due and 0 < sleep_h < 6:
        return f"⚠️ 오늘 마감 {len(today_due)}건 + 수면 {sleep_h:.0f}h, 무리하지 마"
    # 2. 오늘 마감
    if today_due:
        head = today_due[0]["name"][:20]
        return f"🚨 오늘 마감 {len(today_due)}건 (가장 급한 건 [{head}])"
    # 3. 심각한 수면 부족
    if 0 < sleep_h < 5:
        return f"🛏 어제 수면 {sleep_h:.0f}h, 컨디션 회복 우선"
    # 4. D-1 ~ D-2 마감
    if soon_due:
        return f"⏰ D-2 이내 마감 {len(soon_due)}건"
    # 5. 거의 안 움직인 어제
    if 0 < steps < 1000:
        return "🏃 어제 거의 안 움직임, 오늘은 잠깐이라도"
    # 6. 공지가 있다
    if announcements:
        return f"📣 어제 공지 {len(announcements)}건 확인"
    # 7. 평일 default
    return "✨ 평범한 하루, 루틴대로"


# =====================
# Telegram
# =====================
def send_telegram(text: str) -> None:
    if DRY_RUN:
        print(f"[DRY_RUN] Telegram message:\n{text}\n")
        return
    payload = {
        "chat_id": TELEGRAM_CHAT_ID,
        "text": text,
        "parse_mode": "HTML",
    }
    if TELEGRAM_THREAD_ID:
        payload["message_thread_id"] = int(TELEGRAM_THREAD_ID)
    resp = httpx.post(
        f"https://api.telegram.org/bot{TELEGRAM_BOT_TOKEN}/sendMessage",
        json=payload,
        timeout=30.0,
    )
    resp.raise_for_status()
    print("Telegram sent.")


# =====================
# Main
# =====================
def main() -> int:
    try:
        print("Fetching health...")
        health = fetch_health()

        print("Fetching Canvas courses...")
        courses = fetch_active_courses()
        course_ids = [c["id"] for c in courses]

        print(f"Fetching assignments + announcements for {len(course_ids)} courses...")
        assignments = fetch_upcoming_assignments(course_ids)
        announcements = fetch_recent_announcements(course_ids)

        sections = {
            "health": format_health(health),
            "icampus": format_icampus(assignments, announcements, courses),
        }

        highlight = deterministic_highlight(health, assignments, announcements)

        brief = "\n\n".join(
            [
                f"🌅 <b>Morning Brief — {TODAY} ({WEEKDAY_KR})</b>",
                sections["health"],
                sections["icampus"],
                f"<b>✨ 오늘 한 줄</b>\n{_esc(highlight)}",
            ]
        )
        send_telegram(brief)
        print("Done.")
        return 0

    except Exception as e:
        err = (
            f"⚠️ brain-agent FAILED: {_esc(type(e).__name__)}: {_esc(e)}\n\n"
            f"<pre>{_esc(traceback.format_exc()[:1000])}</pre>"
        )
        print(err, file=sys.stderr)
        try:
            send_telegram(err)
        except Exception:
            pass
        return 1


if __name__ == "__main__":
    sys.exit(main())
