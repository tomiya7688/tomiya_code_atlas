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
        return completed.returncode

    response = json.loads(completed.stdout)
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
    assert "WorkContract" in worker["bases"]
    assert interface["declaration_kind"] == "interface"
    assert run["call_sequence"].count("helper") == 2
    assert len(run["resolved_calls"]) >= 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
