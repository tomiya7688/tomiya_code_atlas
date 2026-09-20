from __future__ import annotations

import json
from pathlib import Path


ROOT = Path("tests/fixtures/backend_conformance")
EVALUATION = Path("docs/backend-evaluations/java.json")


def test_java_conformance_fixture_covers_semantic_selection_features() -> None:
    source = (ROOT / "java.java").read_text(encoding="utf-8")

    for marker in (
        "package fixture;",
        "import java.util.concurrent.CompletableFuture;",
        "interface WorkContract<T>",
        "class Worker<T> extends BaseWorker implements WorkContract<T>",
        "T helper(T item)",
        "T helper(T item, int count)",
        "Function<T, T> nested = value -> value",
        "CompletableFuture<T> async_probe",
    ):
        assert marker in source


def test_java_backend_selection_records_helper_and_license_policy() -> None:
    evaluation = json.loads(EVALUATION.read_text(encoding="utf-8"))
    selected = evaluation["selected_backend"]
    distribution = evaluation["helper_distribution"]

    assert selected["backend_id"] == "java-javaparser-symbol-solver-helper"
    assert selected["role"] == "primary-syntax-and-semantic"
    assert selected["implementation_status"] == "integrated"
    assert selected["external_jvm_required_at_runtime"] is False
    assert selected["external_jdk_required_at_runtime"] is False
    assert selected["license"].startswith("Apache-2.0")

    assert distribution["directory"] == "backends/java/"
    assert distribution["protocol"] == "Parser Backend Contract v1 JSON"
    assert "private Java runtime" in distribution["launcher"]


def test_java_selection_prefers_semantics_over_syntax_only_backend() -> None:
    evaluation = json.loads(EVALUATION.read_text(encoding="utf-8"))
    candidates = {candidate["id"]: candidate for candidate in evaluation["candidates"]}

    assert candidates["javaparser-symbol-solver"]["status"] == "selected"
    assert "resolves declarations" in candidates["javaparser-symbol-solver"]["symbol_resolution"]
    assert candidates["tree-sitter-java"]["status"] == "fallback-candidate"
    assert candidates["tree-sitter-java"]["symbol_resolution"].startswith("syntax-only")
    assert evaluation["follow_up_required"] is False
