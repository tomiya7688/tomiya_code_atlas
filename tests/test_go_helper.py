import json
import os
import shutil
import subprocess
from pathlib import Path

import pytest


@pytest.mark.skipif(shutil.which("go") is None, reason="Go toolchain is optional for Python development")
def test_go_stdlib_helper_emits_contract_v1_response() -> None:
    output_dir = Path(".build/backend-conformance")
    output_dir.mkdir(parents=True, exist_ok=True)
    binary = output_dir.resolve() / ("tomiya-go-backend.exe" if os.name == "nt" else "tomiya-go-backend")
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

