"""Run the bundled C++ Clang helper without an installed compiler or SDK."""

from __future__ import annotations

import json
import os
from pathlib import Path
import subprocess
import sys


def call_helper(executable: Path, request: dict[str, object]) -> dict[str, object]:
    system_root = Path(os.environ["SystemRoot"])
    env = {
        "SystemRoot": str(system_root),
        "PATH": str(system_root / "System32"),
        "PYTHONUTF8": "1",
    }
    completed = subprocess.run(
        [str(executable)],
        input=json.dumps(request, ensure_ascii=False),
        capture_output=True,
        text=True,
        encoding="utf-8",
        cwd=executable.parent,
        env=env,
        timeout=60,
        check=False,
    )
    if completed.returncode != 0:
        raise RuntimeError(f"C++ helper exited {completed.returncode}: {completed.stderr.strip()}")
    try:
        response = json.loads(completed.stdout)
    except json.JSONDecodeError as error:
        raise RuntimeError(f"C++ helper returned invalid JSON: {completed.stdout!r}") from error
    if not isinstance(response, dict):
        raise RuntimeError(f"C++ helper response must be an object: {response!r}")
    return response


def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("usage: cpp_backend_smoke.py <helper.exe>")
    executable = Path(sys.argv[1]).resolve()
    license_path = executable.parent / "licenses" / "LLVM-LICENSE.txt"
    if not license_path.is_file() or license_path.stat().st_size == 0:
        raise RuntimeError(f"bundled LLVM license is missing: {license_path}")

    root = Path(__file__).resolve().parents[1]
    fixture = root / "tests" / "fixtures" / "backend_conformance" / "cpp.cpp"
    source = fixture.read_text(encoding="utf-8")
    parsed = call_helper(
        executable,
        {
            "contract_version": "1",
            "request_id": "onedir-cpp-translation-unit",
            "operation": "parse",
            "language": "cpp",
            "source": source,
            "path": str(fixture),
        },
    )
    ir = parsed.get("ir")
    entities = ir.get("entities") if isinstance(ir, dict) else None
    worker = next((item for item in entities or [] if item.get("name") == "Worker"), None)
    run = next((item for item in entities or [] if item.get("name") == "run"), None)
    if (
        not parsed.get("ok")
        or worker is None
        or worker.get("type_parameters") != ["T"]
        or worker.get("bases") != ["BaseWorker"]
        or run is None
        or run.get("call_sequence", []).count("helper") != 2
        or "support.hpp" not in ir.get("imports", [])
    ):
        raise RuntimeError(f"bundled Clang helper failed C++ conformance smoke: {parsed!r}")

    missing = call_helper(
        executable,
        {
            "contract_version": "1",
            "request_id": "missing-system-sdk",
            "operation": "parse",
            "language": "cpp",
            "source": "#include <missing_sdk/header.hpp>\nstruct StillParsed {};\n",
        },
    )
    if (
        not missing.get("ok")
        or not any(item.get("name") == "StillParsed" for item in missing.get("ir", {}).get("entities", []))
        or not any(item.get("kind") == "missing_header" for item in missing.get("ir", {}).get("diagnostics", []))
    ):
        raise RuntimeError(f"missing SDK behavior was not reported: {missing!r}")

    print("Clang C++ one-dir helper and missing-SDK diagnostic smoke passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
