from __future__ import annotations

import json
from pathlib import Path

import pytest

from Src.languages import CSharpRoslynBackend, ParserBackendError, ParserBackendKind
from Src.languages.helper_backend import module_from_wire_ir


ROOT = Path(__file__).resolve().parents[1]


def test_csharp_backend_descriptor_and_manifest_match_contract() -> None:
    descriptor = CSharpRoslynBackend.descriptor
    manifest = json.loads((ROOT / "backends" / "manifest.json").read_text(encoding="utf-8"))
    entry = next(
        item
        for item in manifest["backends"]
        if item["backend_id"] == descriptor.backend_id
    )

    assert descriptor.backend_id == "csharp-roslyn-helper"
    assert descriptor.language == "csharp"
    assert descriptor.kind is ParserBackendKind.HELPER
    assert descriptor.contract_version == manifest["parser_backend_contract_version"] == "1"
    assert entry["language"] == "csharp"
    assert entry["kind"] == "helper"
    assert entry["path"] == "csharp/tomiya-csharp-backend.exe"
    assert "licenses/roslyn-LICENSE.txt" in entry["licenses"]


def test_wire_ir_preserves_roslyn_semantic_facts_without_roslyn_types() -> None:
    module = module_from_wire_ir(
        {
            "language": "csharp",
            "imports": ["System.Threading.Tasks"],
            "entities": [
                {
                    "kind": "class",
                    "name": "Worker",
                    "line": 5,
                    "end_line": 20,
                    "visibility": "public",
                    "bases": ["BaseWorker", "IWorker"],
                    "declaration_kind": "class",
                    "type_parameters": ["T"],
                    "type_constraints": ["where T : class"],
                    "symbol_id": "global::Fixture.Worker<T>",
                },
                {
                    "kind": "method",
                    "name": "run",
                    "line": 7,
                    "end_line": 10,
                    "parent": "Worker",
                    "parameters": ["item"],
                    "calls": ["helper"],
                    "call_sequence": ["helper", "helper"],
                    "resolved_calls": [
                        "Fixture.Worker<T>.helper(T)",
                        "Fixture.Worker<T>.helper(T)",
                    ],
                    "visibility": "public",
                    "declaration_kind": "method",
                    "is_async": False,
                },
            ],
            "diagnostics": [
                {"kind": "warning", "message": "unresolved external reference", "line": 1}
            ],
        }
    )

    worker = module.entities[0]
    run = module.entities[1]

    assert module.language == "csharp"
    assert module.imports == ("System.Threading.Tasks",)
    assert worker.bases == ("BaseWorker", "IWorker")
    assert worker.type_parameters == ("T",)
    assert worker.type_constraints == ("where T : class",)
    assert worker.symbol_id == "global::Fixture.Worker<T>"
    assert run.call_sequence == ("helper", "helper")
    assert run.resolved_calls == (
        "Fixture.Worker<T>.helper(T)",
        "Fixture.Worker<T>.helper(T)",
    )
    assert module.diagnostics[0].kind == "warning"


def test_csharp_backend_does_not_fabricate_missing_helper(tmp_path: Path) -> None:
    backend = CSharpRoslynBackend(app_root=tmp_path)

    with pytest.raises(ParserBackendError, match="helper not found"):
        backend.parse("public class Demo {}", "Demo.cs")
