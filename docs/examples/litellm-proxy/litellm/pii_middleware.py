import logging
import re
import difflib
import uuid
from collections import defaultdict
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
        self._store: dict[str, dict[str, str]] = {}

    def _get_client(self) -> httpx.AsyncClient:
        if self.client is None:
            self.client = httpx.AsyncClient(timeout=httpx.Timeout(10.0))
        return self.client

    @staticmethod
    def _extract_replacements(
        original: str, anonymized: str
    ) -> dict[str, str]:
        replacements: dict[str, str] = {}
        s = difflib.SequenceMatcher(None, original, anonymized)
        for tag, i1, i2, j1, j2 in s.get_opcodes():
            if tag == "replace":
                orig_val = original[i1:i2]
                anon_val = anonymized[j1:j2]
                replacements[anon_val] = orig_val
        return replacements

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
        request_id = str(uuid.uuid4())

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

            replacements = self._extract_replacements(content, anonymized)
            if replacements:
                self._store[request_id] = replacements
                msg["content"] = anonymized

        data["_pii_request_id"] = request_id
        return data

    async def async_post_call_success_hook(
        self,
        data: dict,
        user_api_key_dict: UserAPIKeyAuth,
        response: Any,
    ) -> None:
        request_id = data.get("_pii_request_id")
        if not request_id or request_id not in self._store:
            return

        replacements = self._store.pop(request_id)

        choices = getattr(response, "choices", [])
        for choice in choices:
            message = getattr(choice, "message", None)
            if message is None:
                continue
            resp_content = getattr(message, "content", None)
            if not isinstance(resp_content, str):
                continue
            for placeholder, original in replacements.items():
                resp_content = resp_content.replace(placeholder, original)
            message.content = resp_content


proxy_handler_instance = PiiMiddleware()
