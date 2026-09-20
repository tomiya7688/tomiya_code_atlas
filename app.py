"""Kadoka Code Atlas application entry point."""
from __future__ import annotations
import argparse
import sys
from pathlib import Path
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
PROJECT_NAME = "Kadoka Code Atlas"
PROJECT_VERSION = "0.1.0"
CONFIG_FILE_NAME = "kadoka-code-atlas.json"


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


def main(argv: list[str] | None = None) -> int:
    arguments = list(sys.argv[1:] if argv is None else argv)
    if not arguments:
        return _launch_gui()

    parser = argparse.ArgumentParser(description=PROJECT_NAME)
    parser.add_argument("--version", action="store_true", help="Show version and exit.")
    sub = parser.add_subparsers(dest="command")
    sub.add_parser("gui", help="Open the desktop GUI.")
    comment = sub.add_parser("comment", help="Generate deterministic source comments.")
    comment.add_argument("source")
    comment.add_argument("--language", help="Adapter name; inferred from the extension.")
    comment.add_argument("--output")
    comment.add_argument("--in-place", action="store_true")
    ci = sub.add_parser("ci", help="Analyze a GitHub Actions workflow.")
    ci.add_argument("source")
    ci.add_argument("--output")
    ci.add_argument("--check", action="store_true", help="Report quality findings and fail on errors.")
    deployment = sub.add_parser("deployment", help="Generate project deployment diagrams.")
    deployment.add_argument("root")
    deployment.add_argument("--mode", choices=("simple", "full"), default="full")
    deployment.add_argument("--output-dir")
    timing = sub.add_parser("timing", help="Generate logical timing charts from source.")
    timing.add_argument("source")
    timing.add_argument("--language", help="Adapter name; inferred from the extension.")
    timing.add_argument("--output-dir")
    use_cases = sub.add_parser("use-cases", help="Generate GUI-originated use case diagrams.")
    use_cases.add_argument("source")
    use_cases.add_argument("--language", help="Adapter name; inferred from the extension.")
    use_cases.add_argument("--output-dir")
    use_cases.add_argument("--max-depth", type=int, default=5)
    call_graph = sub.add_parser("call-graph", help="Generate partitioned call graphs.")
    call_graph.add_argument("source")
    call_graph.add_argument("--language", help="Adapter name; inferred from the extension.")
    call_graph.add_argument("--output-dir")
    call_graph.add_argument("--fan-in-threshold", type=int, default=3)
    call_graph.add_argument("--root")
    call_graph.add_argument("--max-depth", type=int)
    class_diagram = sub.add_parser("class-diagram", help="Generate class diagrams.")
    class_diagram.add_argument("source")
    class_diagram.add_argument("--language", help="Adapter name; inferred from the extension.")
    class_diagram.add_argument("--renderer", choices=("mermaid", "plantuml"))
    class_diagram.add_argument("--output-dir")
    sequence_diagram = sub.add_parser("sequence-diagram", help="Generate sequence diagrams.")
    sequence_diagram.add_argument("source")
    sequence_diagram.add_argument("--language", help="Adapter name; inferred from the extension.")
    sequence_diagram.add_argument("--renderer", choices=("mermaid", "plantuml"))
    sequence_diagram.add_argument("--output-dir")
    sequence_diagram.add_argument("--max-depth", type=int, default=8)
    sequence_diagram.add_argument("--hide-duplicate-calls", action="store_true")
    sequence_diagram.add_argument("--show-returns", action="store_true")
    backend_smoke = sub.add_parser("backend-smoke", help=argparse.SUPPRESS)
    backend_smoke.add_argument("language", choices=("gdscript", "csharp", "java"))
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
        if args.language == "gdscript":
            from Src.languages import GDScriptTreeSitterBackend

            module = GDScriptTreeSitterBackend().parse(
                "class_name Smoke\nextends RefCounted\nfunc run():\n    pass\n",
                "<backend-smoke>",
            )
            if not any(entity.name == "Smoke" for entity in module.entities):
                return 1
            print(GDScriptTreeSitterBackend.descriptor.backend_id)
            return 0
        if args.language == "csharp":
            from Src.languages import CSharpRoslynBackend

            module = CSharpRoslynBackend().parse(
                "public interface I<T> { T Run(T value); }\n"
                "public class Smoke<T> : I<T> {\n"
                "    public T Run(T value) { return Helper(value); }\n"
                "    private T Helper(T value) { return value; }\n"
                "}\n",
                "<backend-smoke.cs>",
            )
            smoke = next(
                (
                    entity
                    for entity in module.entities
                    if entity.kind.value == "class" and entity.name == "Smoke"
                ),
                None,
            )
            if smoke is None or "I" not in smoke.bases or smoke.type_parameters != ("T",):
                return 1
            print(CSharpRoslynBackend.descriptor.backend_id)
            return 0
        if args.language == "java":
            from Src.languages import JavaParserSymbolSolverBackend

            module = JavaParserSymbolSolverBackend().parse(
                "package smoke;\n"
                "interface I<T> { T run(T value); }\n"
                "class Smoke<T> implements I<T> {\n"
                "    public T run(T value) { return helper(value); }\n"
                "    private T helper(T value) { return value; }\n"
                "}\n",
                "<backend-smoke.java>",
            )
            smoke = next(
                (
                    entity
                    for entity in module.entities
                    if entity.kind.value == "class" and entity.name == "Smoke"
                ),
                None,
            )
            if smoke is None or "I" not in smoke.bases or smoke.type_parameters != ("T",):
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
            parser.error(f"unsupported source file type: {path.suffix or path.name}")
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
            parser.error(f"unsupported source file type: {path.suffix or path.name}")
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
            parser.error(f"unsupported source file type: {path.suffix or path.name}")
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
            parser.error(f"unsupported source file type: {path.suffix or path.name}")
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
                print(f"{finding.severity}: {finding.code}: {finding.message}", file=sys.stderr)
            return 1 if any(finding.severity == "error" for finding in result.findings) else 0
        return 0
    if args.output and args.in_place:
        parser.error("--output and --in-place cannot be combined")
    path = Path(args.source)
    language = args.language or service.detect_language(path)
    if language == "unknown":
        parser.error(f"unsupported source file type: {path.suffix or path.name}")
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