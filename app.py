"""Tomiya Code Atlas application entry point."""
from __future__ import annotations
import argparse
import sys
from pathlib import Path
from typing import Any
from Src.process.application import (
    ApplicationService,
    CIRequest,
    CommentRequest,
    SourceAnalysisRequest,
)
from Src.process.call_graph_service import CallGraphAnalysisRequest, CallGraphService
from Src.process.config_service import load_config
from Src.process.deployment_service import DeploymentAnalysisRequest, DeploymentService
from Src.process.timing_service import TimingAnalysisRequest, TimingService
from Src.process.use_case_service import UseCaseAnalysisRequest, UseCaseService
from Src.data.files import write_text
from Src.version import __version__
PROJECT_NAME = "Tomiya Code Atlas"
PROJECT_VERSION = __version__
CONFIG_FILE_NAME = "tomiya-code-atlas.json"


class JapaneseArgumentParser(argparse.ArgumentParser):
    """Present argparse's user-facing help and common errors in Japanese."""

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)
        self._positionals.title = "引数"
        self._optionals.title = "オプション"
        for action in self._actions:
            if isinstance(action, argparse._HelpAction):
                action.help = "このヘルプを表示して終了"

    def format_usage(self) -> str:
        return super().format_usage().replace("usage:", "使い方:", 1)

    def format_help(self) -> str:
        return super().format_help().replace("usage:", "使い方:", 1)

    def error(self, message: str) -> None:
        if message.startswith("unrecognized arguments: "):
            message = message.replace("unrecognized arguments: ", "認識できない引数: ", 1)
        elif message.startswith("the following arguments are required: "):
            message = message.replace(
                "the following arguments are required: ", "必須の引数が指定されていません: ", 1
            )
        elif message.startswith("invalid choice: "):
            message = message.replace("invalid choice: ", "無効な選択肢です: ", 1)
            message = message.replace(" (choose from ", "（選択可能: ").replace(")", "）")
        elif message.startswith("argument ") and message.endswith(": expected one argument"):
            argument = message.removeprefix("argument ").removesuffix(": expected one argument")
            message = f"{argument} の値を指定してください。"
        elif ": invalid int value:" in message:
            argument, value = message.split(": invalid int value:", maxsplit=1)
            message = f"{argument} には整数を指定してください: {value.strip()}"
        else:
            message = f"引数の指定を確認してください: {message}"
        super().error(message)


def _runtime_root() -> Path:
    """Return the source/package root or the PyInstaller onedir root."""
    if getattr(sys, "frozen", False):
        return Path(sys.executable).resolve().parent
    return Path(__file__).resolve().parent


def _default_config_path() -> Path:
    return _runtime_root() / "config" / CONFIG_FILE_NAME


def _application_service() -> ApplicationService:
    return ApplicationService(load_config(_default_config_path()))


def _launch_gui() -> int:
    from Src.ui import launch_gui

    launch_gui(_application_service())
    return 0


def _print_outputs(result: object) -> None:
    for output in getattr(result, "outputs", ()):
        marker = "%%" if output.format == "mermaid" else "'"
        print(f"{marker} {output.name}")
        print(output.content, end="")


def _configure_cli_streams() -> None:
    """Keep Japanese CLI output usable under non-Japanese Windows locales."""
    for stream in (sys.stdout, sys.stderr):
        reconfigure = getattr(stream, "reconfigure", None)
        if reconfigure is not None:
            reconfigure(encoding="utf-8", errors="replace")


