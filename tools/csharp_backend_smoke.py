"""Exercise the Roslyn helper directly against the shared C# fixture."""

from __future__ import annotations

import json
import os
import subprocess
import sys
from pathlib import Path


FIXTURE = Path("tests/fixtures/backend_conformance/csharp.cs")
UNITY_FIXTURE = Path("tests/fixtures/backend_conformance/csharp_unity.cs")
PROTOCOL_FIXTURE = Path("tests/fixtures/backend_conformance/helper_protocol_cases_v1.json")


def main(argv: list[str] | None = None) -> int:
    command = list(sys.argv[1:] if argv is None else argv)
    if not command:
        raise SystemExit("usage: csharp_backend_smoke.py <helper command...>")

    helper_env = None
    if os.name == "nt" and len(command) == 1 and command[0].lower().endswith(".exe"):
        windows_dir = os.environ.get("WINDIR", r"C:\Windows")
        helper_env = {
            "PATH": str(Path(windows_dir) / "System32"),
            "SYSTEMROOT": windows_dir,
            "WINDIR": windows_dir,
        }
        license_dir = Path(command[0]).resolve().parent / "licenses"
        for license_name in (
            "roslyn-LICENSE.txt",
            "dotnet-LICENSE.txt",
            "dotnet-ThirdPartyNotices.txt",
        ):
            assert (license_dir / license_name).is_file(), f"missing bundled license: {license_name}"

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
        env=helper_env,
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
        env=helper_env,
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

    unity_request = {
        "contract_version": "1",
        "request_id": "csharp-unity-smoke",
        "operation": "parse",
        "language": "csharp",
        "source": UNITY_FIXTURE.read_text(encoding="utf-8"),
        "path": str(UNITY_FIXTURE),
    }
    unity_completed = subprocess.run(
        command,
        input=json.dumps(unity_request),
        text=True,
        encoding="utf-8",
        capture_output=True,
        check=False,
        env=helper_env,
    )
    if unity_completed.returncode != 0:
        sys.stderr.write(unity_completed.stderr)
        return unity_completed.returncode
    unity_response = json.loads(unity_completed.stdout)
    assert unity_response["ok"] is True
    unity_entities = unity_response["ir"]["entities"]
    hero = next(item for item in unity_entities if item["name"] == "HeroController")
    struct = next(item for item in unity_entities if item["name"] == "MoveResult")
    target = next(item for item in unity_entities if item["name"] == "Target")
    move = next(item for item in unity_entities if item["name"] == "Move")
    speed = next(item for item in unity_entities if item["name"] == "speed")
    mode = next(item for item in unity_entities if item["name"] == "MovementMode")
    assert hero["declaration_kind"] == "class"
    assert hero["bases"] == ["MonoBehaviour"]
    assert struct["declaration_kind"] == "struct"
    assert mode["declaration_kind"] == "enum"
    assert (target["kind"], target["type_name"], target["visibility"]) == (
        "property", "Transform?", "public"
    )
    assert (speed["kind"], speed["type_name"], speed["visibility"]) == (
        "field", "float", "private"
    )
    assert move["return_type"] == "void"
    assert move["parameter_types"] == [["direction", "Vector3"]]
    assert move["line"] > 0 and move["end_line"] >= move["line"]

    invalid_request = {
        **unity_request,
        "request_id": "csharp-invalid-syntax",
        "source": "public class Invalid { public void Broken( { } }",
    }
    invalid_completed = subprocess.run(
        command,
        input=json.dumps(invalid_request),
        text=True,
        encoding="utf-8",
        capture_output=True,
        check=False,
        env=helper_env,
    )
    if invalid_completed.returncode != 0:
        sys.stderr.write(invalid_completed.stderr)
        return invalid_completed.returncode
    invalid_response = json.loads(invalid_completed.stdout)
    assert invalid_response["ok"] is False
    assert invalid_response["request_id"] == "csharp-invalid-syntax"
    assert invalid_response["error"]["kind"] == "unsupported_syntax"

    malformed_request = {
        "contract_version": "1",
        "request_id": "csharp-missing-source",
        "operation": "parse",
        "language": "csharp",
    }
    malformed_completed = subprocess.run(
        command,
        input=json.dumps(malformed_request),
        text=True,
        encoding="utf-8",
        capture_output=True,
        check=False,
        env=helper_env,
    )
    if malformed_completed.returncode != 0:
        sys.stderr.write(malformed_completed.stderr)
        return malformed_completed.returncode
    malformed_response = json.loads(malformed_completed.stdout)
    assert malformed_response["ok"] is False
    assert malformed_response["request_id"] == "csharp-missing-source"
    assert malformed_response["error"]["kind"] == "protocol_error"
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
