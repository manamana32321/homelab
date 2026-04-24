#!/usr/bin/env python3
"""
Brain Agent — Morning Brief (v2.1: REST direct)

매일 07:30 KST 실행. 철학: Compiled AI / Crystallization
  - 데이터 수집·포맷팅 = 결정론 (REST 직접 호출, MCP 경유 X)
  - LLM 메타 레이어 1회 호출만 = 다중 소스 종합 "오늘 한 줄"

Sources (v2.1):
  1. health-hub REST — https://health.json-server.win/api/v1/summary
  2. Canvas REST — 7일 내 과제 + 최근 24h 공지
  3. (v3 예정) Google Calendar, Google Tasks, brain git log

MCP는 LLM agent의 tool selection 용 프로토콜.
결정론 cron엔 REST 직접이 정답 (세션 handshake 3회 → 1회 요청).
"""
import datetime
import json
import os
import sys
import traceback
from typing import Any

import httpx
from openai import OpenAI

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
OPENAI_API_KEY = os.environ["OPENAI_API_KEY"]
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
def format_health(data: dict) -> str:
    lines = [f"*어제 건강 ({YESTERDAY})*"]

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
        types = {e.get("exercise_type_name", "?") for e in exercises}
        lines.append(f"🏃 운동: {len(exercises)}회 · 총 {total_min}분 ({', '.join(sorted(types))})")

    if len(lines) == 1:
        lines.append("_데이터 없음 (기기 미착용?)_")
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
    lines = ["*🎓 아이캠퍼스*"]

    if announcements:
        lines.append("")
        lines.append("_최근 24h 공지:_")
        for a in announcements[:5]:
            code = a.get("context_code", "").replace("course_", "")
            try:
                code_name = course_map.get(int(code), code)
            except ValueError:
                code_name = code
            lines.append(f"• \\[{code_name}] {a['title']}")
    else:
        lines.append("_최근 공지 없음_")

    if assignments:
        lines.append("")
        lines.append("_7일 내 과제:_")
        for a in assignments[:10]:
            due = datetime.datetime.fromisoformat(
                a["due_at"].replace("Z", "+00:00")
            ).astimezone(KST)
            days_left = (due.date() - TODAY).days
            marker = "🚨" if days_left == 0 else ("⏰" if days_left <= 2 else "📅")
            course_name = course_map.get(a["course_id"], f"c{a['course_id']}")
            lines.append(
                f"{marker} \\[{course_name}] {a['name']} — "
                f"{due.strftime('%m/%d %H:%M')} (D-{days_left})"
            )
    else:
        lines.append("")
        lines.append("_7일 내 제출 과제 없음_")

    return "\n".join(lines)


# =====================
# Meta layer (the only LLM call)
# =====================
META_PROMPT = (
    "너는 장수님의 개인 비서. 아래 오늘 아침 데이터를 보고 단 한 줄로 "
    "'오늘 가장 주목할 것'을 뽑아줘.\n\n"
    "규칙:\n"
    "- 반말, 짧은 단정형 한국어\n"
    "- 데이터가 말하지 않는 것을 지어내지 말 것 (환각 금지)\n"
    "- 이모지 1개로 시작\n"
    "- 50자 이내\n"
    "- 평범한 날이면 '평범한 하루, 루틴대로' 같이 담담하게"
)


def meta_summarize(sections: dict) -> str:
    client = OpenAI(api_key=OPENAI_API_KEY)
    context = "\n\n".join(f"## {name}\n{body}" for name, body in sections.items())
    resp = client.chat.completions.create(
        model="gpt-4o-mini",
        messages=[
            {"role": "system", "content": META_PROMPT},
            {"role": "user", "content": context},
        ],
        temperature=0.3,
        max_tokens=120,
    )
    return resp.choices[0].message.content.strip()


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
        "parse_mode": "Markdown",
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

        print("Meta summarizing...")
        meta = meta_summarize(sections)

        brief = "\n\n".join(
            [
                f"🌅 *Morning Brief — {TODAY} ({WEEKDAY_KR})*",
                sections["health"],
                sections["icampus"],
                f"*✨ 오늘 한 줄*\n{meta}",
            ]
        )
        send_telegram(brief)
        print("Done.")
        return 0

    except Exception as e:
        err = (
            f"⚠️ brain-agent FAILED: {type(e).__name__}: {e}\n\n"
            f"```\n{traceback.format_exc()[:1000]}\n```"
        )
        print(err, file=sys.stderr)
        try:
            send_telegram(err)
        except Exception:
            pass
        return 1


if __name__ == "__main__":
    sys.exit(main())
