"""Install and exercise built Python package artifacts in a clean virtual environment."""

from __future__ import annotations

import os
import subprocess
import sys
import tempfile
from pathlib import Path


def run(command: list[str], *, cwd: Path, env: dict[str, str]) -> subprocess.CompletedProcess[str]:
    completed = subprocess.run(
        command,
        cwd=cwd,
        env=env,
        text=True,
        encoding="utf-8",
        errors="replace",
        capture_output=True,
        check=False,
        timeout=120,
    )
    if completed.returncode:
        raise SystemExit(
            f"Command failed ({completed.returncode}): {command!r}\n"
            f"stdout:\n{completed.stdout}\nstderr:\n{completed.stderr}"
        )
    return completed


def require_markers(path: Path, *markers: str) -> None:
    if not path.is_file():
        raise SystemExit(f"Expected artifact output was not created: {path}")
    content = path.read_text(encoding="utf-8").replace("\r\n", "\n")
    missing = [marker for marker in markers if marker not in content]
    if missing:
        raise SystemExit(f"{path.name} is missing expected output markers {missing!r}:\n{content}")


def require_golden(actual: Path, name: str) -> None:
    golden = Path(__file__).resolve().parents[1] / "tests" / "fixtures" / "distribution_e2e" / "golden" / name
    if not golden.is_file():
        raise SystemExit(f"Expected output golden was not found: {golden}")
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
        raise SystemExit(f"Output differed from golden {name}:\n{difference}")

def verify(artifact: Path) -> None:
    if not artifact.is_file() or not (artifact.name.endswith(".whl") or artifact.name.endswith(".tar.gz")):
        raise SystemExit(f"wheel or source archive not found: {artifact}")

    with tempfile.TemporaryDirectory(prefix="tomiya-package-e2e-") as temp:
        root = Path(temp)
        venv = root / "venv"
        workspace = root / "run"
        workspace.mkdir()
        subprocess.run([sys.executable, "-m", "venv", str(venv)], check=True, timeout=120)
        scripts = venv / ("Scripts" if os.name == "nt" else "bin")
        python = scripts / ("python.exe" if os.name == "nt" else "python")
        cli = scripts / ("tomiya-code-atlas.exe" if os.name == "nt" else "tomiya-code-atlas")
        env = os.environ.copy()
        env.pop("PYTHONPATH", None)
        env.pop("PYTHONHOME", None)

        run([str(python), "-m", "pip", "install", str(artifact)], cwd=workspace, env=env)
        run(
            [str(python), "-c", "from Src.analyzers import CallGraph; from Src.generators import CommentGenerator; import yaml"],
            cwd=workspace,
            env=env,
        )
        version = run([str(cli), "--version"], cwd=workspace, env=env)
        if "Tomiya Code Atlas" not in version.stdout:
            raise SystemExit(f"Unexpected installed CLI version output: {version.stdout!r}")

        source = workspace / "sample.py"
        source.write_text("def load_config():\n    return {}\n", encoding="utf-8")
        comment_output = workspace / "commented.py"
        run(
            [str(cli), "comment", str(source), "--output", str(comment_output)],
            cwd=workspace,
            env=env,
        )
        require_markers(comment_output, "# Retrieves config.", "def load_config():")
        require_golden(comment_output, "comment-python.py")

        workflow = workspace / "workflow.yml"
        workflow.write_text(
            "name: Build\non: [push]\njobs:\n  test:\n    steps:\n"
            "      - name: pytest\n        run: pytest\n",
            encoding="utf-8",
        )
        ci_output = workspace / "ci.mmd"
        run([str(cli), "ci", str(workflow), "--output", str(ci_output)], cwd=workspace, env=env)
        require_markers(ci_output, "flowchart LR", "ci_test")
        require_golden(ci_output, "ci.mmd")

        class_source = workspace / "classes.py"
        class_source.write_text(
            "class Base:\n    pass\n\n"
            "class Worker(Base):\n"
            "    def run(self):\n        return transform()\n\n"
            "def transform():\n    return finalize()\n\n"
            "def finalize():\n    return True\n",
            encoding="utf-8",
        )
        diagrams = workspace / "class-diagrams"
        run([str(cli), "class-diagram", str(class_source), "--output-dir", str(diagrams)], cwd=workspace, env=env)
        outputs = sorted(diagrams.rglob("*.mmd"))
        if not outputs:
            raise SystemExit("Installed package CLI did not produce a class diagram.")
        require_markers(outputs[0], "classDiagram", "Base", "Worker")
        require_golden(outputs[0], "class-diagram.mmd")

    print(f"Installed package E2E passed: {artifact.name}")


def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("usage: verify_wheel.py PATH_TO_WHEEL_OR_SDIST")
    verify(Path(sys.argv[1]).resolve())
    return 0


if __name__ == "__main__":
    raise SystemExit(main())