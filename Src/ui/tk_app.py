"""Tkinter GUI for the Python reference implementation."""

from __future__ import annotations

from dataclasses import dataclass
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
from Src.process.operation_requirements import (
    ALL_OPERATIONS,
    OPERATION_CALL_GRAPH as _OPERATION_CALL_GRAPH,
    OPERATION_CI as _OPERATION_CI,
    OPERATION_CLASS_DIAGRAM as _OPERATION_CLASS_DIAGRAM,
    OPERATION_COMMENTS as _OPERATION_COMMENTS,
    OPERATION_COMMUNICATION_DIAGRAM as _OPERATION_COMMUNICATION_DIAGRAM,
    OPERATION_COMPONENT_DIAGRAM as _OPERATION_COMPONENT_DIAGRAM,
    OPERATION_DEPLOYMENT_DIAGRAM as _OPERATION_DEPLOYMENT_DIAGRAM,
    OPERATION_DESIGN_QUALITY as _OPERATION_DESIGN_QUALITY,
    OPERATION_OBJECT_DIAGRAM as _OPERATION_OBJECT_DIAGRAM,
    OPERATION_PACKAGE_DIAGRAM as _OPERATION_PACKAGE_DIAGRAM,
    OPERATION_RESPONSIBILITY as _OPERATION_RESPONSIBILITY,
    OPERATION_SEQUENCE_DIAGRAM as _OPERATION_SEQUENCE_DIAGRAM,
    OPERATION_STATE_DIAGRAM as _OPERATION_STATE_DIAGRAM,
    OPERATION_TIMING_CHART as _OPERATION_TIMING_CHART,
    OPERATION_USE_CASE_DIAGRAM as _OPERATION_USE_CASE_DIAGRAM,
    compatible_operations,
    default_operation_for,
)

PROJECT_NAME = "Tomiya Code Atlas"

_OPERATION_LABELS = {
    _OPERATION_COMMENTS: "コメントを生成",
    _OPERATION_CALL_GRAPH: "コールグラフ（Mermaid）",
    _OPERATION_CLASS_DIAGRAM: "クラス図",
    _OPERATION_OBJECT_DIAGRAM: "オブジェクト図（Mermaid）",
    _OPERATION_SEQUENCE_DIAGRAM: "シーケンス図",
    _OPERATION_COMMUNICATION_DIAGRAM: "コミュニケーション図（Mermaid）",
    _OPERATION_STATE_DIAGRAM: "状態遷移図（Mermaid）",
    _OPERATION_PACKAGE_DIAGRAM: "パッケージ図（Mermaid）",
    _OPERATION_COMPONENT_DIAGRAM: "コンポーネント図（Mermaid）",
    _OPERATION_DEPLOYMENT_DIAGRAM: "デプロイメント図（Mermaid）",
    _OPERATION_TIMING_CHART: "タイミングチャート（Mermaid）",
    _OPERATION_USE_CASE_DIAGRAM: "ユースケース図（Mermaid）",
    _OPERATION_RESPONSIBILITY: "クラス責務表",
    _OPERATION_DESIGN_QUALITY: "設計品質レポート",
    _OPERATION_CI: "GitHub Actions CI図",
}


@dataclass(frozen=True, slots=True)
class GuiOutput:
    name: str
    content: str
    format: str
    relative_dir: tuple[str, ...] = ()
    extension: str | None = None


