"""Exercise the Clang helper against the shared C++ fixture."""

from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

FIXTURE = Path("tests/fixtures/backend_conformance/cpp.cpp")


def main(argv: list[str] | None = None) -> int:
    command = list(sys.argv[1:] if argv is None else argv)
    if not command:
        raise SystemExit("usage: cpp_backend_smoke.py <helper command...>")

    request = {
        "contract_version": "1",
        "request_id": "cpp-ci-smoke",
        "operation": "parse",
        "language": "cpp",
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
    assert response["request_id"] == "cpp-ci-smoke"
    assert response["ok"] is True
    module = response["ir"]
    entities = module["entities"]

    worker = next(item for item in entities if item["kind"] == "class" and item["name"] == "Worker")
    run = next(
        item
        for item in entities
        if item["kind"] == "method" and item["name"] == "run" and item.get("parent") == "Worker"
    )
    helpers = [item for item in entities if item["kind"] == "method" and item["name"] == "helper"]
    macro = next(
        item
        for item in entities
        if item["kind"] == "module"
        and item["name"] == "KADOKA_TOUCH"
        and item.get("declaration_kind") == "macro"
    )

    assert worker["type_parameters"] == ["T"]
    assert any("BaseWorker" in base for base in worker["bases"])
    assert len(helpers) == 2
    assert run["call_sequence"].count("helper") >= 2
    assert len(run["resolved_calls"]) >= 2
    assert macro["name"] == "KADOKA_TOUCH"
    assert "future" in module["imports"]
    assert "support.hpp" in module["imports"]
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
