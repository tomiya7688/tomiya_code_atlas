"""Parser Backend Contract v1 entry point backed by Clang's C++ parser."""

from __future__ import annotations

import json
import os
from pathlib import Path
import sys
from typing import Any

ROOT = Path(__file__).resolve().parents[2]
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))

from clang import cindex  # noqa: E402

from Src.analyzers.ir import CodeEntity, DiagnosticIR, EntityKind, ModuleIR, Visibility  # noqa: E402


BACKEND_ID = "cpp-libclang-helper"
CONTRACT_VERSION = "1"


def handle_request(request: Any) -> dict[str, Any]:
    request_id = request.get("request_id") if isinstance(request, dict) else None
    envelope: dict[str, Any] = {"contract_version": CONTRACT_VERSION}
    if isinstance(request_id, str):
        envelope["request_id"] = request_id
    if (
        not isinstance(request, dict)
        or request.get("contract_version") != CONTRACT_VERSION
        or not isinstance(request_id, str)
        or not request_id
        or request.get("operation") != "parse"
        or request.get("language") != "cpp"
        or not isinstance(request.get("source"), str)
        or ("path" in request and not isinstance(request["path"], str))
    ):
        return _failure(envelope, "protocol_error", "contract_version 1, request_id, operation=parse, language=cpp, and source string are required")

    source = request["source"]
    source_path = Path(request.get("path") or "tomiya-code-atlas.cpp").resolve()
    try:
        args, working_directory = _compile_arguments(source_path)
        include_dir = str(source_path.parent if request.get("path") else Path.cwd())
        if include_dir not in args:
            args.extend(["-I", include_dir])
        if not any(item.startswith("-std=") for item in args):
            args.append("-std=c++20")
        translation_unit = cindex.Index.create().parse(
            str(source_path),
            args=args,
            unsaved_files=[(str(source_path), source)],
            options=cindex.TranslationUnit.PARSE_DETAILED_PROCESSING_RECORD,
        )
        module = _normalize_translation_unit(translation_unit, source_path)
    except Exception as error:
        return _failure(envelope, "failure", str(error) or error.__class__.__name__)

    parse_errors = [
        diagnostic
        for diagnostic in translation_unit.diagnostics
        if diagnostic.severity >= cindex.Diagnostic.Error
        and "parse issue" in str(getattr(diagnostic, "category_name", "")).lower()
    ]
    if parse_errors:
        diagnostic = parse_errors[0]
        line = diagnostic.location.line if diagnostic.location else None
        return _failure(
            envelope,
            "unsupported_syntax",
            str(diagnostic),
            line=line,
        )

    envelope.update(ok=True, ir={"schema_version": "1", **_module_dict(module)})
    return envelope


def _compile_arguments(source_path: Path) -> tuple[list[str], Path]:
    """Use a nearby compile_commands.json when it contains this translation unit."""
    for directory in (source_path.parent, *source_path.parent.parents):
        if not (directory / "compile_commands.json").is_file():
            continue
        database = cindex.CompilationDatabase.fromDirectory(str(directory))
        commands = database.getCompileCommands(str(source_path))
        if commands is None:
            continue
        for command in commands:
            working_directory = Path(command.directory).resolve()
            values = list(command.arguments)
            args = _normalize_compile_command(values, source_path, working_directory)
            return args, working_directory
    return ["-x", "c++", "-fsyntax-only"], Path.cwd()


def _normalize_compile_command(
    values: list[str], source_path: Path, working_directory: Path
) -> list[str]:
    if values:
        values = values[1:]
    result: list[str] = []
    skip_next = False
    output_flags = {"-o", "-MF", "-MT", "-MQ", "/Fo", "/Fd", "/Fe"}
    for value in values:
        if skip_next:
            skip_next = False
            continue
        if value in output_flags:
            skip_next = True
            continue
        if value in {"-c", "/c", "-MD", "-MMD", "-MP", "/nologo"}:
            continue
        if value.lower().endswith((".cpp", ".cc", ".cxx", ".c++")):
            try:
                if (working_directory / value).resolve() == source_path:
                    continue
            except OSError:
                pass
        if value in {"-I", "-isystem", "-iquote"}:
            result.append(value)
            continue
        if result and result[-1] in {"-I", "-isystem", "-iquote"}:
            include_path = Path(value)
            result.append(str(include_path if include_path.is_absolute() else working_directory / include_path))
            continue
        if value.startswith("/I"):
            include_path = Path(value[2:])
            result.extend(["-I", str(include_path if include_path.is_absolute() else working_directory / include_path)])
            continue
        if value.startswith("/D"):
            result.append("-D" + value[2:])
            continue
        if value.startswith("/std:"):
            result.append("-std=" + value[5:])
            continue
        if value.startswith("/") and len(value) > 2:
            continue
        result.append(value)
    if not any(item.startswith("-std=") for item in result):
        result.append("-std=c++20")
    result.append("-fsyntax-only")
    return result


