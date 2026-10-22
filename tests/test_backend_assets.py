from pathlib import Path

import pytest

from Src.data.backend_assets import (
    BACKEND_MANIFEST_VERSION,
    backend_asset_root,
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
    assert manifest["source_root"] == "parser-backends"
    assert manifest["build_output_root"] == ".build/dist/backends"
    assert not Path(manifest["source_root"]).is_absolute()
    assert ".." not in Path(manifest["source_root"]).parts
    assert not Path(manifest["build_output_root"]).is_absolute()
    assert ".." not in Path(manifest["build_output_root"]).parts
    assert not (ROOT / "backend-src").exists()
    assert not (ROOT / "backends").exists()

    backends_by_language = {
        backend["language"]: backend for backend in manifest["backends"]
    }
    for backend in manifest["backends"]:
        assert backend["parser_engine"]
        assert backend["parser_version"]
        assert backend["grammar_version"]
        source_relative = Path(backend["source_directory"])
        assert not source_relative.is_absolute()
        assert ".." not in source_relative.parts
        source_directory = Path(manifest["source_root"]) / source_relative
        assert (ROOT / source_directory).is_dir()
        for relative_path in backend["source_licenses"]:
            license_relative = Path(relative_path)
            assert not license_relative.is_absolute()
            assert ".." not in license_relative.parts
            assert (ROOT / manifest["source_root"] / license_relative).is_file()
        distribution_paths = [backend["path"], *backend.get("licenses", [])]
        for relative_path in distribution_paths:
            assert not Path(relative_path).is_absolute()
            assert ".." not in Path(relative_path).parts

    assert (
        backends_by_language["gdscript"]["source_directory"]
        == backends_by_language["python"]["source_directory"]
    )
    assert backends_by_language["gdscript"]["source_shared_with"] == "python"

    generated_file_suffixes = {".dll", ".exe", ".jar"}
    generated_directory_names = {"bin", "obj", "runtime", "target"}
    for path in (ROOT / "parser-backends").rglob("*"):
        if path.is_file():
            assert path.suffix.lower() not in generated_file_suffixes
        if path.is_dir():
            assert path.name.lower() not in generated_directory_names


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


def test_source_checkout_uses_manifest_build_output_root(tmp_path: Path) -> None:
    source_root = tmp_path / "parser-backends"
    source_root.mkdir()
    (source_root / "manifest.json").write_text(
        '{"build_output_root":".build/dist/backends"}',
        encoding="utf-8",
    )
    distribution_root = tmp_path / ".build" / "dist" / "backends"
    distribution_root.mkdir(parents=True)
    (distribution_root / "manifest.json").write_text("{}", encoding="utf-8")

    assert backend_asset_root(tmp_path) == distribution_root
    assert resolve_backend_asset("go/tomiya-go-backend.exe", tmp_path) == (
        distribution_root / "go" / "tomiya-go-backend.exe"
    )
