#!/usr/bin/env python3
"""
Minimal GraphQL OTel span emitter.

Posts OTLP/HTTP JSON traces with graphql.operation.* span attributes to an
OTLP collector (in-cluster collector → ClickHouse, or ClickStack :4318 with an
ingestion key).

Env:
  OTLP_ENDPOINT   default http://localhost:4318/v1/traces
  OTLP_AUTH       optional Authorization header value (ClickStack ingestion key)
  SERVICE_NAME    default graphql-demo
  SERVICE_NS      default demo
  ENVIRONMENT     default ci  (also set as resource cluster + deployment.environment)
  INTERVAL_SEC    default 15; set 0 for one-shot
  ONCE            if 1, emit once and exit (overrides INTERVAL_SEC)

Example (local collector):
  python3 scripts/graphql-demo/emit_graphql_spans.py

Example (remote ClickStack OTLP):
  OTLP_ENDPOINT=http://clickhouse-host:4318/v1/traces \\
  OTLP_AUTH="$CLICKSTACK_INGESTION_KEY" \\
  ENVIRONMENT=prod \\
  python3 scripts/graphql-demo/emit_graphql_spans.py
"""

from __future__ import annotations

import json
import os
import random
import sys
import time
import urllib.error
import urllib.request
from typing import Any

OTLP_ENDPOINT = os.environ.get("OTLP_ENDPOINT", "http://localhost:4318/v1/traces")
OTLP_AUTH = os.environ.get("OTLP_AUTH", "").strip()
SERVICE_NAME = os.environ.get("SERVICE_NAME", "graphql-demo")
SERVICE_NS = os.environ.get("SERVICE_NS", "demo")
ENVIRONMENT = os.environ.get("ENVIRONMENT", "ci")
ONCE = os.environ.get("ONCE", "0") == "1"
INTERVAL_SEC = 0 if ONCE else int(os.environ.get("INTERVAL_SEC", "15"))

# Mirror scripts/fixtures/graphql-otel-spans.sql + one extra query.
OPS: list[dict[str, Any]] = [
    {
        "name": "GetInvoice",
        "type": "query",
        "document": "query GetInvoice { invoice(id: $id) { id } }",
        "duration_ns": 12_500_000,
        "status": "Ok",
        "message": "",
    },
    {
        "name": "CreateInvoice",
        "type": "mutation",
        "document": "mutation CreateInvoice($input: InvoiceInput!) { createInvoice(input: $input) { id } }",
        "duration_ns": 42_000_000,
        "status": "Error",
        "message": "resolver failed",
    },
    {
        "name": "ListInvoices",
        "type": "query",
        "document": "query ListInvoices { invoices { id } }",
        "duration_ns": 8_000_000,
        "status": "Ok",
        "message": "",
    },
]


def _hex_id(nbytes: int) -> str:
    return os.urandom(nbytes).hex()


def _now_ns() -> int:
    return time.time_ns()


def _attr(key: str, value: str) -> dict[str, Any]:
    return {"key": key, "value": {"stringValue": value}}


def _status_code(label: str) -> int:
    # OTLP: UNSET=0, OK=1, ERROR=2
    if label == "Error":
        return 2
    if label == "Ok":
        return 1
    return 0


def build_payload(ops: list[dict[str, Any]]) -> dict[str, Any]:
    end = _now_ns()
    spans = []
    for op in ops:
        start = end - int(op["duration_ns"])
        attrs = [
            _attr("graphql.operation.name", op["name"]),
            _attr("graphql.operation.type", op["type"]),
            # Survive otel-collector-logs tail_sampling keep-api-paths policy.
            _attr("url.path", "/api/graphql"),
            _attr("http.method", "POST"),
        ]
        if op.get("document"):
            attrs.append(_attr("graphql.document", op["document"]))
        spans.append(
            {
                "traceId": _hex_id(16),
                "spanId": _hex_id(8),
                "name": f"{op['type']} {op['name']}",
                "kind": 2,  # SPAN_KIND_SERVER
                "startTimeUnixNano": str(start),
                "endTimeUnixNano": str(end),
                "attributes": attrs,
                "status": {
                    "code": _status_code(op["status"]),
                    "message": op.get("message") or "",
                },
            }
        )
        # stagger next span slightly so timestamps differ
        end = start - random.randint(1_000_000, 5_000_000)

    return {
        "resourceSpans": [
            {
                "resource": {
                    "attributes": [
                        _attr("service.name", SERVICE_NAME),
                        _attr("service.namespace", SERVICE_NS),
                        _attr("deployment.environment", ENVIRONMENT),
                        _attr("cluster", ENVIRONMENT),
                    ]
                },
                "scopeSpans": [
                    {
                        "scope": {"name": "graphql-demo", "version": "0.0.1"},
                        "spans": spans,
                    }
                ],
            }
        ]
    }


def export_once() -> None:
    body = json.dumps(build_payload(OPS)).encode("utf-8")
    headers = {"Content-Type": "application/json"}
    if OTLP_AUTH:
        headers["Authorization"] = OTLP_AUTH
    req = urllib.request.Request(OTLP_ENDPOINT, data=body, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            code = resp.getcode()
            raw = resp.read()
    except urllib.error.HTTPError as e:
        err_body = e.read().decode("utf-8", errors="replace")
        raise SystemExit(f"OTLP export failed HTTP {e.code}: {err_body}") from e
    except urllib.error.URLError as e:
        raise SystemExit(f"OTLP export failed: {e}") from e
    print(
        f"exported {len(OPS)} GraphQL spans → {OTLP_ENDPOINT} HTTP {code}"
        + (f" body={raw[:120]!r}" if raw else ""),
        flush=True,
    )


def main() -> None:
    if INTERVAL_SEC <= 0:
        export_once()
        return
    print(
        f"graphql-demo emitter → {OTLP_ENDPOINT} every {INTERVAL_SEC}s "
        f"(service={SERVICE_NAME} ns={SERVICE_NS} env={ENVIRONMENT})",
        flush=True,
    )
    while True:
        try:
            export_once()
        except SystemExit as e:
            print(str(e), file=sys.stderr, flush=True)
        time.sleep(INTERVAL_SEC)


if __name__ == "__main__":
    main()
