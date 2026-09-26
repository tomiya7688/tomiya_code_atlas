"""Exercise a built Tomiya Code Atlas distribution from an isolated working directory."""

from __future__ import annotations

import argparse
import difflib
import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path


LANGUAGE_FIXTURES: tuple[tuple[str, str, str, str], ...] = (
    ("python", ".py", "def load_config():\n    return {}\n", "# Retrieves config."),
    ("gdscript", ".gd", "class_name LoadConfig\nfunc load_config():\n    return {}\n", "# Retrieves config."),
    ("csharp", ".cs", "using UnityEngine;\npublic class LoadConfig : MonoBehaviour { public void Start() {} }\n", "// Coordinates this Unity load config component."),
    ("cpp", ".cpp", "class LoadConfig { public: void load() {} };\n", "// Groups behavior related to load config."),
    ("java", ".java", "class LoadConfig { void load() {} }\n", "// Groups behavior related to load config."),
    ("go", ".go", "package e2e\ntype LoadConfig struct {}\nfunc Load() {}\n", "// Groups behavior related to load config."),
)


class DistributionCheckError(RuntimeError):
    pass


def _environment(workspace: Path) -> dict[str, str]:
    """Keep execution in the packaged app and OS runtime, away from dev tools."""
    env = {
        key: value
        for key, value in os.environ.items()
        if key.upper() in {"SYSTEMROOT", "WINDIR", "TEMP", "TMP", "USERPROFILE"}
    }
    system_root = env.get("SystemRoot", r"C:\Windows")
    env["PATH"] = os.pathsep.join((str(workspace), str(Path(system_root) / "System32"), system_root))
    env["TEMP"] = str(workspace)
    env["TMP"] = str(workspace)
    return env


def _run(exe: Path, workspace: Path, *arguments: str, expected: int = 0) -> subprocess.CompletedProcess[str]:
    command = [str(exe), *arguments]
    result = subprocess.run(
        command,
        cwd=workspace,
        env=_environment(workspace),
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
        timeout=120,
        check=False,
    )
    if result.returncode != expected:
        raise DistributionCheckError(
            f"Command returned {result.returncode}, expected {expected}: {arguments!r}\n"
            f"stdout:\n{result.stdout}\nstderr:\n{result.stderr}"
        )
    return result


def _require(path: Path, *markers: str) -> str:
    if not path.is_file():
        raise DistributionCheckError(f"Expected output file was not created: {path}")
    text = path.read_text(encoding="utf-8").replace("\r\n", "\n")
    missing = [marker for marker in markers if marker not in text]
    if missing:
        raise DistributionCheckError(f"{path.name} is missing expected output markers: {missing!r}\n{text}")
    return text


def _assert_golden(actual: Path, golden_name: str) -> None:
    golden = Path(__file__).resolve().parents[1] / "tests" / "fixtures" / "distribution_e2e" / "golden" / golden_name
    if not golden.is_file():
        raise DistributionCheckError(f"Expected output golden is missing: {golden}")
    actual_text = actual.read_text(encoding="utf-8").replace("\r\n", "\n")
    expected_text = golden.read_text(encoding="utf-8").replace("\r\n", "\n")
    if actual_text != expected_text:
        difference = "".join(
            difflib.unified_diff(
                expected_text.splitlines(keepends=True),
                actual_text.splitlines(keepends=True),
                fromfile=str(golden),
                tofile=str(actual),
            )
        )
        raise DistributionCheckError(f"Output differed from golden {golden_name}:\n{difference}")

def _only_output(root: Path, suffix: str) -> Path:
    matches = sorted(root.rglob(f"*{suffix}"))
    if len(matches) != 1:
        raise DistributionCheckError(f"Expected one {suffix} output below {root}, found {len(matches)}")
    return matches[0]


def _verify_language_comments(exe: Path, workspace: Path) -> None:
    for language, extension, source_text, marker in LANGUAGE_FIXTURES:
        source = workspace / f"comment-{language}{extension}"
        output = workspace / f"comment-{language}.out{extension}"
        source.write_text(source_text, encoding="utf-8")
        _run(exe, workspace, "comment", str(source), "--language", language, "--output", str(output))
        _require(output, marker)
        if language == "python":
            _assert_golden(output, "comment-python.py")


