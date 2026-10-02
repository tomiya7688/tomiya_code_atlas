"""Application service for Common IR-based static source references."""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

from Src.analyzers.call_sequence import resolve_call_sequences
from Src.data.files import read_text
from Src.data.project_files import detect_language, discover_supported_files
from Src.generators.static_specification import build_static_specification
from Src.languages.python import PythonLanguageAdapter
from Src.renderers.static_specification_markdown import render_static_specification

_IGNORED_PROJECT_DIRECTORIES = {
    ".build",
    ".pytest_cache",
    ".mypy_cache",
    ".ruff_cache",
    "node_modules",
    "target",
}


@dataclass(frozen=True, slots=True)
class StaticSpecificationRequest:
    source: Path
    language: str = "python"


@dataclass(frozen=True, slots=True)
class StaticSpecificationResult:
    content: str
    format: str = "markdown"


class StaticSpecificationService:
    """Coordinate source parsing and Markdown output without UI behavior."""

    def generate(self, request: StaticSpecificationRequest) -> StaticSpecificationResult:
        language = request.language.lower().lstrip(".")
        if language not in {"py", "python"}:
            raise ValueError("静的仕様書の解析は、現在Pythonソースに対応しています。")
        if request.source.is_file():
            if detect_language(request.source) != "python":
                raise ValueError("静的仕様書の入力にはPythonの .py ファイルを指定してください。")
            files = [request.source]
            root = request.source.parent
        elif request.source.is_dir():
            root = request.source
            files = [
                path
                for path in discover_supported_files(root)
                if detect_language(path) == "python"
                and not any(
                    part.casefold() in _IGNORED_PROJECT_DIRECTORIES
                    for part in path.relative_to(root).parts[:-1]
                )
            ]
        else:
            raise FileNotFoundError(request.source)

        if not files:
            raise ValueError("指定したフォルダーにPythonソースファイルがありません。")

        rendered: list[str] = []
        for path in files:
            module = PythonLanguageAdapter().parse(read_text(path))
            source_name = path.name if request.source.is_file() else path.relative_to(root).as_posix()
            document = build_static_specification(
                module,
                source_name=source_name,
                resolved_calls=resolve_call_sequences(module),
            )
            rendered.append(render_static_specification(document))
        return StaticSpecificationResult("\n---\n\n".join(rendered))
