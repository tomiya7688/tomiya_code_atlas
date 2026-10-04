"""Tree-sitter based GDScript parser backend normalized to Common IR."""

from __future__ import annotations

from dataclasses import asdict

from tree_sitter import Language, Node, Parser
import tree_sitter_gdscript

from Src.analyzers.ir import (
    CodeEntity,
    DiagnosticIR,
    EntityKind,
    ModuleIR,
    SignalIR,
    Visibility,
)
from Src.languages.backend import (
    ParserBackendDescriptor,
    ParserBackendKind,
    ParserBackendError,
    ParserBackendFailure,
    ParserBackendFailureKind,
    normalize_backend_exception,
)
from Src.languages.helper_backend import JsonHelperBackend


class GDScriptTreeSitterBackend:
    """Primary GDScript syntax backend selected by issue #133."""

    descriptor = ParserBackendDescriptor(
        backend_id="gdscript-tree-sitter",
        language="gdscript",
        kind=ParserBackendKind.NATIVE,
    )

    def __init__(self) -> None:
        language = Language(tree_sitter_gdscript.language())
        self._parser = Parser(language)

    def parse(self, source: str, path: str | None = None) -> ModuleIR:
        try:
            encoded = source.encode("utf-8")
            tree = self._parser.parse(encoded)
            return _GDScriptIRBuilder(encoded).build(tree.root_node)
        except BaseException as error:
            raise normalize_backend_exception(self.descriptor, error) from error


