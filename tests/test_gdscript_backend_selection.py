from __future__ import annotations

import json
from pathlib import Path


ROOT = Path("tests/fixtures/backend_conformance")
EVALUATION = Path("docs/backend-evaluations/gdscript.json")


def test_gdscript_conformance_fixture_covers_godot_specific_features() -> None:
    source = (ROOT / "gdscript.gd").read_text(encoding="utf-8")

    for marker in (
        "class_name FixtureWorker",
        "signal completed",
        'preload("res://support.gd")',
        'load("res://other_resource.tres")',
        "@export var dependency: Resource",
        "var owner_node: Node",
        "class Worker extends BaseWorker",
        "await ",
        "var nested := func",
        "Array[Variant]",
    ):
        assert marker in source


def test_gdscript_backend_selection_records_required_constraints() -> None:
    evaluation = json.loads(EVALUATION.read_text(encoding="utf-8"))
    selected = evaluation["selected_backend"]

    assert selected["backend_id"] == "gdscript-tree-sitter"
    assert selected["role"] == "primary-syntax"
    assert selected["implementation_status"] == "selected-not-yet-integrated"
    assert selected["license"] == "MIT"
    assert selected["external_godot_runtime_required"] is False
    assert evaluation["follow_up_required"] is True

    required = set(evaluation["required_syntax_features"])
    assert {
        "class_name",
        "signal",
        "preload",
        "load",
        "annotations",
        "typed containers",
        "await",
        "lambda",
    } <= required


def test_gdscript_selection_keeps_project_semantics_outside_tree_sitter() -> None:
    evaluation = json.loads(EVALUATION.read_text(encoding="utf-8"))
    semantic = evaluation["semantic_strategy"]

    assert semantic["project_resolver"] == "Tomiya-owned Godot project semantic resolver"
    assert semantic["godot_engine_helper"] == "not selected initially"
    assert "res:// path resolution" in semantic["responsibilities"]
    assert "class_name registry" in semantic["responsibilities"]

    tree_sitter = next(
        candidate
        for candidate in evaluation["candidates"]
        if candidate["id"] == "tree-sitter-gdscript"
    )
    assert tree_sitter["symbol_resolution"] == "syntax-only; requires Tomiya semantic/project resolver"
