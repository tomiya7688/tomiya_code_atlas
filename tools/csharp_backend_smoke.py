"""Exercise the Roslyn helper directly against the shared C# fixture."""

from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path


FIXTURE = Path("tests/fixtures/backend_conformance/csharp.cs")
PROTOCOL_FIXTURE = Path("tests/fixtures/backend_conformance/helper_protocol_cases_v1.json")


def main(argv: list[str] | None = None) -> int:
    command = list(sys.argv[1:] if argv is None else argv)
    if not command:
        raise SystemExit("usage: csharp_backend_smoke.py <helper command...>")

    protocol_fixture = json.loads(PROTOCOL_FIXTURE.read_text(encoding="utf-8"))
    protocol_case = protocol_fixture["cases"]["csharp"]
    protocol_request = {
        "contract_version": protocol_fixture["contract_version"],
        **protocol_case["request"],
    }
    protocol_completed = subprocess.run(
        command,
        input=json.dumps(protocol_request),
        text=True,
        encoding="utf-8",
        capture_output=True,
        check=False,
    )
    if protocol_completed.returncode != 0:
        sys.stderr.write(protocol_completed.stderr)
        return protocol_completed.returncode
    protocol_response = json.loads(protocol_completed.stdout)
    assert protocol_response["contract_version"] == protocol_fixture["contract_version"]
    assert protocol_response["request_id"] == protocol_request["request_id"]
    assert protocol_response["ok"] is True
    protocol_module = protocol_response["ir"]
    assert protocol_module["schema_version"] == protocol_fixture["schema_version"]
    assert protocol_module["language"] == "csharp"
    assert {item["name"] for item in protocol_module["entities"]} >= set(
        protocol_case["expected_entity_names"]
    )
    assert protocol_module["imports"] == protocol_case["expected_imports"]
    assert isinstance(protocol_module["diagnostics"], list)

    request = {
        "contract_version": "1",
        "request_id": "csharp-ci-smoke",
        "operation": "parse",
        "language": "csharp",
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
    assert response["request_id"] == "csharp-ci-smoke"
    assert response["ok"] is True
    module = response["ir"]
    assert module["schema_version"] == "1"
    entities = module["entities"]

    worker = next(
        entity
        for entity in entities
        if entity["kind"] == "class" and entity["name"] == "Worker"
    )
    interface = next(
        entity
        for entity in entities
        if entity["kind"] == "class" and entity["name"] == "IWorker"
    )
    runs = [
        entity
        for entity in entities
        if entity["kind"] == "method"
        and entity["name"] == "run"
        and entity.get("parent") == "Worker"
    ]
    one_arg_run = next(entity for entity in runs if len(entity["parameters"]) == 1)

    assert worker["type_parameters"] == ["T"]
    assert "BaseWorker" in worker["bases"]
    assert "IWorker" in worker["bases"]
    assert interface["declaration_kind"] == "interface"
    assert len(runs) == 2
    assert one_arg_run["call_sequence"].count("helper") == 2
    assert len(one_arg_run["resolved_calls"]) >= 2
    assert all("helper" in target for target in one_arg_run["resolved_calls"][:2])
    assert module["diagnostics"]
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