def _normalize_translation_unit(translation_unit: Any, source_path: Path) -> ModuleIR:
    module = ModuleIR(language="cpp")
    source_key = os.path.normcase(str(source_path.resolve()))
    def belongs_to_source(cursor: Any) -> bool:
        file = cursor.location.file
        return bool(file and os.path.normcase(str(Path(file.name).resolve())) == source_key)

    def visit(cursor: Any, parent: str | None = None) -> None:
        kind = cursor.kind
        if not belongs_to_source(cursor):
            return

        if kind == cindex.CursorKind.INCLUSION_DIRECTIVE:
            reference = cursor.displayname or cursor.spelling
            if reference:
                module.imports = (*module.imports, reference)
            return
        if kind == cindex.CursorKind.NAMESPACE:
            name = cursor.spelling or "<anonymous>"
            qualified = f"{parent}::{name}" if parent else name
            module.entities.append(
                _entity(cursor, EntityKind.NAMESPACE, name, parent, visibility=Visibility.UNSPECIFIED)
            )
            for child in cursor.get_children():
                visit(child, qualified)
            return

        record_kinds = {
            cindex.CursorKind.CLASS_DECL,
            cindex.CursorKind.STRUCT_DECL,
            cindex.CursorKind.CLASS_TEMPLATE,
            cindex.CursorKind.CLASS_TEMPLATE_PARTIAL_SPECIALIZATION,
        }
        if kind in record_kinds:
            name = cursor.spelling or cursor.displayname or "<anonymous>"
            bases = tuple(
                child.type.spelling
                for child in cursor.get_children()
                if child.kind == cindex.CursorKind.CXX_BASE_SPECIFIER and child.type.spelling
            )
            entity = _entity(cursor, EntityKind.CLASS, name, parent, visibility=_visibility(cursor))
            entity.bases = bases
            entity.declaration_kind = (
                "struct" if kind == cindex.CursorKind.STRUCT_DECL else "class"
            )
            entity.type_parameters = tuple(
                child.spelling
                for child in cursor.get_children()
                if child.kind in _template_parameter_kinds()
            )
            module.entities.append(entity)
            qualified = f"{parent}::{name}" if parent else name
            for child in cursor.get_children():
                if child.kind in _template_parameter_kinds() or child.kind == cindex.CursorKind.CXX_BASE_SPECIFIER:
                    continue
                visit(child, qualified)
            return

        function_kinds = {
            cindex.CursorKind.FUNCTION_DECL,
            cindex.CursorKind.CXX_METHOD,
            cindex.CursorKind.CONSTRUCTOR,
            cindex.CursorKind.DESTRUCTOR,
            cindex.CursorKind.FUNCTION_TEMPLATE,
        }
        if kind in function_kinds:
            semantic_parent = cursor.semantic_parent
            is_method = kind in {
                cindex.CursorKind.CXX_METHOD,
                cindex.CursorKind.CONSTRUCTOR,
                cindex.CursorKind.DESTRUCTOR,
            } or semantic_parent is not None and semantic_parent.kind in {
                cindex.CursorKind.CLASS_DECL,
                cindex.CursorKind.STRUCT_DECL,
                cindex.CursorKind.CLASS_TEMPLATE,
                cindex.CursorKind.CLASS_TEMPLATE_PARTIAL_SPECIALIZATION,
            }
            entity = _entity(
                cursor,
                EntityKind.METHOD if is_method else EntityKind.FUNCTION,
                cursor.spelling or cursor.displayname,
                parent,
                visibility=_visibility(cursor),
            )
            arguments = tuple(cursor.get_arguments() or ())
            entity.parameters = tuple(argument.spelling for argument in arguments)
            entity.parameter_types = tuple(
                (argument.spelling, argument.type.spelling) for argument in arguments
            )
            result_type = cursor.result_type.spelling
            entity.return_type = result_type or None
            entity.symbol_id = cursor.get_usr() or None
            entity.type_parameters = tuple(
                child.spelling
                for child in cursor.get_children()
                if child.kind in _template_parameter_kinds()
            )
            calls = _calls(cursor)
            entity.calls = tuple(dict.fromkeys(calls))
            entity.call_sequence = calls
            module.entities.append(entity)
            return

        if kind in {cindex.CursorKind.FIELD_DECL, cindex.CursorKind.VAR_DECL}:
            semantic_parent = cursor.semantic_parent
            entity_kind = (
                EntityKind.FIELD
                if semantic_parent is not None
                and semantic_parent.kind
                in {
                    cindex.CursorKind.CLASS_DECL,
                    cindex.CursorKind.STRUCT_DECL,
                    cindex.CursorKind.CLASS_TEMPLATE,
                    cindex.CursorKind.CLASS_TEMPLATE_PARTIAL_SPECIALIZATION,
                }
                else EntityKind.PROPERTY
            )
            module.entities.append(
                _entity(
                    cursor,
                    entity_kind,
                    cursor.spelling or cursor.displayname,
                    parent,
                    visibility=_visibility(cursor),
                    type_name=cursor.type.spelling,
                )
            )
            return

        for child in cursor.get_children():
            visit(child, parent)

    for child in translation_unit.cursor.get_children():
        visit(child)
    for diagnostic in translation_unit.diagnostics:
        location = diagnostic.location
        message = str(diagnostic)
        kind = "missing_header" if "file not found" in message.lower() else "compiler_diagnostic"
        module.diagnostics.append(
            DiagnosticIR(kind, message, location.line if location else None)
        )
    module.imports = tuple(dict.fromkeys(module.imports))
    return module


