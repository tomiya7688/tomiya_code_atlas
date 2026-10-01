from __future__ import annotations

import struct
from pathlib import Path

from Src.ui.tk_app import _application_icon_root


ROOT = Path(__file__).resolve().parents[1]
CHARACTER_ASSETS = ROOT / "assets" / "characters" / "atlas-kun"


def _png_dimensions(path: Path) -> tuple[int, int]:
    data = path.read_bytes()
    assert data.startswith(b"\x89PNG\r\n\x1a\n")
    return struct.unpack_from(">II", data, 16)


def test_source_gui_uses_the_documented_atlas_kun_assets() -> None:
    assert _application_icon_root() == CHARACTER_ASSETS
    assert _png_dimensions(CHARACTER_ASSETS / "atlas-kun-1920.png") == (1920, 1920)
    assert _png_dimensions(CHARACTER_ASSETS / "atlas-kun-64.png") == (64, 64)
    assert _png_dimensions(CHARACTER_ASSETS / "atlas-kun-256.png") == (256, 256)


def test_atlas_kun_assets_include_the_applicable_character_license() -> None:
    notice = (CHARACTER_ASSETS / "README.md").read_text(encoding="utf-8")
    license_text = (CHARACTER_ASSETS / "LICENSE.md").read_text(encoding="utf-8")

    assert "Tomiya_character_lisence" in notice
    assert "Tomiya Character License v1.0.1" in license_text
    assert "Atras.png" in notice