def main(argv: list[str] | None = None) -> int:
    arguments = list(sys.argv[1:] if argv is None else argv)
    if not arguments:
        return _launch_gui()
    if argv is None:
        _configure_cli_streams()

    parser = JapaneseArgumentParser(description=f"{PROJECT_NAME} のコマンドラインツール")
    parser.add_argument("--version", action="store_true", help="バージョンを表示して終了")
    sub = parser.add_subparsers(dest="command", title="コマンド", parser_class=JapaneseArgumentParser)
    sub.add_parser("gui", help="デスクトップ画面を起動")
    comment = sub.add_parser("comment", help="ソースコードへ説明コメントを生成")
    comment.add_argument("source", help="解析するソースファイル")
    comment.add_argument("--language", help="解析言語（省略時は拡張子から判定）")
    comment.add_argument("--output", help="結果を書き出すファイル")
    comment.add_argument("--in-place", action="store_true", help="入力ファイルを上書き")
    ci = sub.add_parser("ci", help="GitHub Actionsのワークフローを解析")
    ci.add_argument("source", help="解析するYAMLファイル")
    ci.add_argument("--output", help="結果を書き出すファイル")
    ci.add_argument("--check", action="store_true", help="品質上の問題を確認し、エラーがあれば失敗にする")
    deployment = sub.add_parser("deployment", help="プロジェクトの配置図を生成")
    deployment.add_argument("root", help="解析するプロジェクトフォルダー")
    deployment.add_argument("--mode", choices=("simple", "full"), default="full", help="解析範囲")
    deployment.add_argument("--output-dir", help="結果の保存先フォルダー")
    timing = sub.add_parser("timing", help="ソースコードからタイミング図を生成")
    timing.add_argument("source", help="解析するソースファイル")
    timing.add_argument("--language", help="解析言語（省略時は拡張子から判定）")
    timing.add_argument("--output-dir", help="結果の保存先フォルダー")
    use_cases = sub.add_parser("use-cases", help="GUI操作からユースケース図を生成")
    use_cases.add_argument("source", help="解析するソースファイル")
    use_cases.add_argument("--language", help="解析言語（省略時は拡張子から判定）")
    use_cases.add_argument("--output-dir", help="結果の保存先フォルダー")
    use_cases.add_argument("--max-depth", type=int, default=5, help="呼び出しをたどる最大階層")
    call_graph = sub.add_parser("call-graph", help="呼び出し関係図を生成")
    call_graph.add_argument("source", help="解析するソースファイル")
    call_graph.add_argument("--language", help="解析言語（省略時は拡張子から判定）")
    call_graph.add_argument("--output-dir", help="結果の保存先フォルダー")
    call_graph.add_argument("--fan-in-threshold", type=int, default=3, help="参照数による分割しきい値")
    call_graph.add_argument("--root", help="探索の起点")
    call_graph.add_argument("--max-depth", type=int, help="呼び出しをたどる最大階層")
    class_diagram = sub.add_parser("class-diagram", help="クラス図を生成")
    class_diagram.add_argument("source", help="解析するソースファイル")
    class_diagram.add_argument("--language", help="解析言語（省略時は拡張子から判定）")
    class_diagram.add_argument("--renderer", choices=("mermaid", "plantuml"), help="図の形式")
    class_diagram.add_argument("--output-dir", help="結果の保存先フォルダー")
    sequence_diagram = sub.add_parser("sequence-diagram", help="シーケンス図を生成")
    sequence_diagram.add_argument("source", help="解析するソースファイル")
    sequence_diagram.add_argument("--language", help="解析言語（省略時は拡張子から判定）")
    sequence_diagram.add_argument("--renderer", choices=("mermaid", "plantuml"), help="図の形式")
    sequence_diagram.add_argument("--output-dir", help="結果の保存先フォルダー")
    sequence_diagram.add_argument("--max-depth", type=int, default=8, help="呼び出しをたどる最大階層")
    sequence_diagram.add_argument("--hide-duplicate-calls", action="store_true", help="重複する呼び出しを省略")
    sequence_diagram.add_argument("--show-returns", action="store_true", help="戻り値を表示")
    backend_smoke = sub.add_parser(
        "backend-smoke", help="CI向けのbackend動作確認（開発用）"
    )
    backend_smoke.add_argument("language", choices=("gdscript", "csharp", "java"))
    backend_smoke.add_argument("--source", help="組み込み例の代わりに実際のソースファイルを解析")
    args = parser.parse_args(arguments)

    if args.version:
        print(f"{PROJECT_NAME} {PROJECT_VERSION}")
        return 0
    if args.command == "gui":
        return _launch_gui()
    supported = {
        "comment",
        "ci",
        "deployment",
        "timing",
        "use-cases",
        "call-graph",
        "class-diagram",
        "sequence-diagram",
        "backend-smoke",
    }
    if args.command not in supported:
        parser.print_help()
        return 0

    if args.command == "backend-smoke":
        source_path = Path(args.source) if args.source else None
        source_text = source_path.read_text(encoding="utf-8") if source_path else None
        if args.language == "gdscript":
            from Src.languages import GDScriptTreeSitterBackend

            source = source_text or "class_name Smoke\nextends RefCounted\nfunc run():\n    pass\n"
            module = GDScriptTreeSitterBackend().parse(source, str(source_path or "<backend-smoke>"))
            expected = "LoadConfig" if source_path else "Smoke"
            if not any(entity.name == expected for entity in module.entities):
                return 1
            print(GDScriptTreeSitterBackend.descriptor.backend_id)
            return 0
        if args.language == "csharp":
            from Src.languages import CSharpRoslynBackend

            source = source_text or (
                "public interface I<T> { T Run(T value); }\n"
                "public class Smoke<T> : I<T> {\n"
                "    public T Run(T value) { return Helper(value); }\n"
                "    private T Helper(T value) { return value; }\n"
                "}\n"
            )
            module = CSharpRoslynBackend().parse(source, str(source_path or "<backend-smoke.cs>"))
            expected = "LoadConfig" if source_path else "Smoke"
            target = next((entity for entity in module.entities if entity.kind.value == "class" and entity.name == expected), None)
            if target is None:
                return 1
            if not source_path and ("I" not in target.bases or target.type_parameters != ("T",)):
                return 1
            print(CSharpRoslynBackend.descriptor.backend_id)
            return 0
        if args.language == "java":
            from Src.languages import JavaParserSymbolSolverBackend

            source = source_text or (
                "package smoke;\n"
                "interface I<T> { T run(T value); }\n"
                "class Smoke<T> implements I<T> {\n"
                "    public T run(T value) { return helper(value); }\n"
                "    private T helper(T value) { return value; }\n"
                "}\n"
            )
            module = JavaParserSymbolSolverBackend().parse(source, str(source_path or "<backend-smoke.java>"))
            expected = "LoadConfig" if source_path else "Smoke"
            target = next((entity for entity in module.entities if entity.kind.value == "class" and entity.name == expected), None)
            if target is None:
                return 1
            if not source_path and ("I" not in target.bases or target.type_parameters != ("T",)):
                return 1
            print(JavaParserSymbolSolverBackend.descriptor.backend_id)
            return 0
    if args.command == "deployment":
        deployment_service = DeploymentService()
        result = deployment_service.generate(
            DeploymentAnalysisRequest(Path(args.root), args.mode)
        )
        if args.output_dir:
            paths = deployment_service.save(Path(args.output_dir), result)
            for path in paths:
                print(path)
        else:
            _print_outputs(result)
        return 0

    service = _application_service()
    if args.command == "timing":
        path = Path(args.source)
        language = args.language or service.detect_language(path)
        if language == "unknown":
            parser.error(f"対応していないファイル形式です: {path.suffix or path.name}")
        timing_service = TimingService()
        result = timing_service.generate(TimingAnalysisRequest(path, language))
        if args.output_dir:
            paths = timing_service.save(Path(args.output_dir), result)
            for output_path in paths:
                print(output_path)
        else:
            _print_outputs(result)
        return 0

    if args.command == "use-cases":
        path = Path(args.source)
        language = args.language or service.detect_language(path)
        if language == "unknown":
            parser.error(f"対応していないファイル形式です: {path.suffix or path.name}")
        use_case_service = UseCaseService()
        result = use_case_service.generate(
            UseCaseAnalysisRequest(path, language),
            max_depth=args.max_depth,
        )
        if args.output_dir:
            paths = use_case_service.save(Path(args.output_dir), result)
            for output_path in paths:
                print(output_path)
        else:
            _print_outputs(result)
        return 0

    if args.command == "call-graph":
        path = Path(args.source)
        language = args.language or service.detect_language(path)
        if language == "unknown":
            parser.error(f"対応していないファイル形式です: {path.suffix or path.name}")
        call_graph_service = CallGraphService()
        result = call_graph_service.generate(
            CallGraphAnalysisRequest(path, language),
            fan_in_threshold=args.fan_in_threshold,
            root=args.root,
            max_depth=args.max_depth,
        )
        if args.output_dir:
            paths = call_graph_service.save(Path(args.output_dir), result)
            for output_path in paths:
                print(output_path)
        else:
            _print_outputs(result)
        return 0

    if args.command in {"class-diagram", "sequence-diagram"}:
        path = Path(args.source)
        language = args.language or service.detect_language(path)
        if language == "unknown":
            parser.error(f"対応していないファイル形式です: {path.suffix or path.name}")
        request = SourceAnalysisRequest(path, language)
        if args.command == "class-diagram":
            result = service.generate_class_diagrams(request, renderer=args.renderer)
            category = "class_diagrams"
        else:
            result = service.generate_sequence_diagrams(
                request,
                renderer=args.renderer,
                max_depth=args.max_depth,
                show_duplicate_calls=not args.hide_duplicate_calls,
                show_returns=args.show_returns,
            )
            category = "sequence_diagrams"
        if args.output_dir:
            paths = service.save_diagram_set(Path(args.output_dir), result, category=category)
            for output_path in paths:
                print(output_path)
        else:
            _print_outputs(result)
        return 0

    if args.command == "ci":
        result = service.analyze_ci(CIRequest(Path(args.source)))
        if args.output:
            write_text(Path(args.output), result.content)
        else:
            print(result.content, end="")
        if args.check:
            for finding in result.findings:
                severity = {"error": "エラー", "warning": "警告", "info": "情報"}.get(
                    finding.severity, finding.severity
                )
                print(f"{severity}: {finding.code}: {finding.message}", file=sys.stderr)
            return 1 if any(finding.severity == "error" for finding in result.findings) else 0
        return 0
    if args.output and args.in_place:
        parser.error("--output と --in-place は同時に指定できません")
    path = Path(args.source)
    language = args.language or service.detect_language(path)
    if language == "unknown":
        parser.error(f"対応していないファイル形式です: {path.suffix or path.name}")
    result = service.generate_comments(CommentRequest(path, language))
    if args.in_place:
        write_text(path, result.content)
    elif args.output:
        write_text(Path(args.output), result.content)
    else:
        print(result.content, end="")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
