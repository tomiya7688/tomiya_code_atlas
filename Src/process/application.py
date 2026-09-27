"""UI-independent application services."""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

from Src.analyzers.ci import parse_github_actions
from Src.analyzers.component_dependencies import (
    ComponentDependencyUnit,
    build_component_dependency_graph,
)
from Src.analyzers.package_dependencies import (
    ModuleDependencyUnit,
    build_package_dependency_graph,
)
from Src.data.files import read_text, write_text
from Src.data.generated_outputs import GeneratedOutputLike, save_generated_outputs
from Src.data.project_files import detect_language, discover_supported_files
from Src.evaluators import evaluate_ci
from Src.evaluators.design_quality import (
    DesignQualityThresholds,
    evaluate_design_quality as evaluate_design_quality_model,
)
from Src.generators import CommentGenerator, build_call_graph, rows
from Src.generators.class_diagram import ClassDiagramOptions, build_class_diagram_bundle
from Src.generators.communication_diagram import build_communication_diagram_bundle
from Src.generators.component_diagram import build_component_diagram_bundle
from Src.generators.object_diagram import build_object_diagram_bundle
from Src.generators.package_diagram import build_package_diagram_bundle
from Src.generators.responsibility import build_responsibility_table_bundle
from Src.generators.sequence_diagram import SequenceDiagramOptions, build_sequence_diagram_bundle
from Src.generators.state_diagram import build_state_diagram_bundle
from Src.languages.python_backend import PythonStdlibBackend
from Src.models.config import AtlasConfig
from Src.renderers import (
    render_call_graph,
    render_ci_workflow,
    render_responsibility_csv,
    render_responsibility_markdown,
)
from Src.renderers.design_quality_markdown import render_design_quality_markdown
from Src.renderers.mermaid_class_diagram import render_class_diagram
from Src.renderers.mermaid_communication_diagram import render_communication_diagram
from Src.renderers.mermaid_component_diagram import render_component_diagram
from Src.renderers.mermaid_object_diagram import render_object_diagram
from Src.renderers.mermaid_package_diagram import render_package_diagram
from Src.renderers.mermaid_sequence_diagram import render_sequence_diagram
from Src.renderers.mermaid_state_diagram import render_state_diagram
from Src.renderers.plantuml_class_diagram import render_class_diagram as render_class_diagram_plantuml
from Src.renderers.plantuml_sequence_diagram import render_sequence_diagram as render_sequence_diagram_plantuml


@dataclass(frozen=True, slots=True)
class CommentRequest:
    source: Path
    language: str


@dataclass(frozen=True, slots=True)
class CIRequest:
    source: Path


@dataclass(frozen=True, slots=True)
class SourceAnalysisRequest:
    source: Path
    language: str


@dataclass(frozen=True, slots=True)
class ProjectAnalysisRequest:
    root: Path
    language: str = "python"


@dataclass(frozen=True, slots=True)
class CommentResult:
    content: str


@dataclass(frozen=True, slots=True)
class CIResult:
    content: str
    findings: tuple[object, ...]


@dataclass(frozen=True, slots=True)
class TextResult:
    content: str
    format: str = "text"


@dataclass(frozen=True, slots=True)
class GeneratedOutput:
    name: str
    content: str
    format: str
    relative_dir: tuple[str, ...] = ()


@dataclass(frozen=True, slots=True)
class DiagramSetResult:
    outputs: tuple[GeneratedOutput, ...]
    statistics: dict[str, int | float | tuple[int, ...]]


