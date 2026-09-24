"""Tkinter GUI for the Python reference implementation."""

from __future__ import annotations

from pathlib import Path
import tkinter as tk
from tkinter import filedialog, messagebox, scrolledtext, ttk

from Src.generators.class_diagram import ClassDiagramOptions
from Src.process.application import (
    ApplicationService,
    CIRequest,
    CommentRequest,
    ProjectAnalysisRequest,
    SourceAnalysisRequest,
)
from Src.process.call_graph_service import CallGraphAnalysisRequest, CallGraphService
from Src.process.deployment_service import DeploymentAnalysisRequest, DeploymentService
from Src.process.timing_service import TimingAnalysisRequest, TimingService
from Src.process.use_case_service import UseCaseAnalysisRequest, UseCaseService

PROJECT_NAME = "Tomiya Code Atlas"

_OPERATION_COMMENTS = "Generate comments"
_OPERATION_CALL_GRAPH = "Call graph (Mermaid)"
_OPERATION_CLASS_DIAGRAM = "Class diagrams"
_OPERATION_OBJECT_DIAGRAM = "Object diagrams (Mermaid)"
_OPERATION_SEQUENCE_DIAGRAM = "Sequence diagrams"
_OPERATION_COMMUNICATION_DIAGRAM = "Communication diagrams (Mermaid)"
_OPERATION_STATE_DIAGRAM = "State diagrams (Mermaid)"
_OPERATION_PACKAGE_DIAGRAM = "Package diagrams (Mermaid)"
_OPERATION_COMPONENT_DIAGRAM = "Component diagrams (Mermaid)"
_OPERATION_DEPLOYMENT_DIAGRAM = "Deployment diagrams (Mermaid)"
_OPERATION_TIMING_CHART = "Timing charts (Mermaid)"
_OPERATION_USE_CASE_DIAGRAM = "Use case diagrams (Mermaid)"
_OPERATION_RESPONSIBILITY = "Class responsibility tables"
_OPERATION_DESIGN_QUALITY = "Design quality report"
_OPERATION_CI = "GitHub Actions CI graph"
_OPERATIONS = (
    _OPERATION_COMMENTS,
    _OPERATION_CALL_GRAPH,
    _OPERATION_CLASS_DIAGRAM,
    _OPERATION_OBJECT_DIAGRAM,
    _OPERATION_SEQUENCE_DIAGRAM,
    _OPERATION_COMMUNICATION_DIAGRAM,
    _OPERATION_STATE_DIAGRAM,
    _OPERATION_PACKAGE_DIAGRAM,
    _OPERATION_COMPONENT_DIAGRAM,
    _OPERATION_DEPLOYMENT_DIAGRAM,
    _OPERATION_TIMING_CHART,
    _OPERATION_USE_CASE_DIAGRAM,
    _OPERATION_RESPONSIBILITY,
    _OPERATION_DESIGN_QUALITY,
    _OPERATION_CI,
)
_PROJECT_OPERATIONS = {
    _OPERATION_PACKAGE_DIAGRAM,
    _OPERATION_COMPONENT_DIAGRAM,
    _OPERATION_DEPLOYMENT_DIAGRAM,
}


