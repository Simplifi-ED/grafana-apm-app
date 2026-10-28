#!/usr/bin/env python3
"""One-shot OTLP faro.feedback span for ticket 21 plugin Feedback smoke.

Mirrors HyperDX addAction('faro.feedback', …) → SpanName=faro.feedback.
Does NOT replace the Sanad Support-form human smoke; use that for app wiring proof.

Env:
  OTLP_ENDPOINT  default http://otel-collector-logs.observability.svc.cluster.local:4318/v1/traces
  OTLP_AUTH      optional Authorization header
"""

from __future__ import annotations

import json
import os
import time
import urllib.error
import urllib.request
from typing import Any

OTLP_ENDPOINT = os.environ.get(
    "OTLP_ENDPOINT",
    "http://otel-collector-logs.observability.svc.cluster.local:4318/v1/traces",
)
OTLP_AUTH = os.environ.get("OTLP_AUTH", "").strip()
SERVICE_NAME = os.environ.get("SERVICE_NAME", "nova-frontend")
APP_VERSION = os.environ.get("APP_VERSION", "20260422164408-main")
NAMESPACE = os.environ.get("K8S_NAMESPACE", "dev-nova")


def _hex_id(nbytes: int) -> str:
    return os.urandom(nbytes).hex()


def _attr(key: str, value: str) -> dict[str, Any]:
    return {"key": key, "value": {"stringValue": value}}


def build_payload() -> dict[str, Any]:
    end = time.time_ns()
    start = end - 5_000_000
    return {
        "resourceSpans": [
            {
                "resource": {
                    "attributes": [
                        _attr("service.name", SERVICE_NAME),
                        _attr("service.version", APP_VERSION),
                        _attr("telemetry.sdk.name", "@hyperdx/otel-web"),
                        _attr("telemetry.sdk.version", "0.20.0"),
                        _attr("rum.version", "0.20.0"),
                        _attr("rum.sessionId", "ticket21-smoke-session"),
                        _attr("k8s.namespace.name", NAMESPACE),
                    ]
                },
                "scopeSpans": [
                    {
                        "scope": {"name": "@hyperdx/otel-web", "version": "0.20.0"},
                        "spans": [
                            {
                                "traceId": _hex_id(16),
                                "spanId": _hex_id(8),
                                "name": "faro.feedback",
                                "kind": 1,  # INTERNAL
                                "startTimeUnixNano": str(start),
                                "endTimeUnixNano": str(end),
                                "attributes": [
                                    _attr("event_name", "faro.feedback"),
                                    _attr(
                                        "feedback.message",
                                        "ticket-21 synthetic smoke (not from Support form)",
                                    ),
                                    _attr("feedback.category", "other"),
                                    _attr("location.href", "https://app.dev.sanad.cloud/support"),
                                ],
                                "status": {"code": 1},
                            }
                        ],
                    }
                ],
            }
        ]
    }


def main() -> None:
    body = json.dumps(build_payload()).encode("utf-8")
    headers = {"Content-Type": "application/json"}
    if OTLP_AUTH:
        headers["Authorization"] = OTLP_AUTH
    req = urllib.request.Request(OTLP_ENDPOINT, data=body, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            print(f"exported faro.feedback → {OTLP_ENDPOINT} HTTP {resp.getcode()}", flush=True)
    except urllib.error.HTTPError as e:
        raise SystemExit(f"OTLP HTTP {e.code}: {e.read().decode('utf-8', errors='replace')}") from e
    except urllib.error.URLError as e:
        raise SystemExit(f"OTLP failed: {e}") from e


if __name__ == "__main__":
    main()