class _GDScriptIRBuilder:
    def __init__(self, source: bytes) -> None:
        self.source = source
        self.entities: list[CodeEntity] = []
        self.signals: list[SignalIR] = []

    def build(self, root: Node) -> ModuleIR:
        script_name = self._script_class_name(root)
        top_level_base = self._top_level_base(root)

        if script_name:
            class_name_node = next(
                child
                for child in root.named_children
                if child.type == "class_name_statement"
            )
            self.entities.append(
                CodeEntity(
                    kind=EntityKind.CLASS,
                    name=script_name,
                    line=self._line(class_name_node),
                    end_line=self._end_line(root),
                    indent=self._column(class_name_node),
                    visibility=Visibility.PUBLIC,
                    bases=(top_level_base,) if top_level_base else (),
                )
            )

        for child in root.named_children:
            if child.type == "class_definition":
                self._collect_class(child, parent=None)
            elif child.type == "function_definition":
                self._collect_function(child, parent=script_name)
            elif child.type == "variable_statement":
                self._collect_variable(child, parent=script_name)
            elif child.type == "signal_statement":
                self.signals.append(self._signal(child, owner=script_name))

        return ModuleIR(
            language="gdscript",
            entities=self.entities,
            signals=self.signals,
            diagnostics=self._diagnostics(root),
            imports=self._imports(root),
        )

    def _collect_class(self, node: Node, parent: str | None) -> None:
        name_node = node.child_by_field_name("name")
        if name_node is None:
            return
        name = self._text(name_node)
        qname = f"{parent}.{name}" if parent else name
        extends = node.child_by_field_name("extends")
        base = self._extends_target(extends) if extends is not None else None
        self.entities.append(
            CodeEntity(
                kind=EntityKind.CLASS,
                name=name,
                line=self._line(node),
                end_line=self._end_line(node),
                indent=self._column(node),
                parent=parent,
                decorators=self._annotations(node),
                visibility=self._visibility(name),
                bases=(base,) if base else (),
            )
        )

        body = node.child_by_field_name("body")
        if body is None:
            return
        for child in body.named_children:
            if child.type == "class_definition":
                self._collect_class(child, parent=qname)
            elif child.type == "function_definition":
                self._collect_function(child, parent=qname)
            elif child.type == "variable_statement":
                self._collect_variable(child, parent=qname)
            elif child.type == "signal_statement":
                self.signals.append(self._signal(child, owner=qname))

    def _collect_variable(self, node: Node, parent: str | None) -> None:
        name_node = node.child_by_field_name("name")
        if name_node is None:
            return
        type_node = node.child_by_field_name("type")
        setget = node.child_by_field_name("setget")
        self.entities.append(
            CodeEntity(
                kind=EntityKind.PROPERTY if setget is not None else EntityKind.FIELD,
                name=self._text(name_node),
                line=self._line(node),
                end_line=self._end_line(node),
                indent=self._column(node),
                parent=parent,
                decorators=self._annotations(node),
                visibility=self._visibility(self._text(name_node)),
                type_name=(
                    self._text(type_node)
                    if type_node is not None and type_node.type == "type"
                    else None
                ),
            )
        )

    def _collect_function(self, node: Node, parent: str | None) -> None:
        name_node = node.child_by_field_name("name")
        if name_node is None:
            return
        name = self._text(name_node)
        body = node.child_by_field_name("body")
        sequence = self._call_sequence(body) if body is not None else ()
        params = node.child_by_field_name("parameters")
        self.entities.append(
            CodeEntity(
                kind=EntityKind.METHOD if parent else EntityKind.FUNCTION,
                name=name,
                line=self._line(node),
                end_line=self._end_line(node),
                indent=self._column(node),
                parent=parent,
                parameters=self._parameters(params) if params is not None else (),
                decorators=self._annotations(node),
                calls=tuple(dict.fromkeys(sequence)),
                call_sequence=sequence,
                visibility=self._visibility(name),
            )
        )

    def _signal(self, node: Node, owner: str | None) -> SignalIR:
        name_node = node.child_by_field_name("name")
        params = node.child_by_field_name("parameters")
        return SignalIR(
            name=self._text(name_node) if name_node is not None else "<anonymous>",
            line=self._line(node),
            owner=owner,
            parameters=self._parameters(params) if params is not None else (),
        )

    def _script_class_name(self, root: Node) -> str | None:
        for child in root.named_children:
            if child.type != "class_name_statement":
                continue
            name = child.child_by_field_name("name")
            if name is not None:
                return self._text(name)
        return None

    def _top_level_base(self, root: Node) -> str | None:
        for child in root.named_children:
            if child.type == "extends_statement":
                return self._extends_target(child)
            if child.type == "class_name_statement":
                nested = child.child_by_field_name("extends")
                if nested is not None:
                    return self._extends_target(nested)
        return None

    def _imports(self, root: Node) -> tuple[str, ...]:
        references: list[str] = []
        for node in self._walk(root):
            if node.type == "extends_statement":
                target = self._extends_target(node)
                if target and target.startswith("res://"):
                    references.append(target)
            elif node.type == "call":
                target = self._call_target(node)
                if target not in {"preload", "load"}:
                    continue
                args = node.child_by_field_name("arguments")
                string = self._first_descendant(args, "string") if args is not None else None
                if string is not None:
                    references.append(self._strip_quotes(self._text(string)))
        return tuple(dict.fromkeys(references))

    def _call_sequence(self, root: Node) -> tuple[str, ...]:
        calls: list[str] = []

        def visit(node: Node) -> None:
            if node.type in {"function_definition", "class_definition", "lambda"}:
                return
            if node.type == "call":
                target = self._call_target(node)
                if target:
                    calls.append(target)
            for child in node.named_children:
                visit(child)

        visit(root)
        return tuple(calls)

    def _call_target(self, node: Node) -> str | None:
        args = node.child_by_field_name("arguments")
        candidates = [
            child for child in node.named_children
            if child.type != "arguments"
        ]
        if not candidates:
            return None
        return " ".join(self._text(candidates[0]).split())

    def _parameters(self, node: Node) -> tuple[str, ...]:
        names: list[str] = []
        for child in node.named_children:
            if child.type == "identifier":
                names.append(self._text(child))
                continue
            identifier = self._first_descendant(child, "identifier")
            if identifier is not None:
                names.append(self._text(identifier))
        return tuple(names)

    def _annotations(self, node: Node) -> tuple[str, ...]:
        result: list[str] = []
        for child in node.named_children:
            if child.type == "annotation":
                result.append(self._text(child))
            elif child.type == "annotations":
                result.extend(
                    self._text(item)
                    for item in child.named_children
                    if item.type == "annotation"
                )
        return tuple(result)

    def _extends_target(self, node: Node | None) -> str | None:
        if node is None:
            return None
        for child in node.named_children:
            if child.type in {"type", "string"}:
                value = self._text(child)
                return self._strip_quotes(value) if child.type == "string" else value
        text = self._text(node).strip()
        if text.startswith("extends "):
            return self._strip_quotes(text[len("extends "):].strip())
        return None

    def _diagnostics(self, root: Node) -> list[DiagnosticIR]:
        result: list[DiagnosticIR] = []
        seen: set[tuple[str, int]] = set()
        for node in self._walk(root):
            is_missing = bool(getattr(node, "is_missing", False))
            if node.type != "ERROR" and not is_missing:
                continue
            line = self._line(node)
            kind = "missing_syntax" if is_missing else "syntax_error"
            key = (kind, line)
            if key in seen:
                continue
            seen.add(key)
            result.append(
                DiagnosticIR(
                    kind=kind,
                    message="GDScript syntax could not be fully parsed.",
                    line=line,
                )
            )
        return result

    def _walk(self, root: Node):
        yield root
        for child in root.named_children:
            yield from self._walk(child)

    def _first_descendant(self, root: Node | None, node_type: str) -> Node | None:
        if root is None:
            return None
        if root.type == node_type:
            return root
        for child in root.named_children:
            found = self._first_descendant(child, node_type)
            if found is not None:
                return found
        return None

    def _text(self, node: Node) -> str:
        return self.source[node.start_byte:node.end_byte].decode("utf-8")

    @staticmethod
    def _strip_quotes(value: str) -> str:
        value = value.strip()
        if len(value) >= 2 and value[0] == value[-1] and value[0] in {"'", '"'}:
            return value[1:-1]
        return value

    @staticmethod
    def _visibility(name: str) -> Visibility:
        return Visibility.PRIVATE if name.startswith("_") else Visibility.PUBLIC

    @staticmethod
    def _line(node: Node) -> int:
        point = node.start_point
        row = point.row if hasattr(point, "row") else point[0]
        return int(row) + 1

    @staticmethod
    def _end_line(node: Node) -> int:
        point = node.end_point
        row = point.row if hasattr(point, "row") else point[0]
        return int(row) + 1

    @staticmethod
    def _column(node: Node) -> int:
        point = node.start_point
        column = point.column if hasattr(point, "column") else point[1]
        return int(column)


class GDScriptHelperBackend(JsonHelperBackend):
    """Use the bundled Tree-sitter GDScript parser through Contract v1."""

    descriptor = ParserBackendDescriptor(
        backend_id="gdscript-tree-sitter",
        language="gdscript",
        kind=ParserBackendKind.HELPER,
    )
    helper_relative_path = "tomiya-python-backend/tomiya-python-backend.exe"


def parse_gdscript_request(source: str, path: str | None = None) -> dict[str, object]:
    """Parse one complete GDScript source file into its language-neutral wire IR."""
    module = GDScriptTreeSitterBackend().parse(source, path)
    if module.diagnostics:
        raise ParserBackendError(
            ParserBackendFailure(
                ParserBackendFailureKind.UNSUPPORTED_SYNTAX,
                "GDScript source contains syntax the bundled grammar could not parse completely.",
                GDScriptTreeSitterBackend.descriptor.backend_id,
            )
        )
    return asdict(module)
