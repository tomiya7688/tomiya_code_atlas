import sys
import tomllib
from pathlib import Path

import app
from app import _default_config_path, main
from Src.version import __version__


def test_version_does_not_require_gui(capsys):
    assert main(["--version"]) == 0
    assert f"Tomiya Code Atlas {app.PROJECT_VERSION}" in capsys.readouterr().out


def test_cli_help_is_japanese_and_keeps_command_identifiers(capsys):
    import pytest

    with pytest.raises(SystemExit) as error:
        main(["--help"])

    assert error.value.code == 0
    output = capsys.readouterr().out
    assert "使い方:" in output
    assert "オプション:" in output
    assert "コマンド:" in output
    assert "デスクトップ画面を起動" in output
    assert "class-diagram" in output
    assert "options:" not in output


def test_cli_unknown_source_type_has_a_japanese_error(capsys):
    import pytest

    with pytest.raises(SystemExit) as error:
        main(["comment", "sample.unsupported"])

    assert error.value.code == 2
    assert "対応していないファイル形式です" in capsys.readouterr().err


def test_cli_integer_option_errors_are_japanese(capsys):
    import pytest

    with pytest.raises(SystemExit) as error:
        main(["call-graph", "sample.py", "--max-depth", "deep"])

    assert error.value.code == 2
    output = capsys.readouterr().err
    assert "整数を指定してください" in output
    assert "invalid int value" not in output


def test_package_version_comes_from_canonical_source():
    pyproject = tomllib.loads((Path(__file__).resolve().parents[1] / "pyproject.toml").read_text(encoding="utf-8"))
    assert "version" in pyproject["project"]["dynamic"]
    assert pyproject["tool"]["setuptools"]["dynamic"]["version"]["attr"] == "Src.version.__version__"
    assert app.PROJECT_VERSION == __version__


def test_source_config_lives_under_config_directory():
    assert _default_config_path() == Path(app.__file__).resolve().parent / "config" / "tomiya-code-atlas.json"


def test_frozen_config_is_resolved_beside_exe_under_config(monkeypatch, tmp_path):
    exe = tmp_path / "tomiya-code-atlas.exe"
    monkeypatch.setattr(sys, "frozen", True, raising=False)
    monkeypatch.setattr(sys, "executable", str(exe))
    assert _default_config_path() == tmp_path / "config" / "tomiya-code-atlas.json"
