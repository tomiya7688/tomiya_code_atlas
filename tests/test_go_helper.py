import json
import os
import shutil
import subprocess
from pathlib import Path

import pytest

from Src.languages.go_backend import GoStandardLibraryBackend


@pytest.mark.skipif(shutil.which("go") is None, reason="Go toolchain is optional for Python development")
def test_go_stdlib_helper_emits_contract_v1_response() -> None:
    output_dir = Path(".build/backend-conformance")
    helper_dir = output_dir / "backends/go"
    helper_dir.mkdir(parents=True, exist_ok=True)
    binary = helper_dir.resolve() / ("tomiya-go-backend.exe" if os.name == "nt" else "tomiya-go-backend")
    go_cache = Path(".build/go-cache").resolve()
    go_temp = Path(".build/go-temp").resolve()
    go_cache.mkdir(parents=True, exist_ok=True)
    go_temp.mkdir(parents=True, exist_ok=True)
    environment = os.environ.copy()
    environment.update({"GOCACHE": str(go_cache), "GOTMPDIR": str(go_temp)})
    subprocess.run(
        ["go", "build", "-o", str(binary), "."],
        cwd="backends/go",
        env=environment,
        check=True,
    )
    fixture = json.loads(Path("tests/fixtures/backend_conformance/helper_protocol_cases_v1.json").read_text(encoding="utf-8"))
    case = fixture["cases"]["go"]
    request = {"contract_version": fixture["contract_version"], **case["request"]}
    result = subprocess.run([str(binary)], input=json.dumps(request), text=True, capture_output=True, check=True)
    response = json.loads(result.stdout)
    assert response["contract_version"] == "1"
    assert response["request_id"] == request["request_id"]
    assert response["ok"] is True
    module = response["ir"]
    assert module["schema_version"] == fixture["schema_version"]
    assert module["language"] == "go"
    assert {item["name"] for item in module["entities"]} >= set(case["expected_entity_names"])
    assert module["imports"] == case["expected_imports"]
    assert isinstance(module["diagnostics"], list)
    run = next(item for item in module["entities"] if item["name"] == "Run")
    assert run["calls"] == ["Println"]
    assert run["call_sequence"] == ["Println"]
    assert any("fmt.Println" in value for value in run["resolved_calls"])

    fixture_source = Path("tests/fixtures/backend_conformance/go.go").read_text(encoding="utf-8")
    adapted = GoStandardLibraryBackend(app_root=output_dir.resolve()).parse(fixture_source, "fixture.go")
    assert adapted.language == "go"
    assert "context" in adapted.imports
    assert any(item.name == "Worker" and item.declaration_kind == "struct" for item in adapted.entities)
    fixture_request = {**request, "request_id": "go-conformance", "source": fixture_source, "path": "fixture.go"}
    fixture_result = subprocess.run(
        [str(binary)], input=json.dumps(fixture_request), text=True, capture_output=True, check=True
    )
    fixture_response = json.loads(fixture_result.stdout)
    assert fixture_response["ok"] is True
    entities = fixture_response["ir"]["entities"]
    worker = next(item for item in entities if item["name"] == "Worker")
    assert worker["declaration_kind"] == "struct"
    assert worker["type_parameters"] == ["T"]
    assert worker["bases"] == ["BaseWorker"]
    assert worker["docstring"] == "Worker stores a value and implements WorkContract."
    run = next(item for item in entities if item["name"] == "run" and item.get("parent") == "Worker")
    assert run["parent"] == "Worker"
    assert run["parameters"] == ["item"]
    assert run["parameter_types"] == [["item", "T"]]
    assert run["return_type"] == "T"
    assert run["call_sequence"] == ["helper", "helper"]
    assert len(run["resolved_calls"]) == 2
    assert next(item for item in entities if item["name"] == "New")["type_parameters"] == ["T"]

    project_root = Path("tests/fixtures/backend_conformance/go_project")
    for source_path in (project_root / "shared/shared.go", project_root / "app/app.go"):
        source_request = {
            **request,
            "request_id": source_path.parent.name,
            "source": source_path.read_text(encoding="utf-8"),
            "path": str(source_path),
        }
        source_result = subprocess.run(
            [str(binary)], input=json.dumps(source_request), text=True, capture_output=True, check=True
        )
        source_response = json.loads(source_result.stdout)
        assert source_response["ok"] is True
        assert source_response["ir"]["entities"][0]["name"] == source_path.parent.name

    invalid_request = {**request, "request_id": "go-invalid", "source": "package demo\nfunc f( {"}
    invalid_result = subprocess.run(
        [str(binary)], input=json.dumps(invalid_request), text=True, capture_output=True, check=True
    )
    invalid_response = json.loads(invalid_result.stdout)
    assert invalid_response["ok"] is False
    assert invalid_response["error"]["kind"] == "unsupported_syntax"


