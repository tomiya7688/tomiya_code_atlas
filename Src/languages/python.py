"""Python AST adapter."""

from __future__ import annotations

import ast

from Src.analyzers.ir import (
    CodeEntity,
    EntityKind,
    ModuleIR,
    ObjectInstanceIR,
    StateMachineIR,
    StateTransitionIR,
    Visibility,
)
from Src.analyzers.ir_queries import qualified_name


class PythonLanguageAdapter:
    """Convert Python source into Tomiya Code Atlas' common IR."""

    language = "python"

    def parse(self, source: str) -> ModuleIR:
        tree = ast.parse(source)
        entities: list[CodeEntity] = []
        self._collect(tree.body, source, entities, parent=None, parent_kind=None)
        objects = self._collect_objects(tree, source)
        state_machines = self._collect_state_machines(tree, source)
        return ModuleIR(
            language=self.language,
            entities=entities,
            objects=objects,
            state_machines=state_machines,
        )

    def _collect(
        self,
        nodes: list[ast.stmt],
        source: str,
        entities: list[CodeEntity],
        parent: str | None,
        parent_kind: EntityKind | None,
    ) -> None:
        for node in nodes:
            if isinstance(node, ast.ClassDef):
                entity = self._class_entity(node, source, parent)
                entities.append(entity)
                self._collect(
                    node.body,
                    source,
                    entities,
                    parent=qualified_name(entity),
                    parent_kind=EntityKind.CLASS,
                )
            elif isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
                kind = (
                    EntityKind.METHOD
                    if parent_kind is EntityKind.CLASS
                    else EntityKind.FUNCTION
                )
                entity = self._function_entity(node, source, parent, kind)
                entities.append(entity)
                self._collect(
                    node.body,
                    source,
                    entities,
                    parent=qualified_name(entity),
                    parent_kind=kind,
                )

    def _class_entity(
        self, node: ast.ClassDef, source: str, parent: str | None
    ) -> CodeEntity:
        return CodeEntity(
            kind=EntityKind.CLASS,
            name=node.name,
            line=node.lineno,
            end_line=node.end_lineno or node.lineno,
            indent=node.col_offset,
            parent=parent,
            docstring=ast.get_docstring(node, clean=False),
            decorators=tuple(self._expr_text(item, source) for item in node.decorator_list),
            visibility=self._visibility(node.name),
            bases=tuple(self._expr_text(item, source) for item in node.bases),
        )

    def _function_entity(
        self,
        node: ast.FunctionDef | ast.AsyncFunctionDef,
        source: str,
        parent: str | None,
        kind: EntityKind,
    ) -> CodeEntity:
        parameters = [argument.arg for argument in node.args.posonlyargs]
        parameters.extend(argument.arg for argument in node.args.args)
        if node.args.vararg:
            parameters.append(f"*{node.args.vararg.arg}")
        parameters.extend(argument.arg for argument in node.args.kwonlyargs)
        if node.args.kwarg:
            parameters.append(f"**{node.args.kwarg.arg}")

        call_sequence = self._function_call_sequence(node)
        return CodeEntity(
            kind=kind,
            name=node.name,
            line=node.lineno,
            end_line=node.end_lineno or node.lineno,
            indent=node.col_offset,
            parent=parent,
            docstring=ast.get_docstring(node, clean=False),
            parameters=tuple(parameters),
            decorators=tuple(self._expr_text(item, source) for item in node.decorator_list),
            calls=tuple(dict.fromkeys(call_sequence)),
            call_sequence=call_sequence,
            visibility=self._visibility(node.name),
        )

    @classmethod
    def _function_call_sequence(
        cls, node: ast.FunctionDef | ast.AsyncFunctionDef
    ) -> tuple[str, ...]:
        """Collect calls in source order for one lexical function scope only."""

        calls: list[str] = []

        class ScopeVisitor(ast.NodeVisitor):
            def visit_Call(self, call: ast.Call) -> None:
                name = cls._call_name(call.func)
                if name:
                    calls.append(name)
                self.generic_visit(call)

            def visit_FunctionDef(self, nested: ast.FunctionDef) -> None:
                return None

            def visit_AsyncFunctionDef(self, nested: ast.AsyncFunctionDef) -> None:
                return None

            def visit_ClassDef(self, nested: ast.ClassDef) -> None:
                return None

            def visit_Lambda(self, nested: ast.Lambda) -> None:
                return None

        visitor = ScopeVisitor()
        for statement in node.body:
            visitor.visit(statement)
        return tuple(calls)

    @classmethod
    def _collect_objects(cls, tree: ast.Module, source: str) -> list[ObjectInstanceIR]:
        """Collect deterministic static object construction/value/reference facts."""

        class_names = {node.name for node in ast.walk(tree) if isinstance(node, ast.ClassDef)}
        result: list[ObjectInstanceIR] = []

        def scan_scope(nodes: list[ast.stmt], scope: str | None) -> None:
            builders: dict[str, dict[str, object]] = {}

            def ensure(name: str, type_name: str, line: int) -> dict[str, object]:
                existing = builders.get(name)
                if existing is not None:
                    return existing
                created: dict[str, object] = {
                    "name": name,
                    "type_name": type_name,
                    "line": line,
                    "values": {},
                    "references": {},
                }
                builders[name] = created
                return created

            def construct(target: ast.AST, value: ast.AST, line: int) -> bool:
                target_name = cls._call_name(target)
                type_name = cls._constructor_name(value, class_names)
                if not target_name or not type_name:
                    return False
                item = ensure(target_name, type_name, line)
                if isinstance(value, ast.Call):
                    values = item["values"]
                    assert isinstance(values, dict)
                    for index, argument in enumerate(value.args, start=1):
                        literal = cls._literal_text(argument)
                        if literal is not None:
                            values[f"arg{index}"] = literal
                    for keyword in value.keywords:
                        if keyword.arg is None:
                            continue
                        literal = cls._literal_text(keyword.value)
                        if literal is not None:
                            values[keyword.arg] = literal
                if isinstance(target, ast.Attribute):
                    owner = cls._call_name(target.value)
                    if owner in builders:
                        references = builders[owner]["references"]
                        assert isinstance(references, dict)
                        references[target.attr] = target_name
                return True

            def assign_attribute(target: ast.Attribute, value: ast.AST) -> None:
                owner = cls._call_name(target.value)
                if owner not in builders:
                    return
                item = builders[owner]
                literal = cls._literal_text(value)
                if literal is not None:
                    values = item["values"]
                    assert isinstance(values, dict)
                    values[target.attr] = literal
                    return
                reference = cls._call_name(value)
                if reference:
                    references = item["references"]
                    assert isinstance(references, dict)
                    references[target.attr] = reference

            class ObjectVisitor(ast.NodeVisitor):
                def visit_Assign(self, node: ast.Assign) -> None:
                    for target in node.targets:
                        if construct(target, node.value, node.lineno):
                            continue
                        if isinstance(target, ast.Attribute):
                            assign_attribute(target, node.value)

                def visit_AnnAssign(self, node: ast.AnnAssign) -> None:
                    if node.value is None:
                        return
                    if construct(node.target, node.value, node.lineno):
                        return
                    if isinstance(node.target, ast.Attribute):
                        assign_attribute(node.target, node.value)

                def visit_FunctionDef(self, node: ast.FunctionDef) -> None:
                    return None

                def visit_AsyncFunctionDef(self, node: ast.AsyncFunctionDef) -> None:
                    return None

                def visit_ClassDef(self, node: ast.ClassDef) -> None:
                    return None

                def visit_Lambda(self, node: ast.Lambda) -> None:
                    return None

            visitor = ObjectVisitor()
            for statement in nodes:
                visitor.visit(statement)

            for item in builders.values():
                values = item["values"]
                references = item["references"]
                assert isinstance(values, dict) and isinstance(references, dict)
                result.append(
                    ObjectInstanceIR(
                        name=str(item["name"]),
                        type_name=str(item["type_name"]),
                        line=int(item["line"]),
                        scope=scope,
                        values=tuple((str(key), str(value)) for key, value in values.items()),
                        references=tuple(
                            (str(key), str(value)) for key, value in references.items()
                        ),
                    )
                )

        def recurse(nodes: list[ast.stmt], parent: str | None) -> None:
            scan_scope(nodes, parent)
            for node in nodes:
                if isinstance(node, ast.ClassDef):
                    class_scope = cls._join_scope(parent, node.name)
                    recurse(node.body, class_scope)
                elif isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
                    function_scope = cls._join_scope(parent, node.name)
                    recurse(node.body, function_scope)

        recurse(tree.body, None)
        return result

    @classmethod
    def _collect_state_machines(
        cls, tree: ast.Module, source: str
    ) -> list[StateMachineIR]:
        """Collect explicit enum-backed state machines without inferring flag states."""

        enum_states: dict[str, tuple[str, ...]] = {}
        enum_nodes: set[int] = set()
        for node in ast.walk(tree):
            if not isinstance(node, ast.ClassDef):
                continue
            bases = {cls._call_name(base).rsplit(".", 1)[-1] for base in node.bases}
            if not bases & {"Enum", "IntEnum", "StrEnum", "Flag", "IntFlag"}:
                continue
            states: list[str] = []
            for statement in node.body:
                if isinstance(statement, ast.Assign):
                    for target in statement.targets:
                        if isinstance(target, ast.Name) and not target.id.startswith("_"):
                            states.append(target.id)
                elif isinstance(statement, ast.AnnAssign):
                    if isinstance(statement.target, ast.Name) and not statement.target.id.startswith("_"):
                        states.append(statement.target.id)
            if states:
                enum_states[node.name] = tuple(dict.fromkeys(states))
                enum_nodes.add(id(node))

        if not enum_states:
            return []

        result: list[StateMachineIR] = []

        def class_methods(node: ast.ClassDef):
            return [
                item
                for item in node.body
                if isinstance(item, (ast.FunctionDef, ast.AsyncFunctionDef))
            ]

        def assignment_facts(method: ast.FunctionDef | ast.AsyncFunctionDef):
            facts: list[tuple[str, str, str, int, ast.AST]] = []

            class AssignmentVisitor(ast.NodeVisitor):
                def visit_Assign(self, node: ast.Assign) -> None:
                    member = cls._enum_member(node.value, enum_states)
                    if member:
                        enum_name, state = member
                        for target in node.targets:
                            variable = cls._state_target(target)
                            if variable:
                                facts.append((variable, enum_name, state, node.lineno, node))
                    self.generic_visit(node)

                def visit_AnnAssign(self, node: ast.AnnAssign) -> None:
                    if node.value is not None:
                        member = cls._enum_member(node.value, enum_states)
                        variable = cls._state_target(node.target)
                        if member and variable:
                            enum_name, state = member
                            facts.append((variable, enum_name, state, node.lineno, node))
                    self.generic_visit(node)

                def visit_FunctionDef(self, nested: ast.FunctionDef) -> None:
                    return None

                def visit_AsyncFunctionDef(self, nested: ast.AsyncFunctionDef) -> None:
                    return None

                def visit_ClassDef(self, nested: ast.ClassDef) -> None:
                    return None

                def visit_Lambda(self, nested: ast.Lambda) -> None:
                    return None

            visitor = AssignmentVisitor()
            for statement in method.body:
                visitor.visit(statement)
            return facts

        def scan_machine(
            owner: str,
            methods: list[ast.FunctionDef | ast.AsyncFunctionDef],
            variable: str,
            enum_name: str,
        ) -> StateMachineIR:
            initial_state: str | None = None
            transitions: list[StateTransitionIR] = []
            seen: set[tuple[str, str, str, str | None]] = set()

            for method in methods:
                if method.name == "__init__":
                    for fact_variable, fact_enum, state, _line, _node in assignment_facts(method):
                        if fact_variable == variable and fact_enum == enum_name:
                            initial_state = initial_state or state

                def add_transition(
                    target: str,
                    line: int,
                    source_state: str | None,
                    condition: str | None,
                ) -> None:
                    if method.name == "__init__" and source_state is None:
                        return
                    transition_source = source_state or "*"
                    key = (transition_source, target, method.name, condition)
                    if key in seen:
                        return
                    seen.add(key)
                    transitions.append(
                        StateTransitionIR(
                            source=transition_source,
                            target=target,
                            line=line,
                            event=method.name,
                            condition=condition,
                        )
                    )

                def scan_statements(
                    statements: list[ast.stmt],
                    source_state: str | None = None,
                    condition: str | None = None,
                ) -> None:
                    for statement in statements:
                        if isinstance(statement, ast.If):
                            detected = cls._condition_source(
                                statement.test, variable, enum_name, enum_states
                            )
                            condition_text = cls._expr_text(statement.test, source)
                            scan_statements(
                                statement.body,
                                detected or source_state,
                                condition_text if detected else condition,
                            )
                            scan_statements(statement.orelse, source_state, condition)
                            continue
                        if isinstance(statement, ast.Match):
                            subject = cls._state_target(statement.subject)
                            if subject == variable:
                                for case in statement.cases:
                                    detected = cls._match_state(case.pattern, enum_name, enum_states)
                                    guard_text = (
                                        cls._expr_text(case.guard, source)
                                        if case.guard is not None
                                        else None
                                    )
                                    scan_statements(
                                        case.body,
                                        detected or source_state,
                                        guard_text or condition,
                                    )
                                continue
                        if isinstance(statement, ast.Assign):
                            member = cls._enum_member(statement.value, enum_states)
                            if member and member[0] == enum_name:
                                for target in statement.targets:
                                    if cls._state_target(target) == variable:
                                        add_transition(
                                            member[1], statement.lineno, source_state, condition
                                        )
                        elif isinstance(statement, ast.AnnAssign) and statement.value is not None:
                            member = cls._enum_member(statement.value, enum_states)
                            if (
                                member
                                and member[0] == enum_name
                                and cls._state_target(statement.target) == variable
                            ):
                                add_transition(member[1], statement.lineno, source_state, condition)
                        for child in ast.iter_child_nodes(statement):
                            if isinstance(child, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef, ast.Lambda)):
                                continue

                scan_statements(method.body)

            states = enum_states[enum_name]
            outgoing = {item.source for item in transitions if item.source != "*"}
            incoming = {item.target for item in transitions}
            terminal_states = tuple(
                state for state in states if state in incoming and state not in outgoing
            )
            return StateMachineIR(
                owner=owner,
                state_type=enum_name,
                state_variable=variable,
                states=states,
                transitions=tuple(transitions),
                initial_state=initial_state,
                terminal_states=terminal_states,
            )

        def recurse_classes(nodes: list[ast.stmt], parent: str | None = None) -> None:
            for node in nodes:
                if not isinstance(node, ast.ClassDef):
                    continue
                owner = cls._join_scope(parent, node.name)
                if id(node) not in enum_nodes:
                    methods = class_methods(node)
                    candidates: dict[tuple[str, str], int] = {}
                    for method in methods:
                        for variable, enum_name, _state, _line, _raw in assignment_facts(method):
                            candidates[(variable, enum_name)] = candidates.get((variable, enum_name), 0) + 1
                    for variable, enum_name in sorted(candidates):
                        result.append(scan_machine(owner, methods, variable, enum_name))
                recurse_classes(node.body, owner)

        recurse_classes(tree.body)
        return result

    @classmethod
    def _enum_member(
        cls, node: ast.AST, enum_states: dict[str, tuple[str, ...]]
    ) -> tuple[str, str] | None:
        if not isinstance(node, ast.Attribute):
            return None
        enum_ref = cls._call_name(node.value)
        enum_name = enum_ref.rsplit(".", 1)[-1] if enum_ref else ""
        if enum_name in enum_states and node.attr in enum_states[enum_name]:
            return enum_name, node.attr
        return None

    @classmethod
    def _condition_source(
        cls,
        node: ast.AST,
        variable: str,
        enum_name: str,
        enum_states: dict[str, tuple[str, ...]],
    ) -> str | None:
        if not isinstance(node, ast.Compare) or len(node.ops) != 1 or len(node.comparators) != 1:
            return None
        if not isinstance(node.ops[0], (ast.Eq, ast.Is)):
            return None
        left_variable = cls._state_target(node.left)
        right_variable = cls._state_target(node.comparators[0])
        left_member = cls._enum_member(node.left, enum_states)
        right_member = cls._enum_member(node.comparators[0], enum_states)
        if left_variable == variable and right_member and right_member[0] == enum_name:
            return right_member[1]
        if right_variable == variable and left_member and left_member[0] == enum_name:
            return left_member[1]
        return None

    @classmethod
    def _match_state(
        cls,
        pattern: ast.pattern,
        enum_name: str,
        enum_states: dict[str, tuple[str, ...]],
    ) -> str | None:
        if isinstance(pattern, ast.MatchValue):
            member = cls._enum_member(pattern.value, enum_states)
            if member and member[0] == enum_name:
                return member[1]
        return None

    @classmethod
    def _state_target(cls, node: ast.AST) -> str:
        if isinstance(node, ast.Name):
            return node.id
        if isinstance(node, ast.Attribute):
            owner = cls._call_name(node.value)
            if owner in {"self", "cls"}:
                return node.attr
        return ""

    @classmethod
    def _constructor_name(cls, node: ast.AST, class_names: set[str]) -> str:
        if not isinstance(node, ast.Call):
            return ""
        name = cls._call_name(node.func)
        if not name:
            return ""
        tail = name.rsplit(".", 1)[-1]
        if tail in class_names or (tail and tail[0].isupper()):
            return name
        return ""

    @staticmethod
    def _literal_text(node: ast.AST) -> str | None:
        try:
            value = ast.literal_eval(node)
        except (ValueError, TypeError, SyntaxError):
            return None
        rendered = repr(value)
        return rendered if len(rendered) <= 80 else rendered[:77] + "..."

    @staticmethod
    def _join_scope(parent: str | None, name: str) -> str:
        return f"{parent}.{name}" if parent else name

    @staticmethod
    def _visibility(name: str) -> Visibility:
        if name.startswith("__") and not name.endswith("__"):
            return Visibility.PRIVATE
        if name.startswith("_") and not (name.startswith("__") and name.endswith("__")):
            return Visibility.PROTECTED
        return Visibility.PUBLIC

    @staticmethod
    def _expr_text(node: ast.AST, source: str) -> str:
        return ast.get_source_segment(source, node) or ""

    @classmethod
    def _call_name(cls, node: ast.AST) -> str:
        if isinstance(node, ast.Name):
            return node.id
        if isinstance(node, ast.Attribute):
            prefix = cls._call_name(node.value)
            return f"{prefix}.{node.attr}" if prefix else node.attr
        return ""
