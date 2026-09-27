"""Regression tests for Operation / selection compatibility (#73)."""

from __future__ import annotations

from Src.process.operation_requirements import (
    OPERATION_CALL_GRAPH,
    OPERATION_CI,
    OPERATION_COMMENTS,
    OPERATION_COMPONENT_DIAGRAM,
    OPERATION_DEPLOYMENT_DIAGRAM,
    OPERATION_PACKAGE_DIAGRAM,
    OPERATION_SEQUENCE_DIAGRAM,
    compatible_operations,
    is_operation_compatible,
    resolve_operation_after_selection,
)


def test_yaml_to_python_clears_ci_operation():
    resolved = resolve_operation_after_selection(
        OPERATION_CI,
        language="python",
        has_project_folder=True,
    )
    assert resolved == OPERATION_COMMENTS
    assert not is_operation_compatible(
        OPERATION_CI,
        language="python",
        has_project_folder=True,
    )


def test_python_to_yaml_auto_selects_ci():
    resolved = resolve_operation_after_selection(
        OPERATION_COMMENTS,
        language="yaml",
        has_project_folder=True,
    )
    assert resolved == OPERATION_CI


def test_yaml_keeps_deployment_when_project_folder_open():
    resolved = resolve_operation_after_selection(
        OPERATION_DEPLOYMENT_DIAGRAM,
        language="yaml",
        has_project_folder=True,
    )
    assert resolved == OPERATION_DEPLOYMENT_DIAGRAM


def test_yaml_without_project_falls_back_from_deployment_to_ci():
    resolved = resolve_operation_after_selection(
        OPERATION_DEPLOYMENT_DIAGRAM,
        language="yaml",
        has_project_folder=False,
    )
    assert resolved == OPERATION_CI


def test_project_operations_require_folder_not_language():
    assert is_operation_compatible(
        OPERATION_PACKAGE_DIAGRAM,
        language="python",
        has_project_folder=True,
    )
    assert not is_operation_compatible(
        OPERATION_COMPONENT_DIAGRAM,
        language="gdscript",
        has_project_folder=True,
    )
    assert not is_operation_compatible(
        OPERATION_PACKAGE_DIAGRAM,
        language="python",
        has_project_folder=False,
    )


def test_python_diagram_ops_incompatible_with_non_python_source():
    for language in ("gdscript", "csharp", "cpp", "java", "go", "yaml"):
        assert not is_operation_compatible(
            OPERATION_SEQUENCE_DIAGRAM,
            language=language,
            has_project_folder=True,
        )
    assert is_operation_compatible(
        OPERATION_CALL_GRAPH,
        language="python",
        has_project_folder=False,
    )


def test_compatible_operations_for_python_include_diagrams_exclude_ci():
    allowed = compatible_operations(language="python", has_project_folder=True)
    assert OPERATION_COMMENTS in allowed
    assert OPERATION_CALL_GRAPH in allowed
    assert OPERATION_PACKAGE_DIAGRAM in allowed
    assert OPERATION_CI not in allowed


def test_compatible_operations_for_yaml_include_ci_and_optional_deployment():
    with_project = compatible_operations(language="yaml", has_project_folder=True)
    without_project = compatible_operations(language="yaml", has_project_folder=False)
    assert OPERATION_CI in with_project
    assert OPERATION_DEPLOYMENT_DIAGRAM in with_project
    assert OPERATION_CI in without_project
    assert OPERATION_DEPLOYMENT_DIAGRAM not in without_project
    assert OPERATION_COMMENTS not in with_project
