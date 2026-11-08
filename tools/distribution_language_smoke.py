"""Verify that the packaged Go CLI parses each bundled source language."""

from __future__ import annotations

import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
FIXTURE_ROOT = ROOT / "tests/fixtures/backend_conformance"
LANGUAGE_FIXTURES = {
    "python": ("python.py", "Worker", "run", "helper"),
    "gdscript": ("gdscript.gd", "Worker", "run", "helper"),
    "csharp": ("csharp.cs", "Worker", "run", "helper"),
    "cpp": ("cpp.cpp", "Worker", "run", "helper"),
    "java": ("java.java", "Worker", "run", "helper"),
    "go": ("go.go", "Worker", "run", "helper"),
}


def run_language(
    executable: Path,
    language: str,
    source_name: str,
    class_name: str,
    caller_suffix: str,
    callee_suffix: str,
) -> None:
    """Run two shared operations for one parser using the packaged CLI."""
    with tempfile.TemporaryDirectory(
        dir=ROOT / ".build", prefix="distribution-language-"
    ) as temporary_directory:
        smoke_directory = Path(temporary_directory)
        source = smoke_directory / source_name
        shutil.copyfile(FIXTURE_ROOT / source_name, source)
        output_directory = smoke_directory / "output"
        output_directory.mkdir()
        command = [
            str(executable),
            "analyze",
            str(source),
            "--operation",
            "call-graph",
            "--operation",
            "class-diagram",
            "--output-dir",
            str(output_directory),
        ]
        try:
            result = subprocess.run(
                command,
                cwd=executable.parent,
                capture_output=True,
                text=True,
                encoding="utf-8",
                timeout=90,
                check=False,
            )
        except subprocess.TimeoutExpired as error:
            raise RuntimeError(
                f"{language} 配布解析が90秒以内に完了しませんでした。"
            ) from error
        if result.returncode != 0:
            raise RuntimeError(
                f"{language} 配布解析に失敗しました: {result.stderr.strip()}"
            )

        call_graph = read_output(
            output_directory / f"{source.stem}_call-graph.md", language
        )
        assert_call_relation(call_graph, language, caller_suffix, callee_suffix)
        class_diagram = read_output(
            output_directory / f"{source.stem}_class-diagram.md", language
        )
        assert_class_member(class_diagram, language, class_name, callee_suffix)


def read_output(output_path: Path, language: str) -> str:
    """Read one required Markdown output from a packaged analysis run."""
    if not output_path.is_file():
        raise RuntimeError(f"{language} 配布解析の出力がありません: {output_path.name}")
    try:
        return output_path.read_text(encoding="utf-8")
    except UnicodeDecodeError as error:
        raise RuntimeError(
            f"{language} の出力がUTF-8ではありません: {output_path.name}"
        ) from error


def assert_call_relation(
    contents: str, language: str, caller_suffix: str, callee_suffix: str
) -> None:
    """Require a Mermaid edge from the fixture caller to its helper."""
    labels = {
        node_id: label
        for node_id, label in re.findall(
            r'^\s*(n_\w+)\["([^"]+)"\]$', contents, re.MULTILINE
        )
    }
    edges = re.findall(r"^\s*(n_\w+)\s+-->\s+(n_\w+)\s*$", contents, re.MULTILINE)
    has_expected_relation = any(
        labels.get(caller, "").endswith(caller_suffix)
        and labels.get(callee, "").endswith(callee_suffix)
        for caller, callee in edges
    )
    if "flowchart LR" not in contents or not has_expected_relation:
        raise RuntimeError(
            f"{language} のcall graphに {caller_suffix} から {callee_suffix} への辺がありません。"
        )


def assert_class_member(
    contents: str, language: str, class_name: str, member_name: str
) -> None:
    """Require the fixture class and helper member inside its Mermaid block."""
    class_label = r"(?:[\w$]+(?:\.|::))*" + re.escape(class_name) + r"(?:<[^>]+>)?"
    pattern = rf'^\s*class\s+\w+\["{class_label}"\]\s*\{{\n(?P<members>.*?)^\s*\}}'
    match = re.search(pattern, contents, re.MULTILINE | re.DOTALL)
    if "classDiagram" not in contents or not match:
        raise RuntimeError(
            f"{language} のclass diagramにclass {class_name}がありません。"
        )
    member_pattern = rf"^\s*[+#~-]{re.escape(member_name)}\("
    if not re.search(member_pattern, match.group("members"), re.MULTILINE):
        raise RuntimeError(
            f"{language} のclass {class_name}にmember {member_name}がありません。"
        )


def main() -> int:
    for stream in (sys.stdout, sys.stderr):
        reconfigure = getattr(stream, "reconfigure", None)
        if reconfigure is not None:
            reconfigure(encoding="utf-8", errors="replace")
    if len(sys.argv) != 2:
        print(
            "使い方: distribution_language_smoke.py <CLI実行ファイル>", file=sys.stderr
        )
        return 2

    executable = Path(sys.argv[1]).resolve()
    if not executable.is_file():
        print(
            f"[エラー] CLI実行ファイルが見つかりません: {executable}", file=sys.stderr
        )
        return 1

    try:
        for language, fixture in LANGUAGE_FIXTURES.items():
            run_language(executable, language, *fixture)
    except RuntimeError as error:
        print(f"[エラー] {error}", file=sys.stderr)
        return 1

    print("配布CLIがPython、GDScript、C#、C++、Java、Goの解析と図出力に成功しました。")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