class ApplicationService:
    """Orchestrate analysis and generation without presentation decisions."""

    def __init__(self, config: AtlasConfig | None = None) -> None:
        self.config = config or AtlasConfig()

    def generate_comments(self, request: CommentRequest) -> CommentResult:
        return CommentResult(CommentGenerator().generate(read_text(request.source), request.language))

    def analyze_ci(self, request: CIRequest) -> CIResult:
        workflow = parse_github_actions(read_text(request.source))
        return CIResult(render_ci_workflow(workflow), evaluate_ci(workflow))

    def evaluate_design_quality(
        self,
        request: SourceAnalysisRequest,
        *,
        high_fan_in: int = 3,
        high_fan_out: int = 5,
        large_series: int = 8,
    ) -> TextResult:
        module = self._python_module(request)
        report = evaluate_design_quality_model(
            module,
            thresholds=DesignQualityThresholds(
                high_fan_in=high_fan_in,
                high_fan_out=high_fan_out,
                large_series=large_series,
            ),
        )
        return TextResult(render_design_quality_markdown(report), format="markdown")

    def generate_call_graph(
        self,
        request: SourceAnalysisRequest,
        *,
        root: str | None = None,
        max_depth: int | None = None,
    ) -> TextResult:
        module = self._python_module(request)
        graph = build_call_graph(module, root=root, max_depth=max_depth)
        return TextResult(render_call_graph(graph), format="mermaid")

    def generate_responsibility_table(
        self,
        request: SourceAnalysisRequest,
        *,
        output_format: str = "markdown",
    ) -> TextResult:
        module = self._python_module(request)
        table_rows = rows(module)
        normalized = output_format.lower()
        if normalized == "csv":
            return TextResult(render_responsibility_csv(table_rows), format="csv")
        if normalized != "markdown":
            raise ValueError(f"Unsupported responsibility output format: {output_format}")
        return TextResult(render_responsibility_markdown(table_rows), format="markdown")

    def generate_responsibility_tables(
        self,
        request: SourceAnalysisRequest,
        *,
        fan_in_threshold: int = 3,
        output_format: str = "markdown",
    ) -> DiagramSetResult:
        module = self._python_module(request)
        normalized = output_format.lower()
        if normalized not in {"markdown", "csv"}:
            raise ValueError(f"Unsupported responsibility output format: {output_format}")
        bundle = build_responsibility_table_bundle(
            module,
            fan_in_threshold=fan_in_threshold,
        )
        render = (
            render_responsibility_csv
            if normalized == "csv"
            else render_responsibility_markdown
        )
        outputs = tuple(
            GeneratedOutput(
                table.name,
                render(table.rows),
                normalized,
                placement.relative_dir,
            )
            for table, placement in zip(bundle.tables, bundle.placements, strict=True)
        )
        return DiagramSetResult(outputs, bundle.statistics)

    def generate_class_diagrams(
        self,
        request: SourceAnalysisRequest,
        *,
        fan_in_threshold: int = 3,
        options: ClassDiagramOptions | None = None,
        renderer: str | None = None,
    ) -> DiagramSetResult:
        module = self._python_module(request)
        bundle = build_class_diagram_bundle(
            module,
            fan_in_threshold=fan_in_threshold,
            options=options,
        )
        normalized = self._diagram_renderer(renderer)
        render = render_class_diagram_plantuml if normalized == "plantuml" else render_class_diagram
        outputs = tuple(
            GeneratedOutput(
                diagram.name,
                render(diagram),
                normalized,
                placement.relative_dir,
            )
            for diagram, placement in zip(bundle.diagrams, bundle.placements, strict=True)
        )
        return DiagramSetResult(outputs, bundle.statistics)

    def generate_object_diagrams(
        self,
        request: SourceAnalysisRequest,
        *,
        fan_in_threshold: int = 3,
    ) -> DiagramSetResult:
        module = self._python_module(request)
        bundle = build_object_diagram_bundle(module, fan_in_threshold=fan_in_threshold)
        outputs = tuple(
            GeneratedOutput(
                diagram.name,
                render_object_diagram(diagram),
                "mermaid",
                placement.relative_dir,
            )
            for diagram, placement in zip(bundle.diagrams, bundle.placements, strict=True)
        )
        return DiagramSetResult(outputs, bundle.statistics)

    def generate_state_diagrams(
        self,
        request: SourceAnalysisRequest,
    ) -> DiagramSetResult:
        module = self._python_module(request)
        bundle = build_state_diagram_bundle(module)
        outputs = tuple(
            GeneratedOutput(diagram.name, render_state_diagram(diagram), "mermaid")
            for diagram in bundle.diagrams
        )
        return DiagramSetResult(outputs, bundle.statistics)

    def generate_package_diagrams(
        self,
        request: ProjectAnalysisRequest,
        *,
        fan_in_threshold: int = 3,
    ) -> DiagramSetResult:
        project_units = self._python_project_units(request)
        graph = build_package_dependency_graph([unit for _, unit in project_units])
        bundle = build_package_diagram_bundle(
            graph,
            fan_in_threshold=fan_in_threshold,
        )
        outputs = tuple(
            GeneratedOutput(
                diagram.name,
                render_package_diagram(diagram),
                "mermaid",
                placement.relative_dir,
            )
            for diagram, placement in zip(bundle.diagrams, bundle.placements, strict=True)
        )
        return DiagramSetResult(outputs, bundle.statistics)

    def generate_component_diagrams(
        self,
        request: ProjectAnalysisRequest,
        *,
        fan_in_threshold: int = 3,
    ) -> DiagramSetResult:
        project_units = self._python_project_units(request)
        units = [
            ComponentDependencyUnit(
                unit,
                self._python_component_name(request.root, path),
            )
            for path, unit in project_units
        ]
        graph = build_component_dependency_graph(units)
        bundle = build_component_diagram_bundle(
            graph,
            fan_in_threshold=fan_in_threshold,
        )
        outputs = tuple(
            GeneratedOutput(
                diagram.name,
                render_component_diagram(diagram),
                "mermaid",
                placement.relative_dir,
            )
            for diagram, placement in zip(bundle.diagrams, bundle.placements, strict=True)
        )
        return DiagramSetResult(outputs, bundle.statistics)

    def sequence_diagram_options(self) -> SequenceDiagramOptions:
        return SequenceDiagramOptions.from_mapping(
            self.config.generator_options.get("sequence_diagram")
        )

    def sequence_diagram_settings(self) -> dict[str, bool]:
        options = self.sequence_diagram_options()
        return {
            "show_duplicate_calls": options.show_duplicate_calls,
            "show_returns": options.show_returns,
        }

    def generate_sequence_diagrams(
        self,
        request: SourceAnalysisRequest,
        *,
        fan_in_threshold: int = 3,
        max_depth: int = 8,
        show_duplicate_calls: bool | None = None,
        show_returns: bool | None = None,
        renderer: str | None = None,
    ) -> DiagramSetResult:
        module = self._python_module(request)
        defaults = self.sequence_diagram_options()
        options = SequenceDiagramOptions(
            show_duplicate_calls=(
                defaults.show_duplicate_calls
                if show_duplicate_calls is None
                else show_duplicate_calls
            ),
            show_returns=defaults.show_returns if show_returns is None else show_returns,
        )
        bundle = build_sequence_diagram_bundle(
            module,
            fan_in_threshold=fan_in_threshold,
            max_depth=max_depth,
            options=options,
        )
        normalized = self._diagram_renderer(renderer)
        render = render_sequence_diagram_plantuml if normalized == "plantuml" else render_sequence_diagram
        outputs = tuple(
            GeneratedOutput(
                diagram.name,
                render(diagram),
                normalized,
                placement.relative_dir,
            )
            for diagram, placement in zip(bundle.diagrams, bundle.placements, strict=True)
        )
        return DiagramSetResult(outputs, bundle.statistics)

    def generate_communication_diagrams(
        self,
        request: SourceAnalysisRequest,
        *,
        fan_in_threshold: int = 3,
        max_depth: int = 8,
        show_duplicate_calls: bool | None = None,
    ) -> DiagramSetResult:
        module = self._python_module(request)
        defaults = self.sequence_diagram_options()
        bundle = build_communication_diagram_bundle(
            module,
            fan_in_threshold=fan_in_threshold,
            max_depth=max_depth,
            show_duplicate_calls=(
                defaults.show_duplicate_calls
                if show_duplicate_calls is None
                else show_duplicate_calls
            ),
        )
        outputs = tuple(
            GeneratedOutput(
                diagram.name,
                render_communication_diagram(diagram),
                "mermaid",
                placement.relative_dir,
            )
            for diagram, placement in zip(bundle.diagrams, bundle.placements, strict=True)
        )
        return DiagramSetResult(outputs, bundle.statistics)

    def detect_language(self, path: Path) -> str:
        return detect_language(path)

    def discover_supported_files(self, root: Path) -> list[Path]:
        return discover_supported_files(root)

    def save_text(self, path: Path, content: str) -> None:
        write_text(path, content)

    def save_diagram_set(
        self,
        output_root: Path,
        result: DiagramSetResult,
        *,
        category: str,
    ) -> tuple[Path, ...]:
        return save_generated_outputs(output_root, category, result.outputs)

    def save_output_collection(
        self,
        output_root: Path,
        outputs: tuple[GeneratedOutputLike, ...],
        *,
        category: str,
    ) -> tuple[Path, ...]:
        """Persist generated outputs through the shared Data boundary."""
        return save_generated_outputs(output_root, category, outputs)

    def _diagram_renderer(self, renderer: str | None) -> str:
        normalized = (renderer or self.config.renderer).lower()
        if normalized not in {"mermaid", "plantuml"}:
            raise ValueError(f"Unsupported diagram renderer: {normalized}")
        return normalized

    @staticmethod
    def _python_module_name(root: Path, path: Path) -> str:
        relative = path.relative_to(root)
        parts = list(relative.parts)
        stem = Path(parts[-1]).stem
        if stem == "__init__":
            parts = parts[:-1]
        else:
            parts[-1] = stem
        return ".".join(parts) or root.name

    @staticmethod
    def _python_component_name(root: Path, path: Path) -> str:
        relative = path.relative_to(root)
        if len(relative.parts) == 1:
            return root.name if path.name == "__init__.py" else path.stem
        return ".".join(relative.parts[:-1])

    @classmethod
    def _python_project_units(
        cls,
        request: ProjectAnalysisRequest,
    ) -> list[tuple[Path, ModuleDependencyUnit]]:
        language = request.language.lower().lstrip(".")
        if language not in {"python", "py"}:
            raise ValueError("Project dependency analysis is currently available for Python projects only.")
        if not request.root.is_dir():
            raise ValueError("Project dependency analysis expects a project folder.")

        backend = PythonStdlibBackend()
        units: list[tuple[Path, ModuleDependencyUnit]] = []
        for path in discover_supported_files(request.root):
            if path.suffix.lower() != ".py":
                continue
            module = backend.parse(read_text(path), str(path))
            units.append(
                (
                    path,
                    ModuleDependencyUnit(
                        cls._python_module_name(request.root, path),
                        module.imports,
                        path.name == "__init__.py",
                    ),
                )
            )
        return units

    @staticmethod
    def _python_module(request: SourceAnalysisRequest):
        language = request.language.lower().lstrip(".")
        if language not in {"python", "py"}:
            raise ValueError("This analysis is currently available for Python source files only.")
        return PythonStdlibBackend().parse(read_text(request.source), str(request.source))
