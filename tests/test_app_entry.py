import sys
import tomllib
from pathlib import Path

import app
from app import _default_config_path, main
from Src.version import __version__


def test_version_does_not_require_gui(capsys):
    assert main(["--version"]) == 0
    assert f"Tomiya Code Atlas {app.PROJECT_VERSION}" in capsys.readouterr().out


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
