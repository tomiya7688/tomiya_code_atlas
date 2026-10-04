"""Exercise the JavaParser helper against the shared Java fixture."""

from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

FIXTURE = Path("tests/fixtures/backend_conformance/java.java")


def main(argv: list[str] | None = None) -> int:
    command = list(sys.argv[1:] if argv is None else argv)
    if not command:
        raise SystemExit("usage: java_backend_smoke.py <helper command...>")

    request = {
        "contract_version": "1",
        "request_id": "java-ci-smoke",
        "operation": "parse",
        "language": "java",
        "source": FIXTURE.read_text(encoding="utf-8"),
        "path": str(FIXTURE),
    }
    response = invoke(command, request)
    assert response["contract_version"] == "1"
    assert response["request_id"] == "java-ci-smoke"
    assert response["ok"] is True
    module = response["ir"]
    entities = module["entities"]

    worker = next(item for item in entities if item["kind"] == "class" and item["name"] == "Worker")
    interface = next(item for item in entities if item["kind"] == "class" and item["name"] == "WorkContract")
    runs = [item for item in entities if item["kind"] == "method" and item["name"] == "run"]
    run = next(item for item in runs if item.get("parent") == "Worker")

    assert worker["type_parameters"] == ["T"]
    assert "BaseWorker" in worker["bases"]
    assert any(base.startswith("WorkContract") for base in worker["bases"])
    assert interface["declaration_kind"] == "interface"
    assert run["call_sequence"].count("helper") == 2
    assert len(run["resolved_calls"]) >= 2
    assert "static java.util.Objects.requireNonNull" in module["imports"]
    assert any(item["name"] == "Pair" and item["declaration_kind"] == "record" for item in entities)
    assert any(item["name"] == "State" and item["declaration_kind"] == "enum" for item in entities)
    assert any(item["name"] == "Marker" and item["declaration_kind"] == "annotation" for item in entities)
    assert any(item["name"] == "value" and item["declaration_kind"] == "annotation_member" for item in entities)
    assert any(item["name"] == "current" and item["kind"] == "field" for item in entities)
    assert any(item["name"] == "Worker" and item["declaration_kind"] == "constructor" for item in entities)
    nested = next(item for item in entities if item["name"] == "Nested")
    assert nested["parent"] == "Worker"
    run = next(item for item in entities if item["name"] == "run" and item.get("parent") == "Worker")
    assert run["parameter_types"] == [["item", "T"]]
    assert run["return_type"] == "T"

    invalid_request = {**request, "request_id": "java-invalid-smoke", "source": "class Broken { void f( }"}
    invalid_response = invoke(command, invalid_request)
    assert invalid_response["ok"] is False
    assert invalid_response["error"]["kind"] == "unsupported_syntax"
    return 0


def invoke(command: list[str], request: dict[str, object]) -> dict[str, object]:
    completed = subprocess.run(
        command,
        input=json.dumps(request),
        text=True,
        encoding="utf-8",
        capture_output=True,
        check=False,
    )
    if completed.returncode != 0:
        sys.stderr.write(completed.stderr)
        raise SystemExit(completed.returncode)
    return json.loads(completed.stdout)


if __name__ == "__main__":
    raise SystemExit(main())
