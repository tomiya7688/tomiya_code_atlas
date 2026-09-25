from __future__ import annotations

import json
from pathlib import Path


ROOT = Path("tests/fixtures/backend_conformance")
EVALUATION = Path("docs/backend-evaluations/go.json")


def test_go_conformance_fixture_covers_semantic_selection_features() -> None:
    source = (ROOT / "go.go").read_text(encoding="utf-8")

    for marker in (
        "package fixture",
        '"context"',
        '"example/support"',
        "type WorkContract[T any] interface",
        "type Worker[T any] struct",
        "BaseWorker",
        "w.helper(item)",
        "go func(value T)",
        "select {",
        "nested := func(value T) T",
    ):
        assert marker in source


def test_go_backend_selection_is_self_contained() -> None:
    evaluation = json.loads(EVALUATION.read_text(encoding="utf-8"))
    selected = evaluation["selected_backend"]
    distribution = evaluation["helper_distribution"]

    assert selected["backend_id"] == "go-stdlib-types-helper"
    assert selected["role"] == "primary-syntax-and-semantic"
    assert selected["external_go_toolchain_required_at_runtime"] is False
    assert selected["external_go_runtime_required"] is False
    assert distribution["entrypoint"] == "tomiya-go-backend.exe"
    assert distribution["protocol"] == "Parser Backend Contract v1 JSON"


def test_go_packages_is_optional_because_default_loading_needs_go_command() -> None:
    evaluation = json.loads(EVALUATION.read_text(encoding="utf-8"))
    candidates = {candidate["id"]: candidate for candidate in evaluation["candidates"]}

    packages = candidates["go-packages"]
    assert packages["status"] == "optional-toolchain-aware-mode"
    assert "go command" in packages["runtime_constraint"]

    selected = candidates["go-stdlib-parser-types"]
    assert selected["status"] == "selected"
    assert "go/types" in selected["symbol_resolution"]
    assert evaluation["follow_up_required"] is True
