"""Operation input requirements for GUI selection state.

Pure metadata and compatibility checks only. No source analysis and no
Tkinter dependencies. The GUI uses this to keep Operation selection aligned
with the current file / project target before Run is pressed.
"""

from __future__ import annotations

# These are stable operation IDs. Localized presentation labels belong to the UI.
OPERATION_COMMENTS = "comments"
OPERATION_CALL_GRAPH = "call_graph"
OPERATION_CLASS_DIAGRAM = "class_diagram"
OPERATION_OBJECT_DIAGRAM = "object_diagram"
OPERATION_SEQUENCE_DIAGRAM = "sequence_diagram"
OPERATION_COMMUNICATION_DIAGRAM = "communication_diagram"
OPERATION_STATE_DIAGRAM = "state_diagram"
OPERATION_PACKAGE_DIAGRAM = "package_diagram"
OPERATION_COMPONENT_DIAGRAM = "component_diagram"
OPERATION_DEPLOYMENT_DIAGRAM = "deployment_diagram"
OPERATION_TIMING_CHART = "timing_chart"
OPERATION_USE_CASE_DIAGRAM = "use_case_diagram"
OPERATION_RESPONSIBILITY = "responsibility_table"
OPERATION_DESIGN_QUALITY = "design_quality"
OPERATION_CI = "ci_graph"

ALL_OPERATIONS: tuple[str, ...] = (
    OPERATION_COMMENTS,
    OPERATION_CALL_GRAPH,
    OPERATION_CLASS_DIAGRAM,
    OPERATION_OBJECT_DIAGRAM,
    OPERATION_SEQUENCE_DIAGRAM,
    OPERATION_COMMUNICATION_DIAGRAM,
    OPERATION_STATE_DIAGRAM,
    OPERATION_PACKAGE_DIAGRAM,
    OPERATION_COMPONENT_DIAGRAM,
    OPERATION_DEPLOYMENT_DIAGRAM,
    OPERATION_TIMING_CHART,
    OPERATION_USE_CASE_DIAGRAM,
    OPERATION_RESPONSIBILITY,
    OPERATION_DESIGN_QUALITY,
    OPERATION_CI,
)

# Operations that need an opened project folder (not just a single file).
PROJECT_FOLDER_OPERATIONS: frozenset[str] = frozenset(
    {
        OPERATION_PACKAGE_DIAGRAM,
        OPERATION_COMPONENT_DIAGRAM,
        OPERATION_DEPLOYMENT_DIAGRAM,
    }
)

# Operations that currently expect Python source analysis paths.
PYTHON_SOURCE_OPERATIONS: frozenset[str] = frozenset(
    {
        OPERATION_CALL_GRAPH,
        OPERATION_CLASS_DIAGRAM,
        OPERATION_OBJECT_DIAGRAM,
        OPERATION_SEQUENCE_DIAGRAM,
        OPERATION_COMMUNICATION_DIAGRAM,
        OPERATION_STATE_DIAGRAM,
        OPERATION_TIMING_CHART,
        OPERATION_USE_CASE_DIAGRAM,
        OPERATION_RESPONSIBILITY,
        OPERATION_DESIGN_QUALITY,
    }
)

# Non-YAML source languages that support comment generation.
_COMMENT_LANGUAGES: frozenset[str] = frozenset(
    {"python", "gdscript", "csharp", "cpp", "java", "go"}
)


def is_operation_compatible(
    operation: str,
    *,
    language: str | None,
    has_project_folder: bool,
) -> bool:
    """Return whether *operation* can run against the current selection context."""
    if operation == OPERATION_DEPLOYMENT_DIAGRAM:
        return has_project_folder

    if operation in {OPERATION_PACKAGE_DIAGRAM, OPERATION_COMPONENT_DIAGRAM}:
        return language == "python" and has_project_folder

    if operation == OPERATION_CI:
        return language == "yaml"

    if operation == OPERATION_COMMENTS:
        return language in _COMMENT_LANGUAGES

    if operation in PYTHON_SOURCE_OPERATIONS:
        return language == "python"

    return False


def default_operation_for(language: str | None) -> str:
    """Safe default Operation when the current one is incompatible."""
    if language == "yaml":
        return OPERATION_CI
    return OPERATION_COMMENTS


def resolve_operation_after_selection(
    current_operation: str,
    *,
    language: str | None,
    has_project_folder: bool,
) -> str:
    """Keep a compatible Operation, otherwise switch to a safe default.

    YAML selection still prefers CI (existing UX). Deployment stays allowed on
    YAML only when a project folder is open and that Operation was already chosen.
    """
    if language == "yaml":
        if current_operation == OPERATION_DEPLOYMENT_DIAGRAM and has_project_folder:
            return current_operation
        return OPERATION_CI

    if is_operation_compatible(
        current_operation,
        language=language,
        has_project_folder=has_project_folder,
    ):
        return current_operation

    return default_operation_for(language)


def compatible_operations(
    *,
    language: str | None,
    has_project_folder: bool,
) -> tuple[str, ...]:
    """Operations that are valid for the current selection context (combobox values)."""
    return tuple(
        op
        for op in ALL_OPERATIONS
        if is_operation_compatible(
            op,
            language=language,
            has_project_folder=has_project_folder,
        )
    )