class AtlasTkApp:
    """Thin Tkinter front end over application services."""

    def __init__(self, root: tk.Tk, service: ApplicationService | None = None) -> None:
        self.root = root
        self.service = service or ApplicationService()
        self.call_graph_service = CallGraphService()
        self.deployment_service = DeploymentService()
        self.timing_service = TimingService()
        self.use_case_service = UseCaseService()
        self.base_path: Path | None = None
        self.files: list[Path] = []
        self.current_file: Path | None = None
        self.last_result = ""
        self.last_format = "text"
        self.last_diagram_set: object | None = None
        self.last_diagram_category: str | None = None
        self.last_outputs: tuple[object, ...] = ()

        sequence_settings = self.service.sequence_diagram_settings()
        configured_renderer = self.service.config.renderer.lower()
        if configured_renderer not in {"mermaid", "plantuml"}:
            configured_renderer = "mermaid"
        self.path_var = tk.StringVar(value="No file or folder selected")
        self.language_var = tk.StringVar(value="Language: -")
        self.operation_var = tk.StringVar(value=_OPERATION_COMMENTS)
        self.status_var = tk.StringVar(value="Select a source file or project folder.")
        self.renderer_var = tk.StringVar(value=configured_renderer)
        self.sequence_duplicate_var = tk.BooleanVar(
            value=sequence_settings["show_duplicate_calls"]
        )
        self.sequence_returns_var = tk.BooleanVar(value=sequence_settings["show_returns"])
        self.deployment_mode_var = tk.StringVar(value="full")
        self.class_public_var = tk.BooleanVar(value=True)
        self.class_protected_var = tk.BooleanVar(value=True)
        self.class_internal_var = tk.BooleanVar(value=True)
        self.class_private_var = tk.BooleanVar(value=True)
        self.class_methods_var = tk.BooleanVar(value=True)
        self.class_inheritance_var = tk.BooleanVar(value=True)
        self.class_uses_var = tk.BooleanVar(value=True)

        self._build_window()

    def _build_window(self) -> None:
        self.root.title(PROJECT_NAME)
        self.root.geometry("1180x840")
        self.root.minsize(860, 640)

        outer = ttk.Frame(self.root, padding=10)
        outer.pack(fill=tk.BOTH, expand=True)

        chooser = ttk.Frame(outer)
        chooser.pack(fill=tk.X)
        ttk.Label(chooser, textvariable=self.path_var).pack(side=tk.LEFT, fill=tk.X, expand=True)
        ttk.Button(chooser, text="Open File", command=self.open_file).pack(side=tk.LEFT, padx=(8, 0))
        ttk.Button(chooser, text="Open Folder", command=self.open_folder).pack(side=tk.LEFT, padx=(8, 0))

        controls = ttk.Frame(outer)
        controls.pack(fill=tk.X, pady=(10, 6))
        ttk.Label(controls, textvariable=self.language_var).pack(side=tk.LEFT)
        ttk.Label(controls, text="Operation:").pack(side=tk.LEFT, padx=(18, 6))
        operation = ttk.Combobox(
            controls,
            textvariable=self.operation_var,
            values=_OPERATIONS,
            state="readonly",
            width=36,
        )
        operation.pack(side=tk.LEFT)
        ttk.Label(controls, text="Renderer:").pack(side=tk.LEFT, padx=(18, 6))
        ttk.Combobox(
            controls,
            textvariable=self.renderer_var,
            values=("mermaid", "plantuml"),
            state="readonly",
            width=10,
        ).pack(side=tk.LEFT)
        ttk.Button(controls, text="Run / Re-run", command=self.run_selected).pack(side=tk.LEFT, padx=(8, 0))
        ttk.Button(controls, text="Save Result", command=self.save_result).pack(side=tk.LEFT, padx=(8, 0))

        class_settings = ttk.Labelframe(outer, text="Class diagram filters", padding=6)
        class_settings.pack(fill=tk.X, pady=(0, 6))
        for text, variable in (
            ("Public", self.class_public_var),
            ("Protected", self.class_protected_var),
            ("Internal", self.class_internal_var),
            ("Private", self.class_private_var),
            ("Methods", self.class_methods_var),
            ("Inheritance", self.class_inheritance_var),
            ("Type / constructor use", self.class_uses_var),
        ):
            ttk.Checkbutton(class_settings, text=text, variable=variable).pack(
                side=tk.LEFT, padx=(0, 12)
            )

        sequence_settings = ttk.Labelframe(outer, text="Sequence diagram settings", padding=6)
        sequence_settings.pack(fill=tk.X, pady=(0, 6))
        ttk.Checkbutton(
            sequence_settings,
            text="Show duplicate calls",
            variable=self.sequence_duplicate_var,
        ).pack(side=tk.LEFT)
        ttk.Checkbutton(
            sequence_settings,
            text="Show return messages",
            variable=self.sequence_returns_var,
        ).pack(side=tk.LEFT, padx=(18, 0))
        ttk.Label(
            sequence_settings,
            text="Cycles are always excluded from sequence diagrams.",
        ).pack(side=tk.LEFT, padx=(18, 0))

        deployment_settings = ttk.Labelframe(outer, text="Deployment diagram settings", padding=6)
        deployment_settings.pack(fill=tk.X, pady=(0, 8))
        ttk.Label(deployment_settings, text="Mode:").pack(side=tk.LEFT)
        ttk.Combobox(
            deployment_settings,
            textvariable=self.deployment_mode_var,
            values=("simple", "full"),
            state="readonly",
            width=10,
        ).pack(side=tk.LEFT, padx=(6, 0))
        ttk.Label(
            deployment_settings,
            text="Simple: source dependencies only / Full: + Docker, Compose and Kubernetes",
        ).pack(side=tk.LEFT, padx=(18, 0))

        pane = ttk.Panedwindow(outer, orient=tk.HORIZONTAL)
        pane.pack(fill=tk.BOTH, expand=True)

        files_frame = ttk.Labelframe(pane, text="Project files", padding=6)
        result_frame = ttk.Labelframe(pane, text="Generated results / selected source", padding=6)
        pane.add(files_frame, weight=1)
        pane.add(result_frame, weight=3)

        self.file_list = tk.Listbox(files_frame, exportselection=False)
        file_scroll = ttk.Scrollbar(files_frame, orient=tk.VERTICAL, command=self.file_list.yview)
        self.file_list.configure(yscrollcommand=file_scroll.set)
        self.file_list.pack(side=tk.LEFT, fill=tk.BOTH, expand=True)
        file_scroll.pack(side=tk.RIGHT, fill=tk.Y)
        self.file_list.bind("<<ListboxSelect>>", self._on_file_selected)

        ttk.Label(result_frame, text="Outputs:").pack(fill=tk.X)
        self.result_list = tk.Listbox(result_frame, exportselection=False, height=5)
        self.result_list.pack(fill=tk.X, pady=(2, 6))
        self.result_list.bind("<<ListboxSelect>>", self._on_result_selected)

        self.result_text = scrolledtext.ScrolledText(result_frame, wrap=tk.NONE, undo=False)
        self.result_text.pack(fill=tk.BOTH, expand=True)
        self.result_text.configure(state=tk.DISABLED)

        ttk.Label(outer, textvariable=self.status_var, anchor=tk.W).pack(fill=tk.X, pady=(8, 0))

    def open_file(self) -> None:
        selected = filedialog.askopenfilename(
            title="Select source file",
            filetypes=[
                ("Supported files", "*.py *.gd *.cs *.cpp *.cc *.cxx *.hpp *.java *.go *.yml *.yaml"),
                ("All files", "*.*"),
            ],
        )
        if not selected:
            return
        path = Path(selected)
        if self.service.detect_language(path) == "unknown":
            messagebox.showwarning(PROJECT_NAME, "The selected file type is not supported yet.")
            return
        self._load_paths(path, [path])

    def open_folder(self) -> None:
        selected = filedialog.askdirectory(title="Select project folder")
        if not selected:
            return
        root = Path(selected)
        files = self.service.discover_supported_files(root)
        self._load_paths(root, files)
        if not files:
            self.status_var.set(
                "No supported source files were found. Deployment analysis can still inspect project configuration."
            )

    def _load_paths(self, base_path: Path, files: list[Path]) -> None:
        self.base_path = base_path
        self.files = files
        self.current_file = None
        self.path_var.set(str(base_path))
        self.file_list.delete(0, tk.END)
        self._clear_result_catalog()
        for path in files:
            display = str(path.relative_to(base_path)) if base_path.is_dir() else path.name
            self.file_list.insert(tk.END, display)
        if files:
            self.file_list.selection_set(0)
            self.file_list.activate(0)
            self._select_file(0)
            self.status_var.set(f"{len(files)} supported file(s) available.")

    def _on_file_selected(self, _event: object) -> None:
        selection = self.file_list.curselection()
        if selection:
            self._select_file(selection[0])

    def _select_file(self, index: int) -> None:
        self.current_file = self.files[index]
        language = self.service.detect_language(self.current_file)
        self.language_var.set(f"Language: {language}")
        if language == "yaml" and self.operation_var.get() != _OPERATION_DEPLOYMENT_DIAGRAM:
            self.operation_var.set(_OPERATION_CI)
        elif language != "python" and self.operation_var.get() in {
            _OPERATION_CALL_GRAPH,
            _OPERATION_CLASS_DIAGRAM,
            _OPERATION_OBJECT_DIAGRAM,
            _OPERATION_SEQUENCE_DIAGRAM,
            _OPERATION_COMMUNICATION_DIAGRAM,
            _OPERATION_STATE_DIAGRAM,
            _OPERATION_PACKAGE_DIAGRAM,
            _OPERATION_COMPONENT_DIAGRAM,
            _OPERATION_TIMING_CHART,
            _OPERATION_USE_CASE_DIAGRAM,
            _OPERATION_RESPONSIBILITY,
            _OPERATION_DESIGN_QUALITY,
        }:
            self.operation_var.set(_OPERATION_COMMENTS)

    def run_selected(self) -> None:
        operation = self.operation_var.get()
        project_operation = operation in _PROJECT_OPERATIONS
        if self.current_file is None and not (
            project_operation and self.base_path is not None and self.base_path.is_dir()
        ):
            messagebox.showinfo(PROJECT_NAME, "Select a source file first.")
            return

        path = self.current_file or self.base_path
        if path is None:
            return
        language = self.service.detect_language(path) if path.is_file() else "project"
        self.status_var.set(f"Running {operation} for {path.name}...")
        self.root.update_idletasks()
        self.last_diagram_set = None
        self.last_diagram_category = None
        self.last_outputs = ()

        try:
            if operation == _OPERATION_DEPLOYMENT_DIAGRAM:
                if self.base_path is None or not self.base_path.is_dir():
                    raise ValueError("Deployment diagrams require opening a project folder.")
                outputs = self.deployment_service.generate(
                    DeploymentAnalysisRequest(
                        self.base_path,
                        self.deployment_mode_var.get(),
                    )
                )
                self.last_diagram_set = outputs
                self.last_diagram_category = "deployment_diagrams"
                content = self._output_set_text(outputs)
                result_format = "mermaid-bundle"
            elif operation == _OPERATION_TIMING_CHART:
                outputs = self.timing_service.generate(TimingAnalysisRequest(path, language))
                self.last_diagram_set = outputs
                self.last_diagram_category = "timing_charts"
                content = self._output_set_text(outputs)
                result_format = "mermaid-bundle"
            elif operation == _OPERATION_USE_CASE_DIAGRAM:
                outputs = self.use_case_service.generate(UseCaseAnalysisRequest(path, language))
                self.last_diagram_set = outputs
                self.last_diagram_category = "use_case_diagrams"
                content = self._output_set_text(outputs)
                result_format = "mermaid-bundle"
            elif operation == _OPERATION_COMMENTS:
                if language == "yaml":
                    raise ValueError("Comment generation is not available for YAML files.")
                result = self.service.generate_comments(CommentRequest(path, language))
                content = result.content
                result_format = "source"
            elif operation == _OPERATION_CALL_GRAPH:
                outputs = self.call_graph_service.generate(
                    CallGraphAnalysisRequest(path, language)
                )
                self.last_diagram_set = outputs
                self.last_diagram_category = "call_graphs"
                content = self._output_set_text(outputs)
                result_format = "mermaid-bundle"
            elif operation == _OPERATION_CLASS_DIAGRAM:
                outputs = self.service.generate_class_diagrams(
                    SourceAnalysisRequest(path, language),
                    options=ClassDiagramOptions(
                        include_public=self.class_public_var.get(),
                        include_protected=self.class_protected_var.get(),
                        include_internal=self.class_internal_var.get(),
                        include_private=self.class_private_var.get(),
                        include_methods=self.class_methods_var.get(),
                        include_inheritance=self.class_inheritance_var.get(),
                        include_uses=self.class_uses_var.get(),
                    ),
                    renderer=self.renderer_var.get(),
                )
                self.last_diagram_set = outputs
                self.last_diagram_category = "class_diagrams"
                content = self._output_set_text(outputs)
                result_format = f"{self.renderer_var.get()}-bundle"
            elif operation == _OPERATION_OBJECT_DIAGRAM:
                outputs = self.service.generate_object_diagrams(SourceAnalysisRequest(path, language))
                self.last_diagram_set = outputs
                self.last_diagram_category = "object_diagrams"
                content = self._output_set_text(outputs)
                result_format = "mermaid-bundle"
            elif operation == _OPERATION_SEQUENCE_DIAGRAM:
                outputs = self.service.generate_sequence_diagrams(
                    SourceAnalysisRequest(path, language),
                    show_duplicate_calls=self.sequence_duplicate_var.get(),
                    show_returns=self.sequence_returns_var.get(),
                    renderer=self.renderer_var.get(),
                )
                self.last_diagram_set = outputs
                self.last_diagram_category = "sequence_diagrams"
                content = self._output_set_text(outputs)
                result_format = f"{self.renderer_var.get()}-bundle"
            elif operation == _OPERATION_COMMUNICATION_DIAGRAM:
                outputs = self.service.generate_communication_diagrams(
                    SourceAnalysisRequest(path, language),
                    show_duplicate_calls=self.sequence_duplicate_var.get(),
                )
                self.last_diagram_set = outputs
                self.last_diagram_category = "communication_diagrams"
                content = self._output_set_text(outputs)
                result_format = "mermaid-bundle"
            elif operation == _OPERATION_STATE_DIAGRAM:
                outputs = self.service.generate_state_diagrams(SourceAnalysisRequest(path, language))
                self.last_diagram_set = outputs
                self.last_diagram_category = "state_diagrams"
                content = self._output_set_text(outputs)
                result_format = "mermaid-bundle"
            elif operation == _OPERATION_PACKAGE_DIAGRAM:
                if self.base_path is None or not self.base_path.is_dir():
                    raise ValueError("Package diagrams require opening a project folder.")
                outputs = self.service.generate_package_diagrams(
                    ProjectAnalysisRequest(self.base_path, "python")
                )
                self.last_diagram_set = outputs
                self.last_diagram_category = "package_diagrams"
                content = self._output_set_text(outputs)
                result_format = "mermaid-bundle"
            elif operation == _OPERATION_COMPONENT_DIAGRAM:
                if self.base_path is None or not self.base_path.is_dir():
                    raise ValueError("Component diagrams require opening a project folder.")
                outputs = self.service.generate_component_diagrams(
                    ProjectAnalysisRequest(self.base_path, "python")
                )
                self.last_diagram_set = outputs
                self.last_diagram_category = "component_diagrams"
                content = self._output_set_text(outputs)
                result_format = "mermaid-bundle"
            elif operation == _OPERATION_RESPONSIBILITY:
                outputs = self.service.generate_responsibility_tables(
                    SourceAnalysisRequest(path, language),
                    output_format="markdown",
                )
                self.last_diagram_set = outputs
                self.last_diagram_category = "responsibility_tables"
                content = self._output_set_text(outputs)
                result_format = "markdown-bundle"
            elif operation == _OPERATION_DESIGN_QUALITY:
                result = self.service.evaluate_design_quality(SourceAnalysisRequest(path, language))
                content = result.content
                result_format = result.format
            elif operation == _OPERATION_CI:
                if path.suffix.lower() not in {".yml", ".yaml"}:
                    raise ValueError("CI analysis expects a GitHub Actions YAML file.")
                result = self.service.analyze_ci(CIRequest(path))
                content = result.content
                result_format = "mermaid"
            else:
                raise ValueError(f"Unknown operation: {operation}")
        except Exception as exc:
            self.status_var.set("Analysis failed.")
            messagebox.showerror(PROJECT_NAME, str(exc))
            return

        self.last_result = content
        self.last_format = result_format
        if self.last_diagram_set is not None:
            self._show_output_set(self.last_diagram_set)
            count = len(self.last_outputs)
            self.status_var.set(f"Completed: {operation} ({count} output(s), {result_format}).")
        else:
            self._clear_result_catalog()
            self._show_result(content)
            self.status_var.set(f"Completed: {operation} ({result_format}).")

    @staticmethod
    def _output_set_text(result: object) -> str:
        outputs = getattr(result, "outputs", ())
        if not outputs:
            return "No outputs were generated.\n"
        sections = []
        for output in outputs:
            if output.format == "mermaid":
                header = f"%% {output.name}"
            elif output.format == "plantuml":
                header = f"' {output.name}"
            elif output.format == "markdown":
                header = f"## {output.name}"
            else:
                header = f"# {output.name}"
            sections.append(f"{header}\n{output.content.rstrip()}")
        return "\n\n".join(sections) + "\n"

    def _show_output_set(self, result: object) -> None:
        self.last_outputs = tuple(getattr(result, "outputs", ()))
        self.result_list.delete(0, tk.END)
        for output in self.last_outputs:
            relative_dir = getattr(output, "relative_dir", ())
            prefix = "/".join(relative_dir)
            label = f"{prefix + '/' if prefix else ''}{output.name} [{output.format}]"
            self.result_list.insert(tk.END, label)
        if self.last_outputs:
            self.result_list.selection_set(0)
            self.result_list.activate(0)
            self._show_result(getattr(self.last_outputs[0], "content", ""))
        else:
            self._show_result("No outputs were generated.\n")

    def _on_result_selected(self, _event: object) -> None:
        selection = self.result_list.curselection()
        if not selection or not self.last_outputs:
            return
        index = selection[0]
        output = self.last_outputs[index]
        self._show_result(getattr(output, "content", ""))
        self.status_var.set(
            f"Viewing {getattr(output, 'name', 'output')} ({getattr(output, 'format', 'text')})."
        )

    def _clear_result_catalog(self) -> None:
        self.last_outputs = ()
        if hasattr(self, "result_list"):
            self.result_list.delete(0, tk.END)

    def _show_result(self, content: str) -> None:
        self.result_text.configure(state=tk.NORMAL)
        self.result_text.delete("1.0", tk.END)
        self.result_text.insert("1.0", content)
        self.result_text.configure(state=tk.DISABLED)

    def save_result(self) -> None:
        if not self.last_result:
            messagebox.showinfo(PROJECT_NAME, "Run an analysis before saving a result.")
            return

        if self.last_diagram_set is not None and self.last_diagram_category is not None:
            selected = filedialog.askdirectory(title="Select output folder")
            if not selected:
                return
            paths = self.service.save_diagram_set(
                Path(selected),
                self.last_diagram_set,
                category=self.last_diagram_category,
            )
            self.status_var.set(f"Saved {len(paths)} output file(s) under {selected}")
            return

        extension = {
            "mermaid": ".mmd",
            "plantuml": ".puml",
            "markdown": ".md",
            "csv": ".csv",
            "source": self.current_file.suffix if self.current_file else ".txt",
        }.get(self.last_format, ".txt")
        selected = filedialog.asksaveasfilename(
            title="Save result",
            defaultextension=extension,
            filetypes=[("Result file", f"*{extension}"), ("All files", "*.*")],
        )
        if not selected:
            return
        self.service.save_text(Path(selected), self.last_result)
        self.status_var.set(f"Saved result to {selected}")


def launch_gui(service: ApplicationService | None = None) -> None:
    """Start the Tkinter desktop application."""
    root = tk.Tk()
    AtlasTkApp(root, service)
    root.mainloop()
