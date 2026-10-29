"""Verify that the packaged Go CLI parses source and writes selected product outputs."""

from __future__ import annotations

import shutil
import subprocess
import sys
from uuid import uuid4
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def main() -> int:
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    sys.stderr.reconfigure(encoding="utf-8", errors="replace")
    if len(sys.argv) != 2:
        print("使い方: go_cli_smoke.py <CLI実行ファイル>", file=sys.stderr)
        return 2

    executable = Path(sys.argv[1]).resolve()
    if not executable.is_file():
        print(
            f"[エラー] CLI実行ファイルが見つかりません: {executable}", file=sys.stderr
        )
        return 1

    fixture = ROOT / "tests/fixtures/backend_conformance/go_project"
    smoke_directory = ROOT / ".build" / f"go-cli-smoke-{uuid4().hex}"
    source = smoke_directory / "go_project"
    shutil.copytree(fixture, source)
    yaml_marker = "atlas-yaml-configuration-sentinel"
    (source / "settings.yaml").write_text(f"name: {yaml_marker}\n", encoding="utf-8")
    output_directory = smoke_directory / "output"
    output_directory.mkdir(parents=True, exist_ok=False)
    command = [
        str(executable),
        "analyze",
        str(source),
        "--operation",
        "call-graph",
        "--operation",
        "class-diagram",
        "--operation",
        "comments",
        "--operation",
        "static-spec",
        "--operation",
        "design-quality",
        "--output-dir",
        str(output_directory),
    ]
    try:
        completed = subprocess.run(
            command,
            cwd=executable.parent,
            capture_output=True,
            text=True,
            encoding="utf-8",
            timeout=90,
            check=False,
        )
    except subprocess.TimeoutExpired:
        print("[エラー] Go CLIの解析が90秒以内に完了しませんでした。", file=sys.stderr)
        return 1
    if completed.returncode != 0:
        print("[エラー] Go CLIが解析に失敗しました。", file=sys.stderr)
        print(completed.stderr, file=sys.stderr)
        return 1

    expectations = {
        "go_project_call-graph.md": ("flowchart LR", "Runner"),
        "go_project_class-diagram.md": ("classDiagram", "Result"),
        "go_project_static-spec.md": ("## 宣言一覧", "Runner"),
        "go_project_design-quality.md": ("# 設計品質レポート", "## 指標"),
    }
    for file_name, markers in expectations.items():
        output_path = output_directory / file_name
        if not output_path.is_file():
            print(
                f"[エラー] CLIの出力fileが見つかりません: {file_name}", file=sys.stderr
            )
            return 1
        contents = output_path.read_text(encoding="utf-8")
        if "settings.yaml" in contents or yaml_marker in contents:
            print(f"[エラー] {file_name} にYAML設定が含まれています。", file=sys.stderr)
            return 1
        if any(marker not in contents for marker in markers):
            print(
                f"[エラー] {file_name} に期待した図の内容がありません。",
                file=sys.stderr,
            )
            return 1

    if list(output_directory.rglob("settings.yaml")):
        print("[エラー] YAML設定がコメント付き出力に含まれています。", file=sys.stderr)
        return 1

    if not list((output_directory / "comments").rglob("*.go")):
        print("[エラー] Go sourceのコメント付きコピーがありません。", file=sys.stderr)
        return 1

    comment_source = smoke_directory / "comment_input.go"
    original_source = b"package sample\n\nfunc Unannotated() {}\n"
    comment_source.write_bytes(original_source)
    comment_command = [
        str(executable),
        "analyze",
        str(comment_source),
        "--operation",
        "comments",
        "--output-dir",
        str(output_directory),
    ]
    try:
        comment_result = subprocess.run(
            comment_command,
            cwd=executable.parent,
            capture_output=True,
            text=True,
            encoding="utf-8",
            timeout=90,
            check=False,
        )
    except subprocess.TimeoutExpired:
        print(
            "[エラー] comments operationが90秒以内に完了しませんでした。",
            file=sys.stderr,
        )
        return 1
    if comment_result.returncode != 0:
        print("[エラー] Go CLIがcomments operationに失敗しました。", file=sys.stderr)
        print(comment_result.stderr, file=sys.stderr)
        return 1
    commented_source = output_directory / "comments" / "comment_input_commented.go"
    if not commented_source.is_file():
        print(
            "[エラー] comments operationのsource copyが見つかりません。",
            file=sys.stderr,
        )
        return 1
    if "// unannotatedの処理を行います。" not in commented_source.read_text(
        encoding="utf-8"
    ):
        print(
            "[エラー] comments operationがGo形式のcommentを挿入しませんでした。",
            file=sys.stderr,
        )
        return 1
    if comment_source.read_bytes() != original_source:
        print(
            "[エラー] comments operationが入力sourceを変更しました。", file=sys.stderr
        )
        return 1

    print(
        "Go CLI smoke testに成功しました。YAML混在のrecursive project scan、4種類のMarkdown出力、コメント付きsource copyを確認しました。"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
