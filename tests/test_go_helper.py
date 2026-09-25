import json
import shutil
import subprocess
from pathlib import Path

import pytest


@pytest.mark.skipif(shutil.which("go") is None, reason="Go toolchain is optional for Python development")
def test_go_stdlib_helper_emits_contract_v1_response(tmp_path: Path) -> None:
    binary = tmp_path / ("tomiya-go-backend.exe" if __import__("os").name == "nt" else "tomiya-go-backend")
    subprocess.run(["go", "build", "-o", str(binary), "."], cwd="backends/go", check=True)
    request = {
        "contract_version": "1",
        "request_id": "test-1",
        "operation": "parse",
        "language": "go",
        "source": "package demo\ntype Worker[T any] struct{}\nfunc (w Worker[T]) Run() {}\n",
        "path": "demo.go",
    }
    result = subprocess.run([str(binary)], input=json.dumps(request), text=True, capture_output=True, check=True)
    response = json.loads(result.stdout)
    assert response["contract_version"] == "1"
    assert response["request_id"] == "test-1"
    assert response["ok"] is True
    assert {item["kind"] for item in response["ir"]["entities"]} >= {"module", "class", "method"}

