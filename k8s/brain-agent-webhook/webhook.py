#!/usr/bin/env python3
"""
brain-agent-webhook — Telegram 양방향 HTTP 서버

수신:
  - 텍스트 메시지 → health-hub /api/v1/notes 저장
  - 사진 메시지 → OpenAI gpt-4o Vision 분석 → 확인 버튼 → /api/v1/meals 저장
  - callback_query (inline 버튼) → 저장/취소 처리

Stateless 설계:
  - callback_data 에 영양소 숫자를 압축 인코딩 (Telegram 64 byte 제한)
  - 식사 이름은 confirmation message 본문에서 재파싱

철학: Compiled AI
  - 이벤트 수신·검증·routing = 결정론
  - Vision 분석 = LLM (비정형 이미지 → structured JSON, 정당한 LLM 사용처)
  - 저장 = 결정론 (REST POST)
"""
import base64
import datetime
import json
import os
import re
import traceback

import httpx
from fastapi import FastAPI, Header, HTTPException, Request
from openai import OpenAI


# --- Config ---
TELEGRAM_BOT_TOKEN = os.environ["TELEGRAM_BOT_TOKEN"]
TELEGRAM_CHAT_ID = os.environ["TELEGRAM_CHAT_ID"]
TELEGRAM_THREAD_ID = os.environ.get("TELEGRAM_THREAD_ID")
OPENAI_API_KEY = os.environ["OPENAI_API_KEY"]
OPENAI_VISION_MODEL = os.environ.get("OPENAI_VISION_MODEL", "gpt-4o")
HEALTH_HUB_URL = os.environ.get(
    "HEALTH_HUB_URL", "https://health.json-server.win"
).rstrip("/")
HEALTH_HUB_TOKEN = os.environ["HEALTH_HUB_TOKEN"]
WEBHOOK_SECRET = os.environ.get("WEBHOOK_SECRET", "")

TELEGRAM_API = f"https://api.telegram.org/bot{TELEGRAM_BOT_TOKEN}"
TELEGRAM_FILE_API = f"https://api.telegram.org/file/bot{TELEGRAM_BOT_TOKEN}"

app = FastAPI(title="brain-agent-webhook")
openai_client = OpenAI(api_key=OPENAI_API_KEY)


# =====================
# Telegram helpers
# =====================
async def tg_request(method: str, payload: dict) -> dict:
    async with httpx.AsyncClient(timeout=30.0) as c:
        resp = await c.post(f"{TELEGRAM_API}/{method}", json=payload)
        resp.raise_for_status()
        return resp.json()


async def tg_send_message(
    text: str,
    reply_markup: dict | None = None,
    reply_to: int | None = None,
) -> dict:
    payload = {
        "chat_id": TELEGRAM_CHAT_ID,
        "text": text,
        "parse_mode": "Markdown",
    }
    if TELEGRAM_THREAD_ID:
        payload["message_thread_id"] = int(TELEGRAM_THREAD_ID)
    if reply_markup:
        payload["reply_markup"] = reply_markup
    if reply_to:
        payload["reply_to_message_id"] = reply_to
    return await tg_request("sendMessage", payload)


async def tg_edit_message(chat_id: int, message_id: int, text: str) -> dict:
    return await tg_request(
        "editMessageText",
        {
            "chat_id": chat_id,
            "message_id": message_id,
            "text": text,
            "parse_mode": "Markdown",
        },
    )


async def tg_answer_callback(callback_query_id: str, text: str = "") -> dict:
    return await tg_request(
        "answerCallbackQuery",
        {"callback_query_id": callback_query_id, "text": text},
    )


async def tg_download_photo(file_id: str) -> bytes:
    async with httpx.AsyncClient(timeout=30.0) as c:
        resp = await c.get(f"{TELEGRAM_API}/getFile", params={"file_id": file_id})
        resp.raise_for_status()
        file_path = resp.json()["result"]["file_path"]

        resp = await c.get(f"{TELEGRAM_FILE_API}/{file_path}")
        resp.raise_for_status()
        return resp.content


