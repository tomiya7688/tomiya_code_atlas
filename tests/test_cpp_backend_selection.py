from __future__ import annotations

import json
from pathlib import Path


FIXTURE = Path("tests/fixtures/backend_conformance/cpp.cpp")
EVALUATION = Path("tests/fixtures/backend-evaluations/cpp.json")


def test_cpp_fixture_exercises_template_macro_include_and_overload_cases() -> None:
    source = FIXTURE.read_text(encoding="utf-8")

    for marker in (
        '#include "support.hpp"',
        "#define TOMIYA_TOUCH(value)",
        "template <typename T>",
        "class Worker : public BaseWorker",
        "T helper(T item)",
        "T helper(T item, int count)",
        "future<T> async_probe",
        "auto nested =",
    ):
        assert marker in source


def test_cpp_backend_selection_uses_full_clang_helper() -> None:
    evaluation = json.loads(EVALUATION.read_text(encoding="utf-8"))
    selected = evaluation["selected_backend"]
    helper = evaluation["helper_distribution"]

    assert selected["backend_id"] == "cpp-libclang-helper"
    assert selected["role"] == "primary-compiler-parser-and-common-ir-structure"
    assert selected["kind"] == "helper"
    assert selected["license"] == "Apache-2.0 WITH LLVM-exception"
    assert selected["external_llvm_required_at_runtime"] is False

    assert helper["directory"] == "backends/cpp/"
    assert helper["protocol"] == "Parser Backend Contract v1 JSON"
    assert helper["external_compiler_required"] is False
    assert helper["parser_package"] == "libclang-ng==22.1.4.2"


def test_cpp_selection_records_libclang_ast_scope_and_compile_diagnostics() -> None:
    evaluation = json.loads(EVALUATION.read_text(encoding="utf-8"))
    libclang = next(
        candidate
        for candidate in evaluation["candidates"]
        if candidate["id"] == "libclang-c-api"
    )
    tree_sitter = next(
        candidate
        for candidate in evaluation["candidates"]
        if candidate["id"] == "tree-sitter-cpp"
    )

    assert libclang["status"] == "selected-primary"
    assert tree_sitter["status"] == "fallback-candidate"
    assert "common IR structural subset" in libclang["semantic_scope"]
    assert evaluation["project_strategy"]["missing_headers"].startswith(
        "emit diagnostics"
    )
    assert evaluation["follow_up_required"] is True