def _entity(
    cursor: Any,
    kind: EntityKind,
    name: str,
    parent: str | None,
    *,
    visibility: Visibility,
    type_name: str | None = None,
) -> CodeEntity:
    return CodeEntity(
        kind=kind,
        name=name,
        line=max(1, int(cursor.extent.start.line)),
        end_line=max(1, int(cursor.extent.end.line)),
        parent=parent,
        visibility=visibility,
        type_name=type_name,
    )


def _visibility(cursor: Any) -> Visibility:
    name = getattr(cursor.access_specifier, "name", "INVALID").lower()
    return {
        "public": Visibility.PUBLIC,
        "protected": Visibility.PROTECTED,
        "private": Visibility.PRIVATE,
    }.get(name, Visibility.UNSPECIFIED)


def _template_parameter_kinds() -> set[Any]:
    return {
        cindex.CursorKind.TEMPLATE_TYPE_PARAMETER,
        cindex.CursorKind.TEMPLATE_NON_TYPE_PARAMETER,
        cindex.CursorKind.TEMPLATE_TEMPLATE_PARAMETER,
    }


def _calls(cursor: Any) -> tuple[str, ...]:
    calls: list[str] = []

    def visit(node: Any) -> None:
        if node is not cursor and node.kind in {
            cindex.CursorKind.FUNCTION_DECL,
            cindex.CursorKind.CXX_METHOD,
            cindex.CursorKind.LAMBDA_EXPR,
        }:
            return
        if node.kind == cindex.CursorKind.CALL_EXPR:
            referenced = node.referenced
            target = referenced.spelling if referenced is not None else node.spelling
            if not target:
                target = _callee_spelling(node)
            if target:
                calls.append(target)
        for child in node.get_children():
            visit(child)

    visit(cursor)
    return tuple(calls)


def _callee_spelling(call: Any) -> str:
    current = next(iter(call.get_children()), None)
    while current is not None:
        if current.spelling and current.kind in {
            cindex.CursorKind.DECL_REF_EXPR,
            cindex.CursorKind.MEMBER_REF_EXPR,
            cindex.CursorKind.OVERLOADED_DECL_REF,
        }:
            return current.spelling
        current = next(iter(current.get_children()), None)
    return ""


def _module_dict(module: ModuleIR) -> dict[str, Any]:
    from dataclasses import asdict

    return asdict(module)


def _failure(
    envelope: dict[str, Any],
    kind: str,
    message: str,
    *,
    line: int | None = None,
) -> dict[str, Any]:
    error: dict[str, Any] = {
        "kind": kind,
        "message": message,
        "backend_id": BACKEND_ID,
        "retryable": False,
    }
    if line is not None:
        error["line"] = line
    envelope.update(ok=False, error=error)
    return envelope


def main() -> int:
    try:
        request = json.load(sys.stdin)
    except (json.JSONDecodeError, UnicodeDecodeError):
        response = _failure({"contract_version": CONTRACT_VERSION}, "protocol_error", "invalid JSON request")
    else:
        response = handle_request(request)
    sys.stdout.write(json.dumps(response, ensure_ascii=False, separators=(",", ":")))
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
