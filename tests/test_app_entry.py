import sys
from pathlib import Path

import app
from app import _default_config_path, main


def test_version_does_not_require_gui(capsys):
    assert main(["--version"]) == 0
    assert "Tomiya Code Atlas 0.1.0" in capsys.readouterr().out


def test_source_config_lives_under_config_directory():
    assert _default_config_path() == Path(app.__file__).resolve().parent / "config" / "tomiya-code-atlas.json"


def test_frozen_config_is_resolved_beside_exe_under_config(monkeypatch, tmp_path):
    exe = tmp_path / "tomiya-code-atlas.exe"
    monkeypatch.setattr(sys, "frozen", True, raising=False)
    monkeypatch.setattr(sys, "executable", str(exe))
    assert _default_config_path() == tmp_path / "config" / "tomiya-code-atlas.json"
