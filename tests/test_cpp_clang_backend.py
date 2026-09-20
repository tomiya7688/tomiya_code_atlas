from pathlib import Path

import pytest

from Src.languages.backend import ParserBackendError, ParserBackendKind
from Src.languages.cpp_backend import CppClangToolingBackend
from Src.languages.helper_backend import module_from_wire_ir


def test_cpp_clang_backend_descriptor() -> None:
    descriptor = CppClangToolingBackend.descriptor

    assert descriptor.backend_id == "cpp-clang-tooling-helper"
    assert descriptor.language == "cpp"
    assert descriptor.kind is ParserBackendKind.HELPER
    assert CppClangToolingBackend.helper_relative_path == "cpp/kadoka-cpp-backend.exe"


def test_cpp_wire_ir_preserves_semantic_facts() -> None:
    module = module_from_wire_ir(
        {
            "language": "cpp",
            "imports": ["future"],
            "entities": [
                {
                    "kind": "class",
                    "name": "Worker",
                    "line": 3,
                    "end_line": 10,
                    "bases": ["BaseWorker"],
                    "declaration_kind": "class",
                    "type_parameters": ["T"],
                    "symbol_id": "Worker<T>",
                },
                {
                    "kind": "method",
                    "name": "run",
                    "line": 5,
                    "end_line": 8,
                    "parent": "Worker",
                    "calls": ["helper"],
                    "call_sequence": ["helper", "helper"],
                    "resolved_calls": ["Worker::helper(T)", "Worker::helper(T)"],
                    "visibility": "public",
                },
            ],
            "diagnostics": [],
        }
    )

    worker = module.entities[0]
    run = module.entities[1]
    assert worker.bases == ("BaseWorker",)
    assert worker.type_parameters == ("T",)
    assert worker.symbol_id == "Worker<T>"
    assert run.call_sequence == ("helper", "helper")
    assert len(run.resolved_calls) == 2


def test_cpp_backend_reports_missing_helper(tmp_path: Path) -> None:
    backend = CppClangToolingBackend(app_root=tmp_path)

    with pytest.raises(ParserBackendError, match="helper not found"):
        backend.parse("int main() { return 0; }", "main.cpp")
