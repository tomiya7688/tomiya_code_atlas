from pathlib import Path

import pytest

from Src.data.backend_assets import (
    BACKEND_MANIFEST_VERSION,
    load_backend_manifest,
    resolve_backend_asset,
)
from Src.languages.backend import PARSER_BACKEND_CONTRACT_VERSION


ROOT = Path(__file__).resolve().parents[1]


def test_repository_backend_manifest_matches_parser_contract() -> None:
    manifest = load_backend_manifest(ROOT)

    assert manifest["manifest_version"] == BACKEND_MANIFEST_VERSION == "1"
    assert manifest["parser_backend_contract_version"] == PARSER_BACKEND_CONTRACT_VERSION
    assert isinstance(manifest["backends"], list)
    for backend in manifest["backends"]:
        assert backend["parser_engine"]
        assert backend["parser_version"]
        assert backend["grammar_version"]


def test_backend_manifest_requires_supported_version(tmp_path: Path) -> None:
    backend_dir = tmp_path / "backends"
    backend_dir.mkdir()
    (backend_dir / "manifest.json").write_text(
        '{"manifest_version":"999","backends":[]}',
        encoding="utf-8",
    )

    with pytest.raises(ValueError, match="Unsupported bundled backend manifest version"):
        load_backend_manifest(tmp_path)


def test_resolve_backend_asset_stays_inside_backend_directory(tmp_path: Path) -> None:
    expected = tmp_path / "backends" / "python" / "helper.exe"

    assert resolve_backend_asset("python/helper.exe", tmp_path) == expected.resolve()

    with pytest.raises(ValueError, match="escapes the bundled backend directory"):
        resolve_backend_asset("../outside.exe", tmp_path)
