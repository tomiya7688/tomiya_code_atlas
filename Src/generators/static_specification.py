"""Build a language-independent static reference document from Common IR."""

from __future__ import annotations

from Src.analyzers.call_sequence import ResolvedCall
from Src.analyzers.ir import ModuleIR
from Src.analyzers.ir_queries import qualified_name
from Src.models.static_specification import (
    StaticSpecification,
    StaticSpecificationEntity,
)


def build_static_specification(
    module: ModuleIR,
    *,
    source_name: str,
    resolved_calls: dict[str, tuple[ResolvedCall, ...]],
) -> StaticSpecification:
    """Collect declarations and resolvable same-module call relationships."""
    local_callables = set(resolved_calls)
    callers: dict[str, set[str]] = {name: set() for name in resolved_calls}
    for caller, calls in resolved_calls.items():
        for call in calls:
            if call.target in callers:
                callers[call.target].add(caller)

    entities = tuple(
        StaticSpecificationEntity(
            kind=entity.kind.value,
            name=entity.name,
            qualified_name=qualified_name(entity),
            parent=entity.parent,
            line=entity.line,
            end_line=entity.end_line,
            visibility=entity.visibility.value,
            declaration_kind=entity.declaration_kind,
            bases=entity.bases,
            parameters=entity.parameters,
            parameter_types=entity.parameter_types,
            return_type=entity.return_type,
            type_parameters=entity.type_parameters,
            docstring=entity.docstring,
            calls=tuple(
                dict.fromkeys(
                    call.raw
                    for call in resolved_calls.get(qualified_name(entity), ())
                )
            ),
            resolved_callees=tuple(
                dict.fromkeys(
                    call.target
                    for call in resolved_calls.get(qualified_name(entity), ())
                    if call.target in local_callables
                )
            ),
            called_by=tuple(sorted(callers.get(qualified_name(entity), ()))),
        )
        for entity in module.entities
    )
    return StaticSpecification(
        source_name=source_name,
        language=module.language,
        module_docstring=module.module_docstring,
        entities=entities,
    )
