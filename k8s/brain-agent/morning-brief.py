#!/usr/bin/env python3
"""
Brain Agent — Morning Brief (v2: meta-summarizer)

매일 07:30 KST 실행. 철학: Compiled AI / Crystallization
  - 각 소스 데이터 수집 = 결정론 (MCP/REST → Python template)
  - LLM 메타 레이어 1회 호출 = 다중 소스 종합 "오늘 주목할 것" 판단

Sources (v2):
  1. health-hub MCP — 어제 건강 요약
  2. Canvas REST — 7일 내 과제 + 최근 24h 공지
  3. (v3 예정) Google Calendar, Google Tasks, brain git log
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

HEALTH_HUB_URL = os.environ.get("HEALTH_HUB_URL", "https://health.json-server.win/mcp")
HEALTH_HUB_TOKEN = os.environ["HEALTH_HUB_TOKEN"]
CANVAS_BASE_URL = os.environ["CANVAS_BASE_URL"].rstrip("/")
CANVAS_ACCESS_TOKEN = os.environ["CANVAS_ACCESS_TOKEN"]
OPENAI_API_KEY = os.environ["OPENAI_API_KEY"]
TELEGRAM_BOT_TOKEN = os.environ["TELEGRAM_BOT_TOKEN"]
TELEGRAM_CHAT_ID = os.environ["TELEGRAM_CHAT_ID"]
TELEGRAM_THREAD_ID = os.environ.get("TELEGRAM_THREAD_ID")
DRY_RUN = os.environ.get("BRAIN_AGENT_DRY_RUN") == "1"


# =====================
# Source 1: health-hub MCP (Streamable HTTP)
# =====================
def call_mcp_tool(url: str, token: str, tool_name: str, arguments: dict) -> dict:
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json",
        "Accept": "application/json, text/event-stream",
    }
    with httpx.Client(timeout=30.0) as client:
        init_resp = client.post(
            url,
            headers=headers,
            json={
                "jsonrpc": "2.0",
                "id": 1,
                "method": "initialize",
                "params": {
                    "protocolVersion": "2025-06-18",
                    "capabilities": {},
                    "clientInfo": {"name": "brain-agent", "version": "2.0"},
                },
            },
        )
        init_resp.raise_for_status()
        session_id = init_resp.headers.get("mcp-session-id")

        auth_headers = dict(headers)
        if session_id:
            auth_headers["mcp-session-id"] = session_id

        client.post(
            url,
            headers=auth_headers,
            json={"jsonrpc": "2.0", "method": "notifications/initialized"},
        )

        call_resp = client.post(
            url,
            headers=auth_headers,
            json={
                "jsonrpc": "2.0",
                "id": 2,
                "method": "tools/call",
                "params": {"name": tool_name, "arguments": arguments},
            },
        )
        call_resp.raise_for_status()

        body = call_resp.text.strip()
        if body.startswith("event:"):
            for line in body.split("\n"):
                if line.startswith("data: "):
                    return json.loads(line[6:])
        return call_resp.json()


def fetch_health() -> dict:
    result = call_mcp_tool(
        HEALTH_HUB_URL,
        HEALTH_HUB_TOKEN,
        "get_daily_summary",
        {"date": YESTERDAY.isoformat(), "timezone": "Asia/Seoul"},
    )
    r = result.get("result", result)
    if isinstance(r, dict) and isinstance(r.get("content"), list):
        for item in r["content"]:
            if item.get("type") == "text":
                try:
                    return json.loads(item["text"])
                except json.JSONDecodeError:
                    return {"raw": item["text"]}
    return r if isinstance(r, dict) else {}


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
    return canvas_get(
        "/api/v1/courses",
        {"enrollment_state": "active", "per_page": 30, "include[]": ["term"]},
    )


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

    sleep_min = data.get("sleep_minutes") or data.get("total_sleep_minutes")
    if sleep_min:
        lines.append(f"🛏 수면: {sleep_min // 60}h {sleep_min % 60}m")

    if (steps := data.get("steps")):
        lines.append(f"👟 걸음: {steps:,}보")
    if (hr := data.get("heart_rate_avg") or data.get("heart_rate")):
        lines.append(f"❤️ 심박 평균: {hr} bpm")
    if (weight := data.get("weight_kg") or data.get("weight")):
        lines.append(f"⚖️ 체중: {weight}kg")
    if (cal := data.get("active_calories")):
        lines.append(f"🔥 활동 칼로리: {cal} kcal")
    if (spo2 := data.get("spo2")):
        lines.append(f"🫁 SpO2: {spo2}%")

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