# =====================
# health-hub
# =====================
async def hh_post(path: str, body: dict) -> dict:
    async with httpx.AsyncClient(timeout=30.0) as c:
        resp = await c.post(
            f"{HEALTH_HUB_URL}{path}",
            headers={"Authorization": f"Bearer {HEALTH_HUB_TOKEN}"},
            json=body,
        )
        resp.raise_for_status()
        return resp.json()


# =====================
# Vision analysis (OpenAI gpt-4o)
# =====================
ANALYSIS_PROMPT = """다음 식사 사진을 분석해서 영양 정보 JSON을 반환해.

Schema (모든 필드 필수, 누락 금지):
- name: 한국어 식사 이름 (예: "김치찌개 + 공기밥")
- meal_type: "breakfast" / "lunch" / "dinner" / "snack" 중 하나 (시간·플레이팅 추정)
- calories: 총 kcal, integer
- protein_g: 단백질 g, integer
- carbs_g: 탄수화물 g, integer
- fat_g: 지방 g, integer
- note: 한 줄 코멘트 (한국어, 영양 균형·주의점)

포션 크기는 한국 기준 일반 1인분으로 추정. 정확한 수치보다 합리적 추정 우선."""


def analyze_meal_image(image_bytes: bytes, media_type: str = "image/jpeg") -> dict:
    image_b64 = base64.standard_b64encode(image_bytes).decode("utf-8")
    resp = openai_client.chat.completions.create(
        model=OPENAI_VISION_MODEL,
        max_tokens=512,
        temperature=0.3,
        response_format={"type": "json_object"},  # JSON 강제
        messages=[
            {
                "role": "user",
                "content": [
                    {
                        "type": "image_url",
                        "image_url": {
                            "url": f"data:{media_type};base64,{image_b64}"
                        },
                    },
                    {"type": "text", "text": ANALYSIS_PROMPT},
                ],
            }
        ],
    )
    return json.loads(resp.choices[0].message.content)


# =====================
# Stateless callback encoding
# =====================
def encode_callback(action: str, data: dict) -> str:
    """action|k1=v1|k2=v2 — Telegram callback_data 64 byte 제한 내."""
    parts = [action] + [f"{k}={v}" for k, v in data.items()]
    return "|".join(parts)


def decode_callback(cb: str) -> tuple[str, dict]:
    parts = cb.split("|")
    action = parts[0]
    data = {}
    for p in parts[1:]:
        if "=" in p:
            k, v = p.split("=", 1)
            try:
                data[k] = int(v)
            except ValueError:
                data[k] = v
    return action, data


def extract_meal_name_from_message(text: str) -> str:
    """Confirmation message 첫 줄에서 식사 이름 파싱."""
    first = text.split("\n", 1)[0]
    name = re.sub(r"^[🍜🍽🍚🥗🍖🍝]+\s*", "", first).strip()
    return name or "unknown"


# =====================
# Handlers
# =====================
@app.get("/healthz")
def healthz():
    return {"ok": True}


@app.post("/webhook/telegram")
async def telegram_webhook(
    request: Request,
    x_telegram_bot_api_secret_token: str | None = Header(None),
):
    if WEBHOOK_SECRET and x_telegram_bot_api_secret_token != WEBHOOK_SECRET:
        raise HTTPException(status_code=403, detail="invalid secret")

    try:
        update = await request.json()
    except Exception:
        raise HTTPException(status_code=400, detail="invalid json")

    if cbq := update.get("callback_query"):
        try:
            await handle_callback(cbq)
        except Exception:
            traceback.print_exc()
            try:
                await tg_answer_callback(cbq["id"], "처리 실패")
            except Exception:
                pass
        return {"ok": True}

    message = update.get("message") or update.get("edited_message")
    if not message:
        return {"ok": True}

    chat_id = str(message.get("chat", {}).get("id"))
    if chat_id != TELEGRAM_CHAT_ID:
        return {"ok": True}

    if TELEGRAM_THREAD_ID:
        thread_id = message.get("message_thread_id")
        if str(thread_id) != TELEGRAM_THREAD_ID:
            return {"ok": True}

    try:
        if "photo" in message:
            await handle_photo(message)
        elif "text" in message:
            await handle_text(message)
    except Exception as e:
        traceback.print_exc()
        try:
            await tg_send_message(f"❌ 처리 실패: {type(e).__name__}: {e}")
        except Exception:
            pass

    return {"ok": True}


