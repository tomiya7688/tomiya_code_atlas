from pathlib import Path
import os
import subprocess
import sys
from tempfile import TemporaryDirectory

import pytest

from app import main


def test_cli_help_uses_utf8_under_legacy_windows_encoding():
    environment = os.environ.copy()
    environment["PYTHONIOENCODING"] = "cp1252"
    result = subprocess.run(
        [sys.executable, "app.py", "--help"],
        capture_output=True,
        text=True,
        encoding="utf-8",
        env=environment,
        check=False,
    )

    assert result.returncode == 0
    assert "使い方:" in result.stdout
    assert "オプション" in result.stdout


def test_comment_cli_writes_output_file():
    with TemporaryDirectory() as folder:
        source = Path(folder) / "sample.py"
        output = Path(folder) / "annotated.py"
        source.write_text("def load_config():\n    return {}\n", encoding="utf-8")
        assert main(["comment", str(source), "--language", "python", "--output", str(output)]) == 0
        assert "# Retrieves config." in output.read_text(encoding="utf-8")


def test_comment_cli_in_place():
    with TemporaryDirectory() as folder:
        source = Path(folder) / "sample.py"
        source.write_text("def load_config():\n    return {}\n", encoding="utf-8")
        assert main(["comment", str(source), "--in-place"]) == 0
        assert "# Retrieves config." in source.read_text(encoding="utf-8")


@pytest.mark.parametrize("extension", [".cpp", ".cc", ".cxx", ".hpp", ".hxx"])
def test_comment_cli_uses_shared_cpp_language_detection(extension: str):
    with TemporaryDirectory() as folder:
        source = Path(folder) / f"sample{extension}"
        output = Path(folder) / f"annotated{extension}"
        source.write_text("int load_config() { return 0; }\n", encoding="utf-8")

        assert main(["comment", str(source), "--output", str(output)]) == 0
        assert output.exists()


def test_comment_cli_explicit_language_overrides_extension():
    with TemporaryDirectory() as folder:
        source = Path(folder) / "sample.txt"
        output = Path(folder) / "annotated.py"
        source.write_text("def load_config():\n    return {}\n", encoding="utf-8")

        assert main(["comment", str(source), "--language", "python", "--output", str(output)]) == 0
        assert "# Retrieves config." in output.read_text(encoding="utf-8")


def test_comment_cli_rejects_unknown_extension_without_override():
    with TemporaryDirectory() as folder:
        source = Path(folder) / "sample.unknown"
        source.write_text("hello\n", encoding="utf-8")

        with pytest.raises(SystemExit) as error:
            main(["comment", str(source)])

        assert error.value.code == 2


def test_ci_cli_writes_mermaid_output_file():
    with TemporaryDirectory() as folder:
        source = Path(folder) / "build.yml"
        output = Path(folder) / "ci.mmd"
        source.write_text(
            """jobs:\n  test:\n    steps:\n      - name: pytest\n        run: pytest\n""",
            encoding="utf-8",
        )
        assert main(["ci", str(source), "--output", str(output)]) == 0
        assert "ci_test" in output.read_text(encoding="utf-8")


def test_deployment_cli_writes_diagram_folder():
    with TemporaryDirectory() as folder:
        root = Path(folder)
        output = root / "output"
        (root / "app.py").write_text("import requests\n", encoding="utf-8")
        (root / "compose.yaml").write_text(
            "services:\n  app:\n    depends_on: [db]\n  db:\n    image: postgres:17\n",
            encoding="utf-8",
        )
        assert main(["deployment", str(root), "--mode", "full", "--output-dir", str(output)]) == 0
        diagrams = list((output / "deployment_diagrams").glob("*.mmd"))
        assert diagrams
        assert any("postgres" in path.read_text(encoding="utf-8") for path in diagrams)


def test_timing_cli_writes_timing_chart_folder():
    with TemporaryDirectory() as folder:
        root = Path(folder)
        source = root / "sample.py"
        output = root / "output"
        source.write_text(
            "async def load():\n    await fetch()\n",
            encoding="utf-8",
        )

        assert main(["timing", str(source), "--output-dir", str(output)]) == 0
        diagrams = list((output / "timing_charts").glob("*.mmd"))
        assert len(diagrams) == 1
        assert "await fetch" in diagrams[0].read_text(encoding="utf-8")


def test_use_case_cli_writes_gui_originated_diagram_folder():
    with TemporaryDirectory() as folder:
        root = Path(folder)
        source = root / "ui.py"
        output = root / "output"
        source.write_text(
            """
class Window:
    def __init__(self):
        self.save = Button(text="Save", command=self.on_save)
    def on_save(self):
        self.persist()
    def persist(self):
        pass
""",
            encoding="utf-8",
        )

        assert main(["use-cases", str(source), "--output-dir", str(output)]) == 0
        diagrams = list((output / "use_case_diagrams").glob("*.mmd"))
        assert len(diagrams) == 1
        assert "Save" in diagrams[0].read_text(encoding="utf-8")


def test_class_diagram_cli_writes_plantuml_folder():
    with TemporaryDirectory() as folder:
        root = Path(folder)
        source = root / "sample.py"
        output = root / "output"
        source.write_text(
            "class Base: pass\nclass Child(Base): pass\n",
            encoding="utf-8",
        )

        assert main(
            [
                "class-diagram",
                str(source),
                "--renderer",
                "plantuml",
                "--output-dir",
                str(output),
            ]
        ) == 0
        diagrams = list((output / "class_diagrams").rglob("*.puml"))
        assert diagrams
        assert diagrams[0].parent.name.startswith("series_")
        assert diagrams[0].read_text(encoding="utf-8").startswith("@startuml\n")


def test_sequence_diagram_cli_writes_plantuml_with_returns():
    with TemporaryDirectory() as folder:
        root = Path(folder)
        source = root / "sample.py"
        output = root / "output"
        source.write_text(
            "def main(): helper()\ndef helper(): pass\n",
            encoding="utf-8",
        )

        assert main(
            [
                "sequence-diagram",
                str(source),
                "--renderer",
                "plantuml",
                "--show-returns",
                "--output-dir",
                str(output),
            ]
        ) == 0
        diagrams = list((output / "sequence_diagrams").rglob("*.puml"))
        assert diagrams
        assert diagrams[0].parent.name.startswith("series_")
        content = diagrams[0].read_text(encoding="utf-8")
        assert content.startswith("@startuml\n")
        assert " --> " in content
