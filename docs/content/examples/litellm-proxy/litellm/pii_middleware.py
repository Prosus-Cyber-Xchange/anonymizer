import logging
import re
from typing import Any, Literal, Optional

import httpx
from litellm.integrations.custom_logger import CustomLogger
from litellm.proxy.proxy_server import DualCache, UserAPIKeyAuth

ANONYMIZER_URL = "http://anonymizer:8080"

PII_ENTITIES = [
    {"name": "EMAIL", "redaction": {"replacement": "[EMAIL]"}},
    {"name": "PHONE", "redaction": {"replacement": "[PHONE]"}},
    {"name": "CPF_NUMBER", "redaction": {"replacement": "[CPF]"}},
    {"name": "CREDIT_CARD", "redaction": {"replacement": "[CC]"}},
    {"name": "IP_ADDRESS", "redaction": {"replacement": "[IP]"}},
]

PLACEHOLDER_RE = re.compile(r"\[(?:EMAIL|PHONE|CPF|CC|IP)\]")

logger = logging.getLogger(__name__)


class PiiMiddleware(CustomLogger):
    def __init__(self):
        self.client: Optional[httpx.AsyncClient] = None

    def _get_client(self) -> httpx.AsyncClient:
        if self.client is None:
            self.client = httpx.AsyncClient(timeout=httpx.Timeout(10.0))
        return self.client

    async def async_pre_call_hook(
        self,
        user_api_key_dict: UserAPIKeyAuth,
        cache: DualCache,
        data: dict,
        call_type: Literal[
            "completion",
            "text_completion",
            "embeddings",
            "image_generation",
            "moderation",
            "audio_transcription",
        ],
    ) -> Optional[dict]:
        if call_type not in ("completion", "text_completion"):
            return data

        messages = data.get("messages")
        if not messages:
            return data

        client = self._get_client()

        for i, msg in enumerate(messages):
            content = msg.get("content")
            if not isinstance(content, str) or not content.strip():
                continue

            try:
                resp = await client.post(
                    f"{ANONYMIZER_URL}/api/v1/anonymize",
                    json={
                        "text": content,
                        "settings": {"entities": PII_ENTITIES},
                    },
                )
                resp.raise_for_status()
            except httpx.HTTPError as exc:
                logger.warning(
                    "anonymizer request failed for message %d: %s", i, exc
                )
                continue

            result = resp.json()
            anonymized = result.get("anonymized_text", "")
            detected = result.get("detected_entities", [])

            if not detected or anonymized == content:
                continue

            msg["content"] = anonymized

        return data


proxy_handler_instance = PiiMiddleware()