async def handle_text(msg: dict):
    text = msg["text"]

    if text.startswith("/"):
        if text == "/ping":
            await tg_send_message("pong", reply_to=msg["message_id"])
        else:
            await tg_send_message(
                "사용법:\n- 텍스트 → notes 저장\n- 사진 → 식사 분석\n- /ping",
                reply_to=msg["message_id"],
            )
        return

    await hh_post(
        "/api/v1/notes",
        {
            "time": datetime.datetime.now(datetime.timezone.utc).isoformat(),
            "category": "tg-text",
            "text": text,
        },
    )
    await tg_send_message("✅ 노트 저장됨", reply_to=msg["message_id"])


async def handle_photo(msg: dict):
    photos = msg["photo"]
    largest = max(photos, key=lambda p: p.get("width", 0) * p.get("height", 0))
    file_id = largest["file_id"]

    await tg_send_message("🔍 분석 중...", reply_to=msg["message_id"])

    image = await tg_download_photo(file_id)
    meal = analyze_meal_image(image)

    lines = [
        f"🍜 {meal['name']}",
        f"- 칼로리: {meal.get('calories', '?')} kcal",
        f"- 단백질: {meal.get('protein_g', '?')}g",
        f"- 탄수: {meal.get('carbs_g', '?')}g",
        f"- 지방: {meal.get('fat_g', '?')}g",
    ]
    if note := meal.get("note"):
        lines.append(f"_{note}_")
    reply_text = "\n".join(lines)

    cb_save = encode_callback(
        "save_meal",
        {
            "t": (meal.get("meal_type") or "snack")[:3],
            "c": int(meal.get("calories") or 0),
            "p": int(meal.get("protein_g") or 0),
            "cb": int(meal.get("carbs_g") or 0),
            "f": int(meal.get("fat_g") or 0),
        },
    )

    keyboard = {
        "inline_keyboard": [
            [
                {"text": "✅ 저장", "callback_data": cb_save},
                {"text": "❌ 취소", "callback_data": "cancel_meal"},
            ]
        ]
    }

    await tg_send_message(
        reply_text, reply_markup=keyboard, reply_to=msg["message_id"]
    )


MEAL_TYPE_MAP = {
    "bre": "breakfast",
    "lun": "lunch",
    "din": "dinner",
    "sna": "snack",
}


async def handle_callback(cbq: dict):
    data = cbq["data"]
    cbq_id = cbq["id"]
    msg = cbq["message"]
    chat_id = msg["chat"]["id"]
    message_id = msg["message_id"]

    action, fields = decode_callback(data)

    if action == "save_meal":
        meal_name = extract_meal_name_from_message(msg["text"])
        try:
            await hh_post(
                "/api/v1/meals",
                {
                    "time": datetime.datetime.now(datetime.timezone.utc).isoformat(),
                    "name": meal_name,
                    "meal_type": MEAL_TYPE_MAP.get(str(fields.get("t", "")), None),
                    "calories": float(fields.get("c", 0)),
                    "protein_g": float(fields.get("p", 0)),
                    "carbs_g": float(fields.get("cb", 0)),
                    "fat_g": float(fields.get("f", 0)),
                },
            )
            await tg_edit_message(
                chat_id, message_id, msg["text"] + "\n\n✅ 저장됨"
            )
            await tg_answer_callback(cbq_id, "저장 완료")
        except Exception as e:
            await tg_answer_callback(cbq_id, f"실패: {type(e).__name__}")
            raise
    elif action == "cancel_meal":
        await tg_edit_message(chat_id, message_id, msg["text"] + "\n\n❌ 취소됨")
        await tg_answer_callback(cbq_id, "취소")
    else:
        await tg_answer_callback(cbq_id, f"unknown: {action}")
