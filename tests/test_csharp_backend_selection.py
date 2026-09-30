from __future__ import annotations

import json
from pathlib import Path


FIXTURE = Path("tests/fixtures/backend_conformance/csharp.cs")
EVALUATION = Path("tests/fixtures/backend-evaluations/csharp.json")


def test_csharp_fixture_exercises_semantic_binding_cases() -> None:
    source = FIXTURE.read_text(encoding="utf-8")

    for marker in (
        "public interface IWorker<T>",
        "public class Worker<T> : BaseWorker, IWorker<T>",
        "public T run(T item)",
        "public T run(T item, int count)",
        "helper(item);",
        "async Task<T> async_probe",
        "T nested(T value)",
    ):
        assert marker in source


def test_csharp_backend_selection_uses_roslyn_helper() -> None:
    evaluation = json.loads(EVALUATION.read_text(encoding="utf-8"))
    selected = evaluation["selected_backend"]
    helper = evaluation["helper_distribution"]

    assert selected["backend_id"] == "csharp-roslyn-helper"
    assert selected["role"] == "primary-syntax-and-semantic"
    assert selected["kind"] == "helper"
    assert selected["license"] == "MIT"
    assert selected["external_sdk_required_at_runtime"] is False
    assert selected["external_dotnet_runtime_required"] is False

    assert helper["directory"] == "backends/csharp/"
    assert helper["publish_mode"] == "self-contained"
    assert helper["runtime_identifier"] == "win-x64"
    assert helper["publish_single_file"] is False
    assert helper["publish_trimmed"] is False
    assert helper["protocol"] == "Parser Backend Contract v1 JSON"


def test_csharp_selection_keeps_tree_sitter_as_syntax_fallback_only() -> None:
    evaluation = json.loads(EVALUATION.read_text(encoding="utf-8"))
    tree_sitter = next(
        candidate
        for candidate in evaluation["candidates"]
        if candidate["id"] == "tree-sitter-c-sharp"
    )

    assert tree_sitter["status"] == "fallback-candidate"
    assert tree_sitter["overload_resolution"] == "none without a separate semantic layer"
    assert evaluation["follow_up_required"] is False
