"""Exercise the built Go parser helper with protocol and project fixtures."""

from __future__ import annotations

import json
import os
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def request(binary: Path, *, request_id: str, source: str, path: str) -> dict[str, object]:
    payload = {
        "contract_version": "1",
        "request_id": request_id,
        "operation": "parse",
        "language": "go",
        "source": source,
        "path": path,
    }
    environment = {
        key: value
        for key, value in os.environ.items()
        if key.upper() not in {"PATH", "GOROOT", "GOPATH", "GOTOOLDIR", "GOCACHE"}
    }
    environment["PATH"] = ""
    completed = subprocess.run(
        [str(binary)],
        input=json.dumps(payload),
        text=True,
        encoding="utf-8",
        capture_output=True,
        env=environment,
        check=True,
    )
    response = json.loads(completed.stdout)
    if response.get("request_id") != request_id:
        raise AssertionError("Go helper did not preserve the request ID")
    return response


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: go_backend_smoke.py <helper-executable>", file=sys.stderr)
        return 2
    binary = Path(sys.argv[1]).resolve()
    if not binary.is_file():
        raise FileNotFoundError(binary)

    fixture = ROOT / "tests/fixtures/backend_conformance/go.go"
    result = request(binary, request_id="go-smoke", source=fixture.read_text(encoding="utf-8"), path=str(fixture))
    if result.get("ok") is not True:
        raise AssertionError(f"Go conformance fixture failed: {result.get('error')}")
    entities = result["ir"]["entities"]
    if not any(item["name"] == "Worker" and item["declaration_kind"] == "struct" for item in entities):
        raise AssertionError("Go struct declaration was not emitted")
    if not any(item["name"] == "run" and item.get("parent") == "Worker" for item in entities):
        raise AssertionError("Go receiver method was not emitted")

    project_root = ROOT / "tests/fixtures/backend_conformance/go_project"
    for package in ("shared", "app"):
        source_path = project_root / package / f"{package}.go"
        response = request(
            binary,
            request_id=f"go-{package}",
            source=source_path.read_text(encoding="utf-8"),
            path=str(source_path),
        )
        if response.get("ok") is not True:
            raise AssertionError(f"Go {package} package failed: {response.get('error')}")
        if not any(item.get("declaration_kind") == "package" and item["name"] == package for item in response["ir"]["entities"]):
            raise AssertionError(f"Go package declaration was not preserved: {package}")
        if package == "shared" and not any(item["name"] == "Normalize" for item in response["ir"]["entities"]):
            raise AssertionError("Go shared package function was not parsed")
        if package == "app":
            if response["ir"]["imports"] != ["example.test/atlasfixture/shared"]:
                raise AssertionError("Go module import path was not preserved")
            runner = next(item for item in response["ir"]["entities"] if item["name"] == "Runner")
            if runner["calls"] != ["Normalize"] or runner["return_type"] != "shared.Result":
                raise AssertionError("Go cross-package symbol reference was not preserved")

    malformed = request(binary, request_id="go-invalid", source="package x\nfunc f( {", path="invalid.go")
    if malformed.get("ok") is not False or malformed.get("error", {}).get("kind") != "unsupported_syntax":
        raise AssertionError("Malformed Go syntax did not return a normalized parser failure")
    print("Go parser helper smoke passed (protocol, AST, module packages, malformed syntax).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

