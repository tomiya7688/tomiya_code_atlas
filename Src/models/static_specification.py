"""Passive models for generated source-code reference documents."""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True, slots=True)
class StaticSpecificationEntity:
    kind: str
    name: str
    qualified_name: str
    parent: str | None
    line: int
    end_line: int
    visibility: str
    declaration_kind: str | None = None
    bases: tuple[str, ...] = ()
    parameters: tuple[str, ...] = ()
    parameter_types: tuple[tuple[str, str], ...] = ()
    return_type: str | None = None
    type_parameters: tuple[str, ...] = ()
    docstring: str | None = None
    calls: tuple[str, ...] = ()
    resolved_callees: tuple[str, ...] = ()
    called_by: tuple[str, ...] = ()


@dataclass(frozen=True, slots=True)
class StaticSpecification:
    source_name: str
    language: str
    module_docstring: str | None
    entities: tuple[StaticSpecificationEntity, ...]
