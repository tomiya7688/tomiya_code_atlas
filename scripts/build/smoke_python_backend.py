"""Run the frozen parser helper without relying on an installed Python runtime."""

from __future__ import annotations

import json
import os
from pathlib import Path
import subprocess
import sys


GRAMMAR_SOURCE = '''"""Full grammar smoke."""
from collections.abc import Iterable

def decorate(value: int):
    def apply(function):
        return function
    return apply

@decorate(value=1)
class Worker:
    async def run(self, /, values: Iterable[int], *, enabled: bool = True, **options) -> str:
        match values:
            case [first, *rest] if enabled and (count := len(rest)) >= 0:
                return f"{first}:{[value for value in rest]!r}:{count}"
            case _:
                return "empty"
'''

GDSCRIPT_GRAMMAR_SOURCE = '''class_name SmokeWorker extends Node
signal completed(value: int)
@export var speed: float = 1.0
var title: String:
    get:
        return "SmokeWorker"
func run(delta: float) -> void:
    move_and_slide()
'''


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
        timeout=30,
        check=False,
    )
    if completed.returncode != 0:
        raise RuntimeError(
            f"helper exited with code {completed.returncode}: {completed.stderr.strip()}"
        )
    try:
        response = json.loads(completed.stdout)
    except json.JSONDecodeError as error:
        raise RuntimeError(f"helper returned invalid JSON: {completed.stdout!r}") from error
    if not isinstance(response, dict):
        raise RuntimeError(f"helper response must be an object: {response!r}")
    return response


def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("usage: smoke_python_backend.py <helper.exe>")
    executable = Path(sys.argv[1]).resolve()
    runtime_license = executable.parent / "licenses" / "Python-LICENSE.txt"
    if not runtime_license.is_file() or runtime_license.stat().st_size == 0:
        raise RuntimeError(f"bundled CPython license is missing: {runtime_license}")
    for name in ("tree-sitter-LICENSE.txt", "tree-sitter-gdscript-LICENSE.txt"):
        license_file = executable.parent / "licenses" / name
        if not license_file.is_file() or license_file.stat().st_size == 0:
            raise RuntimeError(f"bundled GDScript parser license is missing: {license_file}")
    success = call_helper(
        executable,
        {
            "contract_version": "1",
            "request_id": "onedir-python-grammar",
            "operation": "parse",
            "language": "python",
            "source": GRAMMAR_SOURCE,
            "path": "full_grammar.py",
        },
    )
    if not success.get("ok") or success.get("request_id") != "onedir-python-grammar":
        raise RuntimeError(f"helper did not parse valid Python grammar: {success!r}")
    ir = success.get("ir")
    if not isinstance(ir, dict):
        raise RuntimeError(f"helper omitted Common IR: {success!r}")
    entities = ir.get("entities")
    if not isinstance(entities, list):
        raise RuntimeError(f"helper returned invalid entities: {ir!r}")
    worker = next((entity for entity in entities if entity.get("name") == "Worker"), None)
    run = next((entity for entity in entities if entity.get("name") == "run"), None)
    if (
        worker is None
        or run is None
        or run.get("is_async") is not True
        or worker.get("decorators") != ["decorate(value=1)"]
    ):
        raise RuntimeError(f"helper omitted expected CPython AST facts: {ir!r}")

    failure = call_helper(
        executable,
        {
            "contract_version": "1",
            "request_id": "invalid-python-syntax",
            "operation": "parse",
            "language": "python",
            "source": "def broken(:\n    pass\n",
        },
    )
    if (
        failure.get("ok") is not False
        or not isinstance(failure.get("error"), dict)
        or failure["error"].get("kind") != "unsupported_syntax"
        or "ir" in failure
    ):
        raise RuntimeError(f"invalid syntax was not normalized: {failure!r}")

    gdscript = call_helper(
        executable,
        {
            "contract_version": "1",
            "request_id": "onedir-gdscript-grammar",
            "operation": "parse",
            "language": "gdscript",
            "source": GDSCRIPT_GRAMMAR_SOURCE,
            "path": "smoke.gd",
        },
    )
    if not gdscript.get("ok") or gdscript.get("request_id") != "onedir-gdscript-grammar":
        raise RuntimeError(f"helper did not parse valid GDScript grammar: {gdscript!r}")
    gd_ir = gdscript.get("ir")
    gd_entities = gd_ir.get("entities") if isinstance(gd_ir, dict) else None
    worker = next(
        (entity for entity in gd_entities or [] if entity.get("name") == "SmokeWorker"),
        None,
    )
    speed = next(
        (entity for entity in gd_entities or [] if entity.get("name") == "speed"),
        None,
    )
    run = next(
        (entity for entity in gd_entities or [] if entity.get("name") == "run"),
        None,
    )
    if (
        worker is None
        or worker.get("bases") != ["Node"]
        or speed is None
        or speed.get("type_name") != "float"
        or run is None
        or run.get("line") != 7
        or not isinstance(gd_ir.get("signals"), list)
        or gd_ir["signals"][0].get("name") != "completed"
    ):
        raise RuntimeError(f"helper omitted expected Tree-sitter GDScript facts: {gdscript!r}")

    gd_failure = call_helper(
        executable,
        {
            "contract_version": "1",
            "request_id": "invalid-gdscript-syntax",
            "operation": "parse",
            "language": "gdscript",
            "source": "class_name Broken\nfunc broken(:\n    pass\n",
        },
    )
    if (
        gd_failure.get("ok") is not False
        or not isinstance(gd_failure.get("error"), dict)
        or gd_failure["error"].get("kind") != "unsupported_syntax"
        or "ir" in gd_failure
    ):
        raise RuntimeError(f"invalid GDScript syntax was not normalized: {gd_failure!r}")
    print("CPython and Tree-sitter one-dir parser helper grammar and error smoke passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
