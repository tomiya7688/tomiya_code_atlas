"""Language-independent intermediate representation used by analyzers and generators."""

from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum


class EntityKind(str, Enum):
    MODULE = "module"
    CLASS = "class"
    FUNCTION = "function"
    METHOD = "method"


class Visibility(str, Enum):
    UNSPECIFIED = "unspecified"
    PUBLIC = "public"
    PROTECTED = "protected"
    INTERNAL = "internal"
    PRIVATE = "private"


@dataclass(slots=True)
class CodeEntity:
    """A source entity normalized from a language-specific syntax tree."""

    kind: EntityKind
    name: str
    line: int
    end_line: int
    indent: int = 0
    parent: str | None = None
    docstring: str | None = None
    parameters: tuple[str, ...] = ()
    decorators: tuple[str, ...] = ()
    calls: tuple[str, ...] = ()
    call_sequence: tuple[str, ...] = ()
    visibility: Visibility = Visibility.UNSPECIFIED
    bases: tuple[str, ...] = ()
    declaration_kind: str | None = None
    type_parameters: tuple[str, ...] = ()
    type_constraints: tuple[str, ...] = ()
    resolved_calls: tuple[str, ...] = ()
    symbol_id: str | None = None
    is_async: bool = False
    parameter_types: tuple[tuple[str, str], ...] = ()
    return_type: str | None = None


@dataclass(slots=True)
class ObjectInstanceIR:
    """Passive static object facts normalized by a language adapter."""

    name: str
    type_name: str
    line: int
    scope: str | None = None
    values: tuple[tuple[str, str], ...] = ()
    references: tuple[tuple[str, str], ...] = ()


@dataclass(slots=True)
class StateTransitionIR:
    """Passive state transition fact normalized by a language adapter."""

    source: str
    target: str
    line: int
    event: str | None = None
    condition: str | None = None


@dataclass(slots=True)
class StateMachineIR:
    """Passive explicit state-machine facts with no evaluation behavior."""

    owner: str
    state_type: str
    state_variable: str
    states: tuple[str, ...]
    transitions: tuple[StateTransitionIR, ...] = ()
    initial_state: str | None = None
    terminal_states: tuple[str, ...] = ()


@dataclass(slots=True)
class TimingEventIR:
    """One passive temporal relation observed in source order."""

    order: int
    kind: str
    line: int
    target: str | None = None
    detail: str | None = None


@dataclass(slots=True)
class TimingFlowIR:
    """Passive logical timing facts for one function or method."""

    owner: str
    is_async: bool
    events: tuple[TimingEventIR, ...] = ()


@dataclass(slots=True)
class InputEventIR:
    """Passive user-input to handler registration fact."""

    actor: str
    component: str
    event: str
    handler: str
    line: int
    label: str | None = None
    scope: str | None = None


@dataclass(slots=True)
class SignalIR:
    """Passive event/signal declaration normalized by a language adapter."""

    name: str
    line: int
    owner: str | None = None
    parameters: tuple[str, ...] = ()


@dataclass(slots=True)
class DiagnosticIR:
    """Language-independent parser/semantic diagnostic."""

    kind: str
    message: str
    line: int | None = None


@dataclass(slots=True)
class ModuleIR:
    """Normalized representation of one source module/file."""

    language: str
    entities: list[CodeEntity] = field(default_factory=list)
    objects: list[ObjectInstanceIR] = field(default_factory=list)
    state_machines: list[StateMachineIR] = field(default_factory=list)
    timing_flows: list[TimingFlowIR] = field(default_factory=list)
    input_events: list[InputEventIR] = field(default_factory=list)
    signals: list[SignalIR] = field(default_factory=list)
    diagnostics: list[DiagnosticIR] = field(default_factory=list)
    imports: tuple[str, ...] = ()
    module_docstring: str | None = None