def _verify_semantic_backends(exe: Path, workspace: Path) -> None:
    python_source = workspace / "diagram-python.py"
    python_source.write_text("class LoadConfig:\n    def load(self):\n        return True\n", encoding="utf-8")
    python_output = workspace / "class-python"
    _run(exe, workspace, "class-diagram", str(python_source), "--output-dir", str(python_output))
    _require(_only_output(python_output, ".mmd"), "classDiagram", "LoadConfig")

    cases = (
        ("gdscript", ".gd", "class_name LoadConfig\nextends RefCounted\nfunc load():\n    return true\n"),
        ("csharp", ".cs", "public class LoadConfig { public bool Load() => true; }\n"),
        ("java", ".java", "public class LoadConfig { public boolean load() { return true; } }\n"),
    )
    for language, extension, source_text in cases:
        source = workspace / f"backend-{language}{extension}"
        source.write_text(source_text, encoding="utf-8")
        output = _run(exe, workspace, "backend-smoke", language, "--source", str(source))
        if not output.stdout.strip():
            raise DistributionCheckError(f"{language} backend did not identify itself after parsing real input")

    go_source = "package e2e\ntype LoadConfig struct {}\nfunc (c LoadConfig) Load() bool { return true }\n"
    helper = exe.parent / "backends" / "go" / "tomiya-go-backend.exe"
    if not helper.is_file():
        raise DistributionCheckError(f"Bundled Go parser helper is missing: {helper}")
    request = {
        "contract_version": "1",
        "request_id": "distribution-go-e2e",
        "operation": "parse",
        "language": "go",
        "source": go_source,
        "path": "fixture.go",
    }
    result = subprocess.run(
        [str(helper)],
        cwd=workspace,
        env=_environment(workspace),
        input=json.dumps(request),
        capture_output=True,
        text=True,
        encoding="utf-8",
        timeout=120,
        check=False,
    )
    if result.returncode != 0:
        raise DistributionCheckError(f"Go helper failed: {result.stderr}")
    response = json.loads(result.stdout)
    entities = response.get("ir", {}).get("entities", ())
    if response.get("request_id") != request["request_id"] or not response.get("ok"):
        raise DistributionCheckError(f"Go helper rejected the fixture: {response!r}")
    if not any(entity.get("name") == "LoadConfig" for entity in entities):
        raise DistributionCheckError(f"Go helper omitted the fixture type: {response!r}")

def _verify_python_features(exe: Path, workspace: Path) -> None:
    source = workspace / "sample.py"
    source.write_text(
        "class Base:\n    pass\n\n"
        "class Worker(Base):\n"
        "    def run(self):\n        return transform()\n\n"
        "def transform():\n    return finalize()\n\n"
        "def finalize():\n    return True\n",
        encoding="utf-8",
    )

    ci_source = workspace / "workflow.yml"
    ci_source.write_text(
        "name: Build\non: [push]\njobs:\n  test:\n    steps:\n"
        "      - name: pytest\n        run: pytest\n",
        encoding="utf-8",
    )
    ci_output = workspace / "ci.mmd"
    _run(exe, workspace, "ci", str(ci_source), "--output", str(ci_output))
    _require(ci_output, "flowchart LR", "ci_test")
    _assert_golden(ci_output, "ci.mmd")

    for command, marker, extension in (
        ("call-graph", "transform", ".mmd"),
        ("class-diagram", "Worker", ".mmd"),
        ("sequence-diagram", "transform", ".mmd"),
        ("timing", "run", ".mmd"),
    ):
        output_dir = workspace / f"{command}-mermaid"
        _run(exe, workspace, command, str(source), "--output-dir", str(output_dir))
        outputs = sorted(output_dir.rglob(f"*{extension}"))
        if not outputs:
            raise DistributionCheckError(f"{command} did not produce a {extension} file")
        combined = "\n".join(path.read_text(encoding="utf-8") for path in outputs)
        if marker not in combined:
            raise DistributionCheckError(f"{command} output omitted {marker!r}: {combined}")
        if command == "timing":
            for output in outputs:
                _assert_golden(output, f"timing-{output.name}")
        else:
            golden_name = {"call-graph": "call-graph.mmd", "class-diagram": "class-diagram.mmd", "sequence-diagram": "sequence-diagram.mmd"}[command]
            _assert_golden(_only_output(output_dir, ".mmd"), golden_name)

    for command, marker in (("class-diagram", "@startuml"), ("sequence-diagram", "@startuml")):
        output_dir = workspace / f"{command}-plantuml"
        _run(exe, workspace, command, str(source), "--renderer", "plantuml", "--output-dir", str(output_dir))
        outputs = sorted(output_dir.rglob("*.puml"))
        if not outputs:
            raise DistributionCheckError(f"{command} did not produce PlantUML output")
        if marker not in outputs[0].read_text(encoding="utf-8"):
            raise DistributionCheckError(f"{command} PlantUML output omitted {marker!r}")
        _assert_golden(outputs[0], f"{command}.puml")

    ui_source = workspace / "ui.py"
    ui_source.write_text(
        "class Window:\n"
        "    def __init__(self):\n        self.save = Button(text='Save', command=self.on_save)\n"
        "    def on_save(self):\n        self.persist()\n"
        "    def persist(self):\n        pass\n",
        encoding="utf-8",
    )
    use_case_dir = workspace / "use-cases"
    _run(exe, workspace, "use-cases", str(ui_source), "--output-dir", str(use_case_dir))
    use_case_output = _only_output(use_case_dir, ".mmd")
    _require(use_case_output, "Save")
    _assert_golden(use_case_output, "use-cases.mmd")

    deploy_root = workspace / "deploy-project"
    deploy_root.mkdir()
    (deploy_root / "compose.yaml").write_text(
        "services:\n  app:\n    depends_on: [db]\n  db:\n    image: postgres:17\n",
        encoding="utf-8",
    )
    deploy_dir = workspace / "deployment"
    _run(exe, workspace, "deployment", str(deploy_root), "--mode", "full", "--output-dir", str(deploy_dir))
    deployment_outputs = sorted(deploy_dir.rglob("*.mmd"))
    if not deployment_outputs or not any("postgres" in path.read_text(encoding="utf-8") for path in deployment_outputs):
        raise DistributionCheckError("deployment output did not include the fixture database")
    _assert_golden(_only_output(deploy_dir, ".mmd"), "deployment.mmd")


