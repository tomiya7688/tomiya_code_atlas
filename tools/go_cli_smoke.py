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
    (source / "Dockerfile").write_text("FROM python:3.14-slim\n", encoding="utf-8")
    (source / "compose.yaml").write_text(
        "services:\n  atlas:\n    build: .\n", encoding="utf-8"
    )
    workflow = source / ".github" / "workflows" / "ci.yml"
    workflow.parent.mkdir(parents=True, exist_ok=True)
    workflow.write_text(
        "name: CI\non: [push]\njobs:\n  verify:\n    steps:\n      - run: go test ./...\n",
        encoding="utf-8",
    )
    python_source = source / "atlas_ui.py"
    python_source_text = """from enum import Enum
import asyncio
import tkinter as tk
from tkinter import ttk

class State(Enum):
    IDLE = "idle"
    RUNNING = "running"

class Repository:
    pass

class Service:
    def __init__(self):
        self.state = State.IDLE

    def start(self):
        if self.state == State.IDLE:
            self.state = State.RUNNING

def build_service():
    repository = Repository(path="atlas.db")
    service = Service()
    service.repository = repository
    return service

async def refresh():
    await asyncio.sleep(1)

def on_refresh():
    asyncio.create_task(refresh())

root = tk.Tk()
button = ttk.Button(text="Refresh", command=on_refresh)
"""
    python_source.write_text(python_source_text, encoding="utf-8")
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
        "sequence-diagram",
        "--operation",
        "comments",
        "--operation",
        "static-spec",
        "--operation",
        "design-quality",
        "--operation",
        "object-diagram",
        "--operation",
        "communication-diagram",
        "--operation",
        "state-diagram",
        "--operation",
        "timing-chart",
        "--operation",
        "use-case-diagram",
        "--operation",
        "responsibility-table",
        "--operation",
        "package-diagram",
        "--operation",
        "component-diagram",
        "--operation",
        "deployment-diagram",
        "--operation",
        "ci-graph",
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
        "go_project_call-graph.md": (
            "flowchart LR",
            "Runner",
            "n_Runner --> n_Normalize",
        ),
        "go_project_class-diagram.md": ("classDiagram", "Result"),
        "go_project_sequence-diagram.md": ("sequenceDiagram", "Runner"),
        "go_project_object-diagram.md": ("flowchart LR", "repository : Repository"),
        "go_project_communication-diagram.md": ("flowchart LR", "Runner"),
        "go_project_state-diagram.md": ("stateDiagram-v2", "IDLE --> RUNNING"),
        "go_project_timing-chart.md": ("sequenceDiagram", "asyncio.sleep"),
        "go_project_use-case-diagram.md": ("flowchart LR", "Refresh"),
        "go_project_responsibility-table.md": ("| Class | Responsibility |", "Service"),
        "go_project_package-diagram.md": ("flowchart LR",),
        "go_project_component-diagram.md": ("flowchart LR",),
        "go_project_deployment-diagram.md": ("flowchart", "atlas"),
        "go_project_ci-graph.md": ("# GitHub Actions / CI図", "ci.yml"),
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
        if file_name != "go_project_deployment-diagram.md" and (
            "settings.yaml" in contents or yaml_marker in contents
        ):
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
    if not list((output_directory / "comments").rglob("*.py")):
        print(
            "[エラー] Python sourceのコメント付きコピーがありません。", file=sys.stderr
        )
        return 1
    if python_source.read_text(encoding="utf-8") != python_source_text:
        print("[エラー] CLIの解析がPython入力sourceを変更しました。", file=sys.stderr)
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

    recursion_fixture = ROOT / "tests/fixtures/distribution_e2e/sequence_recursion.go"
    recursion_source = smoke_directory / recursion_fixture.name
    shutil.copyfile(recursion_fixture, recursion_source)
    recursion_output = smoke_directory / "sequence-recursion-output"
    recursion_output.mkdir(parents=True, exist_ok=False)
    recursion_command = [
        str(executable),
        "analyze",
        str(recursion_source),
        "--operation",
        "sequence-diagram",
        "--output-dir",
        str(recursion_output),
    ]
    try:
        recursion_result = subprocess.run(
            recursion_command,
            cwd=executable.parent,
            capture_output=True,
            text=True,
            encoding="utf-8",
            timeout=90,
            check=False,
        )
    except subprocess.TimeoutExpired:
        print(
            "[エラー] 再帰sequenceの解析が90秒以内に完了しませんでした。",
            file=sys.stderr,
        )
        return 1
    if recursion_result.returncode != 0:
        print("[エラー] 再帰sequenceの解析に失敗しました。", file=sys.stderr)
        print(recursion_result.stderr, file=sys.stderr)
        return 1
    recursion_markdown = recursion_output / "sequence_recursion_sequence-diagram.md"
    if not recursion_markdown.is_file():
        print(
            "[エラー] 再帰sequenceのMarkdown出力が一件ではありません。", file=sys.stderr
        )
        return 1
    recursion_contents = recursion_markdown.read_text(encoding="utf-8")
    for recursive_call in ("[recursive] alpha", "[recursive] self"):
        if recursive_call not in recursion_contents:
            print(
                f"[エラー] 配布EXE出力に再帰呼び出しがありません: {recursive_call}",
                file=sys.stderr,
            )
            return 1
    if "sequenceDiagram" not in recursion_contents:
        print(
            "[エラー] 再帰sequence出力にMermaid sequenceDiagramがありません。",
            file=sys.stderr,
        )
        return 1

    partition_fixture = ROOT / "tests/fixtures/distribution_e2e/sequence_partition.go"
    partition_source = smoke_directory / partition_fixture.name
    shutil.copyfile(partition_fixture, partition_source)
    partition_output = smoke_directory / "sequence-partition-output"
    partition_output.mkdir(parents=True, exist_ok=False)
    partition_command = [
        str(executable),
        "analyze",
        str(partition_source),
        "--operation",
        "call-graph",
        "--operation",
        "sequence-diagram",
        "--output-dir",
        str(partition_output),
    ]
    try:
        partition_result = subprocess.run(
            partition_command,
            cwd=executable.parent,
            capture_output=True,
            text=True,
            encoding="utf-8",
            timeout=90,
            check=False,
        )
    except subprocess.TimeoutExpired:
        print(
            "[エラー] 系列分割の解析が90秒以内に完了しませんでした。", file=sys.stderr
        )
        return 1
    if partition_result.returncode != 0:
        print("[エラー] 系列分割の解析に失敗しました。", file=sys.stderr)
        print(partition_result.stderr, file=sys.stderr)
        return 1

    call_graph_path = partition_output / "sequence_partition_call-graph.md"
    sequence_path = partition_output / "sequence_partition_sequence-diagram.md"
    if not call_graph_path.is_file() or not sequence_path.is_file():
        print(
            "[エラー] 系列分割のcall graphまたはsequence出力がありません。",
            file=sys.stderr,
        )
        return 1
    call_graph_contents = call_graph_path.read_text(encoding="utf-8")
    sequence_contents = sequence_path.read_text(encoding="utf-8")
    for contents, diagram_name in (
        (call_graph_contents, "call graph"),
        (sequence_contents, "sequence diagram"),
    ):
        if contents.count("## series_") < 3 or "## shared_" not in contents:
            print(
                f"[エラー] {diagram_name}が3系列とshared diagramに分割されていません。",
                file=sys.stderr,
            )
            return 1
        if "続き" not in contents:
            print(
                f"[エラー] {diagram_name}に分割先への境界参照がありません。",
                file=sys.stderr,
            )
            return 1
        for function_name in (
            "AlphaStart",
            "AlphaMiddle",
            "AlphaWrite",
            "BetaStart",
            "BetaMiddle",
            "BetaWrite",
            "GammaStart",
            "GammaMiddle",
            "GammaWrite",
            "SharedLogger",
        ):
            if function_name not in contents:
                print(
                    f"[エラー] {diagram_name}から関係先が欠落しています: {function_name}",
                    file=sys.stderr,
                )
                return 1

    expected_edges = (
        "n_AlphaStart --> n_AlphaMiddle",
        "n_AlphaMiddle --> n_AlphaWrite",
        "n_BetaStart --> n_BetaMiddle",
        "n_BetaMiddle --> n_BetaWrite",
        "n_GammaStart --> n_GammaMiddle",
        "n_GammaMiddle --> n_GammaWrite",
    )
    if any(edge not in call_graph_contents for edge in expected_edges):
        print(
            "[エラー] 系列分割後のcall graphから内部relationが欠落しています。",
            file=sys.stderr,
        )
        return 1
    expected_shared_edges = (
        "n_AlphaWrite --> n_SharedLogger",
        "n_BetaWrite --> n_SharedLogger",
        "n_GammaWrite --> n_SharedLogger",
    )
    if any(call_graph_contents.count(edge) != 2 for edge in expected_shared_edges):
        print(
            "[エラー] 3系列の共有境界relationがseriesとsharedの両方に保持されていません。",
            file=sys.stderr,
        )
        return 1
    print(
        "Go CLI smoke testに成功しました。YAML混在project scan、5種類のMarkdown出力、系列分割のrelation保持、自己・相互再帰sequence、コメント付きsource copyを確認しました。"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
