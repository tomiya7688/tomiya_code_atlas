"""Render generated static source references as Markdown."""

from __future__ import annotations

from Src.models.static_specification import (
    StaticSpecification,
    StaticSpecificationEntity,
)


def render_static_specification(document: StaticSpecification) -> str:
    lines = [f"# {document.source_name}", "", f"- 言語: {document.language}"]
    if document.module_docstring:
        lines.extend(("", document.module_docstring.strip()))

    if not document.entities:
        lines.extend(("", "宣言されたクラス、関数、メソッドはありません。"))
        return "\n".join(lines) + "\n"

    lines.extend(("", "## 宣言一覧"))
    for entity in document.entities:
        lines.extend(("", *_render_entity(entity)))
    return "\n".join(lines) + "\n"


def _render_entity(entity: StaticSpecificationEntity) -> list[str]:
    heading_level = 3 if entity.kind == "method" else 2
    heading = "#" * heading_level
    signature = _signature(entity)
    kind = entity.declaration_kind or entity.kind
    lines = [f"{heading} {kind}: `{entity.qualified_name}`", ""]
    location = f"- 定義位置: {entity.line}行目"
    if entity.end_line != entity.line:
        location += f"（{entity.end_line}行目まで）"
    lines.append(location)
    lines.append(f"- 公開範囲: {entity.visibility}")
    if entity.bases:
        lines.append(f"- 継承・実装: {', '.join(_code(value) for value in entity.bases)}")
    if signature:
        lines.append(f"- 宣言: `{signature}`")
    if entity.docstring:
        lines.extend(("", entity.docstring.strip()))
    if entity.calls:
        lines.append(f"- 呼び出し式: {', '.join(_code(value) for value in entity.calls)}")
    if entity.resolved_callees:
        lines.append(
            "- 同一ファイル内で解決した呼び出し先: "
            + ", ".join(_code(value) for value in entity.resolved_callees)
        )
    if entity.called_by:
        lines.append(f"- 呼び出し元: {', '.join(_code(value) for value in entity.called_by)}")
    return lines


def _signature(entity: StaticSpecificationEntity) -> str:
    if entity.kind not in {"function", "method"}:
        if entity.kind == "class" and entity.type_parameters:
            return f"{entity.name}[{', '.join(entity.type_parameters)}]"
        return ""
    annotations = dict(entity.parameter_types)
    source_parameters = entity.parameters
    if entity.kind == "method" and source_parameters[:1] in {("self",), ("cls",)}:
        source_parameters = source_parameters[1:]
    parameters = []
    for parameter in source_parameters:
        name = parameter.lstrip("*")
        annotation = annotations.get(name)
        parameters.append(f"{parameter}: {annotation}" if annotation else parameter)
    result = f"{entity.name}({', '.join(parameters)})"
    if entity.return_type:
        result += f" -> {entity.return_type}"
    return result


def _code(value: str) -> str:
    escaped = value.replace("`", "\\`")
    return f"`{escaped}`"
