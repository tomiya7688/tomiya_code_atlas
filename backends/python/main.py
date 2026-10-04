"""Parser Backend Contract v1 entry point backed by CPython's grammar parser."""

from __future__ import annotations

import json
import sys
from dataclasses import asdict
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[2]
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))

from Src.languages import ParserBackendError, PythonStdlibBackend  # noqa: E402


BACKEND_ID = PythonStdlibBackend.descriptor.backend_id


def handle_request(request: Any) -> dict[str, Any]:
    """Normalize one protocol request to either a Common IR or an error."""
    request_id = request.get("request_id") if isinstance(request, dict) else None
    envelope: dict[str, Any] = {"contract_version": "1"}
    if isinstance(request_id, str):
        envelope["request_id"] = request_id

    if not isinstance(request, dict):
        return _failure(envelope, "protocol_error", "request must be a JSON object")
    if (
        request.get("contract_version") != "1"
        or not isinstance(request_id, str)
        or not request_id
        or request.get("operation") != "parse"
        or request.get("language") != "python"
        or not isinstance(request.get("source"), str)
        or ("path" in request and not isinstance(request["path"], str))
    ):
        return _failure(
            envelope,
            "protocol_error",
            "contract_version 1, request_id, operation=parse, language=python, and source string are required",
        )

    try:
        module = PythonStdlibBackend().parse(request["source"], request.get("path"))
    except ParserBackendError as error:
        return _failure(
            envelope,
            error.failure.kind.value,
            error.failure.message,
            backend_id=error.failure.backend_id,
            retryable=error.failure.retryable,
        )

    envelope.update(
        ok=True,
        ir={"schema_version": "1", **asdict(module)},
    )
    return envelope


def _failure(
    envelope: dict[str, Any],
    kind: str,
    message: str,
    *,
    backend_id: str = BACKEND_ID,
    retryable: bool = False,
) -> dict[str, Any]:
    envelope.update(
        ok=False,
        error={
            "kind": kind,
            "message": message,
            "backend_id": backend_id,
            "retryable": retryable,
        },
    )
    return envelope


def main() -> int:
    try:
        request = json.load(sys.stdin)
    except (json.JSONDecodeError, UnicodeDecodeError):
        response = _failure(
            {"contract_version": "1"}, "protocol_error", "invalid JSON request"
        )
    else:
        response = handle_request(request)
    sys.stdout.write(json.dumps(response, ensure_ascii=False, separators=(",", ":")))
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