@dataclass(frozen=True, slots=True)
class GuiOutputSet:
    outputs: tuple[GuiOutput, ...]


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
        self.path_var = tk.StringVar(value="未選択")
        self.output_path_var = tk.StringVar(value=self.service.config.output_dir)
        self.language_var = tk.StringVar(value="言語：未選択")
        self.operation_var = tk.StringVar(value=_OPERATION_COMMENTS)
        self.status_var = tk.StringVar(value="入力ファイルまたはプロジェクトを選択してください。")
        self.renderer_var = tk.StringVar(value=configured_renderer)
        self.sequence_duplicate_var = tk.BooleanVar(
            value=sequence_settings["show_duplicate_calls"]
        )
        self.sequence_returns_var = tk.BooleanVar(value=sequence_settings["show_returns"])
        self.deployment_mode_var = tk.StringVar(value="詳細")
        self.class_public_var = tk.BooleanVar(value=True)
        self.class_protected_var = tk.BooleanVar(value=True)
        self.class_internal_var = tk.BooleanVar(value=True)
        self.class_private_var = tk.BooleanVar(value=True)
        self.class_methods_var = tk.BooleanVar(value=True)
        self.class_inheritance_var = tk.BooleanVar(value=True)
        self.class_uses_var = tk.BooleanVar(value=True)
        self.available_operation_ids = list(ALL_OPERATIONS)
        self.last_run_succeeded = False
        self.last_error = ""

        self._build_window()

    def _build_window(self) -> None:
        self.root.title(PROJECT_NAME)
        self.root.geometry("1180x940")
        self.root.minsize(860, 640)

        outer = ttk.Frame(self.root, padding=10)
        outer.pack(fill=tk.BOTH, expand=True)

        chooser = ttk.Labelframe(outer, text="入力ファイル／プロジェクト", padding=6)
        chooser.pack(fill=tk.X)
        ttk.Label(chooser, textvariable=self.path_var).pack(side=tk.LEFT, fill=tk.X, expand=True)
        ttk.Button(chooser, text="ファイルを選択（Alt+O）", command=self.open_file).pack(
            side=tk.LEFT, padx=(8, 0)
        )
        ttk.Button(chooser, text="フォルダーを選択（Alt+P）", command=self.open_folder).pack(
            side=tk.LEFT, padx=(8, 0)
        )

        output_row = ttk.Labelframe(outer, text="出力先", padding=6)
        output_row.pack(fill=tk.X, pady=(6, 0))
        ttk.Entry(output_row, textvariable=self.output_path_var).pack(
            side=tk.LEFT, fill=tk.X, expand=True
        )
        ttk.Button(output_row, text="参照…", command=self.choose_output_folder).pack(
            side=tk.LEFT, padx=(8, 0)
        )
        ttk.Label(
            outer,
            text="解析結果は指定フォルダー内の analysis_results に保存されます。",
        ).pack(anchor=tk.W, pady=(2, 0))

        operations_frame = ttk.Labelframe(outer, text="実行する解析", padding=6)
        operations_frame.pack(fill=tk.X, pady=(6, 0))
        ttk.Label(
            operations_frame,
            text="複数選択できます（Ctrl または Shift を押しながら選択）。対応する解析だけ表示します。",
        ).pack(anchor=tk.W)
        operation_list_frame = ttk.Frame(operations_frame)
        operation_list_frame.pack(fill=tk.X, pady=(4, 0))
        self.operation_list = tk.Listbox(
            operation_list_frame,
            selectmode=tk.EXTENDED,
            exportselection=False,
            height=5,
        )
        operation_scroll = ttk.Scrollbar(
            operation_list_frame, orient=tk.VERTICAL, command=self.operation_list.yview
        )
        self.operation_list.configure(yscrollcommand=operation_scroll.set)
        self.operation_list.pack(side=tk.LEFT, fill=tk.X, expand=True)
        operation_scroll.pack(side=tk.RIGHT, fill=tk.Y)
        self._populate_operations(ALL_OPERATIONS, (_OPERATION_COMMENTS,))
        self.operation_list.bind("<<ListboxSelect>>", self._on_operation_selected)

        controls = ttk.Frame(outer)
        controls.pack(fill=tk.X, pady=(10, 6))
        ttk.Label(controls, textvariable=self.language_var).pack(side=tk.LEFT)
        ttk.Label(controls, text="図の形式：").pack(side=tk.LEFT, padx=(18, 6))
        ttk.Combobox(
            controls,
            textvariable=self.renderer_var,
            values=("Mermaid", "PlantUML"),
            state="readonly",
            width=10,
        ).pack(side=tk.LEFT)
        ttk.Button(controls, text="解析を実行（Alt+R）", command=self.run_selected).pack(
            side=tk.LEFT, padx=(8, 0)
        )
        ttk.Button(controls, text="出力先へ保存（Alt+S）", command=self.save_result).pack(
            side=tk.LEFT, padx=(8, 0)
        )

        class_settings = ttk.Labelframe(outer, text="クラス図の表示項目", padding=6)
        class_settings.pack(fill=tk.X, pady=(0, 6))
        for text, variable in (
            ("公開", self.class_public_var),
            ("保護", self.class_protected_var),
            ("内部", self.class_internal_var),
            ("非公開", self.class_private_var),
            ("メソッド", self.class_methods_var),
            ("継承", self.class_inheritance_var),
            ("型・生成関係", self.class_uses_var),
        ):
            ttk.Checkbutton(class_settings, text=text, variable=variable).pack(
                side=tk.LEFT, padx=(0, 12)
            )

        sequence_settings = ttk.Labelframe(outer, text="シーケンス図の設定", padding=6)
        sequence_settings.pack(fill=tk.X, pady=(0, 6))
        ttk.Checkbutton(
            sequence_settings,
            text="重複する呼び出しを表示",
            variable=self.sequence_duplicate_var,
        ).pack(side=tk.LEFT)
        ttk.Checkbutton(
            sequence_settings,
            text="戻り値を表示",
            variable=self.sequence_returns_var,
        ).pack(side=tk.LEFT, padx=(18, 0))
        ttk.Label(
            sequence_settings,
            text="循環する呼び出しは表示しません。",
        ).pack(side=tk.LEFT, padx=(18, 0))

        deployment_settings = ttk.Labelframe(outer, text="デプロイメント図の設定", padding=6)
        deployment_settings.pack(fill=tk.X, pady=(0, 8))
        ttk.Label(deployment_settings, text="解析範囲：").pack(side=tk.LEFT)
        ttk.Combobox(
            deployment_settings,
            textvariable=self.deployment_mode_var,
            values=("簡易", "詳細"),
            state="readonly",
            width=10,
        ).pack(side=tk.LEFT, padx=(6, 0))
        ttk.Label(
            deployment_settings,
            text="簡易：ソース依存のみ／詳細：Docker・Compose・Kubernetesも解析",
        ).pack(side=tk.LEFT, padx=(18, 0))

        pane = ttk.Panedwindow(outer, orient=tk.HORIZONTAL)
        pane.pack(fill=tk.BOTH, expand=True)

        files_frame = ttk.Labelframe(pane, text="解析対象ファイル（クリックして切替）", padding=6)
        result_frame = ttk.Labelframe(pane, text="生成結果／選択中の内容", padding=6)
        pane.add(files_frame, weight=1)
        pane.add(result_frame, weight=3)

        self.file_list = tk.Listbox(files_frame, exportselection=False)
        file_scroll = ttk.Scrollbar(files_frame, orient=tk.VERTICAL, command=self.file_list.yview)
        self.file_list.configure(yscrollcommand=file_scroll.set)
        self.file_list.pack(side=tk.LEFT, fill=tk.BOTH, expand=True)
        file_scroll.pack(side=tk.RIGHT, fill=tk.Y)
        self.file_list.bind("<<ListboxSelect>>", self._on_file_selected)

        ttk.Label(result_frame, text="生成ファイル（選択すると内容を表示）：").pack(fill=tk.X)
        self.result_list = tk.Listbox(result_frame, exportselection=False, height=5)
        self.result_list.pack(fill=tk.X, pady=(2, 6))
        self.result_list.bind("<<ListboxSelect>>", self._on_result_selected)

        self.result_text = scrolledtext.ScrolledText(result_frame, wrap=tk.NONE, undo=False)
        self.result_text.pack(fill=tk.BOTH, expand=True)
        self.result_text.configure(state=tk.DISABLED)

        ttk.Label(outer, textvariable=self.status_var, anchor=tk.W).pack(fill=tk.X, pady=(8, 0))

        self.root.bind_all("<Alt-o>", lambda _event: self.open_file() or "break")
        self.root.bind_all("<Alt-p>", lambda _event: self.open_folder() or "break")
        self.root.bind_all("<Alt-r>", lambda _event: self.run_selected() or "break")
        self.root.bind_all("<Alt-s>", lambda _event: self.save_result() or "break")

    def _populate_operations(
        self,
        operations: tuple[str, ...] | list[str],
        selected: tuple[str, ...] | list[str] = (),
    ) -> None:
        self.available_operation_ids = list(operations)
        self.operation_list.delete(0, tk.END)
        for operation_id in self.available_operation_ids:
            self.operation_list.insert(tk.END, _OPERATION_LABELS[operation_id])
        selected_set = set(selected)
        for index, operation_id in enumerate(self.available_operation_ids):
            if operation_id in selected_set:
                self.operation_list.selection_set(index)
        current = next((op for op in self.available_operation_ids if op in selected_set), None)
        if current is not None:
            self.operation_var.set(current)

    def _on_operation_selected(self, _event: object) -> None:
        selected = self._selected_operations()
        if selected:
            self.operation_var.set(selected[0])

    def _selected_operations(self) -> tuple[str, ...]:
        return tuple(
            self.available_operation_ids[index]
            for index in self.operation_list.curselection()
            if index < len(self.available_operation_ids)
        )

    def choose_output_folder(self) -> None:
        current = Path(self.output_path_var.get()).expanduser()
        selected = filedialog.askdirectory(
            title="解析結果の保存先を選択",
            initialdir=str(current if current.is_dir() else Path.cwd()),
        )
        if selected:
            self.output_path_var.set(selected)

    def open_file(self) -> None:
        selected = filedialog.askopenfilename(
            title="解析するファイルを選択",
            filetypes=[
                ("対応ファイル", "*.py *.gd *.cs *.cpp *.cc *.cxx *.hpp *.java *.go *.yml *.yaml"),
                ("すべてのファイル", "*.*"),
            ],
        )
        if not selected:
            return
        path = Path(selected)
        if self.service.detect_language(path) == "unknown":
            messagebox.showwarning(PROJECT_NAME, "このファイル形式には対応していません。")
            return
        self._load_paths(path, [path])

    def open_folder(self) -> None:
        selected = filedialog.askdirectory(title="解析するプロジェクトフォルダーを選択")
        if not selected:
            return
        root = Path(selected)
        files = self.service.discover_supported_files(root)
        self._load_paths(root, files)
        if not files:
            self.status_var.set(
                "対応するソースファイルが見つかりません。デプロイメント図は設定ファイルを解析できます。"
            )

    def _load_paths(self, base_path: Path, files: list[Path]) -> None:
        self.base_path = base_path
        self.files = files
        self.current_file = None
        self.path_var.set(str(base_path))
        output_root = base_path if base_path.is_dir() else base_path.parent
        if not self.output_path_var.get().strip():
            self.output_path_var.set(str(output_root / "tomiya-code-atlas-output"))
        self.file_list.delete(0, tk.END)
        self._clear_result_catalog()
        for path in files:
            display = str(path.relative_to(base_path)) if base_path.is_dir() else path.name
            self.file_list.insert(tk.END, display)
        if files:
            self.file_list.selection_set(0)
            self.file_list.activate(0)
            self._select_file(0)
            self.status_var.set(f"解析対象ファイル：{len(files)}件")
        elif base_path.is_dir():
            allowed = compatible_operations(language=None, has_project_folder=True)
            self._populate_operations(
                allowed,
                (_OPERATION_DEPLOYMENT_DIAGRAM,) if _OPERATION_DEPLOYMENT_DIAGRAM in allowed else (),
            )

    def _on_file_selected(self, _event: object) -> None:
        selection = self.file_list.curselection()
        if selection:
            self._select_file(selection[0])

    def _select_file(self, index: int) -> None:
        self.current_file = self.files[index]
        language = self.service.detect_language(self.current_file)
        self.language_var.set(f"解析言語：{language}")
        has_project_folder = self.base_path is not None and self.base_path.is_dir()
        allowed = compatible_operations(
            language=language,
            has_project_folder=has_project_folder,
        )
        selected = tuple(op for op in self._selected_operations() if op in allowed)
        if not selected:
            preferred = default_operation_for(language)
            selected = (preferred,) if preferred in allowed else allowed[:1]
        self._populate_operations(allowed, selected)

    def run_selected(self) -> None:
        operations = self._selected_operations()
        if not operations:
            messagebox.showinfo(PROJECT_NAME, "実行する解析を1つ以上選択してください。")
            return
        if self.current_file is None and not (
            self.base_path is not None
            and self.base_path.is_dir()
            and _OPERATION_DEPLOYMENT_DIAGRAM in operations
        ):
            messagebox.showinfo(PROJECT_NAME, "先に入力ファイルまたはプロジェクトフォルダーを選択してください。")
            return

        combined: list[GuiOutput] = []
        completed: list[str] = []
        failed: list[tuple[str, str]] = []
        self.last_result = ""
        self.last_outputs = ()
        for operation in operations:
            self.operation_var.set(operation)
            self.last_run_succeeded = False
            self.last_error = ""
            self._run_operation(show_errors=False)
            if not self.last_run_succeeded:
                failed.append((operation, self.last_error))
                continue
            completed.append(operation)
            if self.last_outputs:
                for output in self.last_outputs:
                    combined.append(
                        GuiOutput(
                            name=getattr(output, "name", "output"),
                            content=getattr(output, "content", ""),
                            format=getattr(output, "format", "text"),
                            relative_dir=(operation,)
                            + tuple(getattr(output, "relative_dir", ())),
                            extension=(
                                self.current_file.suffix
                                if getattr(output, "format", "") == "source" and self.current_file
                                else None
                            ),
                        )
                    )
            else:
                source_name = self.current_file.stem if self.current_file else "project"
                combined.append(
                    GuiOutput(
                        name=f"{source_name}-{operation}",
                        content=self.last_result,
                        format=self.last_format,
                        relative_dir=(operation,),
                        extension=(
                            self.current_file.suffix
                            if self.last_format == "source" and self.current_file
                            else None
                        ),
                    )
                )

        self.last_outputs = tuple(combined)
        self.last_diagram_set = GuiOutputSet(self.last_outputs)
        self.last_diagram_category = "analysis_results"
        self.last_result = self._output_set_text(self.last_diagram_set)
        self.last_format = "bundle"
        self._show_output_set(self.last_diagram_set)
        status = f"解析完了：{len(completed)}件、生成ファイル：{len(combined)}件"
        if failed:
            status += f"、失敗：{len(failed)}件"
            details = "\n".join(
                f"・{_OPERATION_LABELS[operation]}：{error}"
                for operation, error in failed
            )
            messagebox.showerror(PROJECT_NAME, f"一部の解析に失敗しました。\n\n{details}")
        self.status_var.set(status)

    def _run_operation(self, *, show_errors: bool = True) -> None:
        operation = self.operation_var.get()
        project_operation = operation in {
            _OPERATION_PACKAGE_DIAGRAM,
            _OPERATION_COMPONENT_DIAGRAM,
            _OPERATION_DEPLOYMENT_DIAGRAM,
        }
        if self.current_file is None and not (
            project_operation and self.base_path is not None and self.base_path.is_dir()
        ):
            self.last_run_succeeded = False
            messagebox.showinfo(PROJECT_NAME, "先に入力ファイルまたはプロジェクトフォルダーを選択してください。")
            return

        path = self.current_file or self.base_path
        if path is None:
            return
        language = self.service.detect_language(path) if path.is_file() else "project"
        self.status_var.set(f"{_OPERATION_LABELS[operation]}を実行中：{path.name}")
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
                        {"簡易": "simple", "詳細": "full"}.get(
                            self.deployment_mode_var.get(), "full"
                        ),
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
            self.last_run_succeeded = False
            self.last_error = str(exc)
            self.status_var.set(f"{_OPERATION_LABELS[operation]}に失敗しました。")
            if show_errors:
                messagebox.showerror(
                    PROJECT_NAME,
                    f"{_OPERATION_LABELS[operation]}に失敗しました。\n\n{exc}",
                )
            return

        self.last_run_succeeded = True
        self.last_result = content
        self.last_format = result_format
        if self.last_diagram_set is not None:
            self._show_output_set(self.last_diagram_set)
            count = len(self.last_outputs)
            self.status_var.set(f"解析完了：{count}件のファイルを生成しました。")
        else:
            self._clear_result_catalog()
            self._show_result(content)
            self.status_var.set("解析が完了しました。")

    @staticmethod
    def _output_set_text(result: object) -> str:
        outputs = getattr(result, "outputs", ())
        if not outputs:
            return "出力は生成されませんでした。\n"
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
            operation_label = _OPERATION_LABELS.get(prefix, prefix)
            label = f"{operation_label + ' / ' if operation_label else ''}{output.name}（{output.format}）"
            self.result_list.insert(tk.END, label)
        if self.last_outputs:
            self.result_list.selection_set(0)
            self.result_list.activate(0)
            self._show_result(getattr(self.last_outputs[0], "content", ""))
        else:
            self._show_result("出力は生成されませんでした。\n")

    def _on_result_selected(self, _event: object) -> None:
        selection = self.result_list.curselection()
        if not selection or not self.last_outputs:
            return
        index = selection[0]
        output = self.last_outputs[index]
        self._show_result(getattr(output, "content", ""))
        self.status_var.set(
            f"表示中：{getattr(output, 'name', 'output')}（{getattr(output, 'format', 'text')}）"
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
            messagebox.showinfo(PROJECT_NAME, "保存する前に解析を実行してください。")
            return

        selected = self.output_path_var.get().strip()
        if not selected:
            messagebox.showinfo(PROJECT_NAME, "画面上部の「出力先」で保存フォルダーを指定してください。")
            return
        try:
            output_root = Path(selected).expanduser()
            if self.last_outputs:
                paths = self.service.save_output_collection(
                    output_root,
                    self.last_outputs,
                    category="analysis_results",
                )
            else:
                extension = {
                    "mermaid": ".mmd",
                    "plantuml": ".puml",
                    "markdown": ".md",
                    "csv": ".csv",
                    "source": self.current_file.suffix if self.current_file else ".txt",
                }.get(self.last_format, ".txt")
                name = self.current_file.stem if self.current_file else "analysis-result"
                target = output_root / "analysis_results" / f"{name}{extension}"
                self.service.save_text(target, self.last_result)
                paths = (target,)
        except (OSError, ValueError) as exc:
            messagebox.showerror(PROJECT_NAME, f"解析結果を保存できませんでした。\n\n{exc}")
            return
        self.status_var.set(f"{len(paths)}件の結果を保存しました：{output_root / 'analysis_results'}")


def launch_gui(service: ApplicationService | None = None) -> None:
    """Start the Tkinter desktop application."""
    root = tk.Tk()
    AtlasTkApp(root, service)
    root.mainloop()
