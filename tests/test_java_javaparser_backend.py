from __future__ import annotations

import json
from pathlib import Path

import pytest

from Src.languages import JavaParserSymbolSolverBackend, ParserBackendError, ParserBackendKind
from Src.languages.helper_backend import module_from_wire_ir

ROOT = Path(__file__).resolve().parents[1]


def test_java_backend_descriptor_and_manifest_match_contract() -> None:
    descriptor = JavaParserSymbolSolverBackend.descriptor
    manifest = json.loads((ROOT / "backends" / "manifest.json").read_text(encoding="utf-8"))
    entry = next(item for item in manifest["backends"] if item["backend_id"] == descriptor.backend_id)

    assert descriptor.backend_id == "java-javaparser-symbol-solver-helper"
    assert descriptor.language == "java"
    assert descriptor.kind is ParserBackendKind.HELPER
    assert entry["path"] == "java/kadoka-java-backend.jar"
    assert entry["runtime"] == "java/runtime/bin/java.exe"


def test_wire_ir_preserves_java_semantic_facts() -> None:
    module = module_from_wire_ir(
        {
            "language": "java",
            "imports": ["java.util.function.Function"],
            "entities": [
                {
                    "kind": "class",
                    "name": "Worker",
                    "line": 3,
                    "end_line": 20,
                    "bases": ["BaseWorker", "WorkContract"],
                    "declaration_kind": "class",
                    "type_parameters": ["T"],
                    "symbol_id": "fixture.Worker",
                },
                {
                    "kind": "method",
                    "name": "run",
                    "line": 5,
                    "end_line": 9,
                    "parent": "Worker",
                    "parameters": ["item"],
                    "calls": ["helper"],
                    "call_sequence": ["helper", "helper"],
                    "resolved_calls": ["fixture.Worker.helper(T)", "fixture.Worker.helper(T)"],
                },
            ],
            "diagnostics": [{"kind": "unresolved_symbol", "message": "support.Service", "line": 5}],
        }
    )

    assert module.imports == ("java.util.function.Function",)
    assert module.entities[0].type_parameters == ("T",)
    assert module.entities[1].call_sequence == ("helper", "helper")
    assert module.entities[1].resolved_calls == ("fixture.Worker.helper(T)", "fixture.Worker.helper(T)")
    assert module.diagnostics[0].kind == "unresolved_symbol"


def test_java_backend_does_not_fabricate_missing_assets(tmp_path: Path) -> None:
    backend = JavaParserSymbolSolverBackend(app_root=tmp_path)
    with pytest.raises(ParserBackendError, match="helper not found"):
        backend.parse("class Demo {}", "Demo.java")