def _verify_cli_contract(exe: Path, workspace: Path) -> None:
    version = _run(exe, workspace, "--version")
    if "Tomiya Code Atlas" not in version.stdout:
        raise DistributionCheckError(f"Unexpected version output: {version.stdout!r}")
    help_result = _run(exe, workspace, "--help")
    for command in ("comment", "ci", "deployment", "timing", "use-cases", "call-graph", "class-diagram", "sequence-diagram"):
        if command not in help_result.stdout:
            raise DistributionCheckError(f"CLI help omitted subcommand {command!r}")
    _run(exe, workspace, "invalid-command", expected=2)
    _run(exe, workspace, "comment", expected=2)
    _run(exe, workspace, "comment", str(workspace / "missing.py"), expected=1)
    _run(exe, workspace, "comment", str(workspace / "sample.py"), "--language", "unsupported", expected=1)

    source = workspace / "in-place.py"
    source.write_text("def load_config():\n    return {}\n", encoding="utf-8")
    _run(exe, workspace, "comment", str(source), "--in-place")
    _require(source, "# Retrieves config.")


def _verify_gui_launch(exe: Path, workspace: Path) -> None:
    process = subprocess.Popen(
        [str(exe)],
        cwd=workspace,
        env=_environment(workspace),
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        creationflags=getattr(subprocess, "CREATE_NO_WINDOW", 0),
    )
    try:
        try:
            exit_code = process.wait(timeout=3)
        except subprocess.TimeoutExpired:
            return
        raise DistributionCheckError(f"GUI process exited during startup with code {exit_code}")
    finally:
        if process.poll() is None:
            process.terminate()
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=5)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--exe", required=True, type=Path, help="Path to the built onedir executable")
    parser.add_argument("--gui", action="store_true", help="Also perform a bounded GUI startup smoke test")
    args = parser.parse_args()
    exe = args.exe.resolve()
    if not exe.is_file():
        parser.error(f"distribution executable does not exist: {exe}")
    if sys.platform != "win32":
        parser.error("the onedir Windows distribution can only be verified on Windows")

    with tempfile.TemporaryDirectory(prefix="tomiya-distribution-e2e-") as temp:
        workspace = Path(temp)
        try:
            _verify_cli_contract(exe, workspace)
            _verify_language_comments(exe, workspace)
            _verify_semantic_backends(exe, workspace)
            _verify_python_features(exe, workspace)
            if args.gui:
                _verify_gui_launch(exe, workspace)
        except (DistributionCheckError, OSError, ValueError, subprocess.SubprocessError) as error:
            print(f"Distribution E2E failed: {error}", file=sys.stderr)
            return 1

    print("Packaged distribution E2E passed (isolated work directory).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())