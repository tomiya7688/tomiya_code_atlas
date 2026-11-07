"""Test checksums, ZIP contents, and exact attestation subjects for distribution."""

from __future__ import annotations

import hashlib
from pathlib import Path
import zipfile

import pytest

import tools.package_windows_distribution as distribution_package
from tools.package_windows_distribution import package_windows_distribution


# {
#   責務: [
#     test_distribution_package_has_verified_binary_hashes: ZIPの全file、binary checksum、attestation subjectが一致することを検証する,
#   ],
# }
def test_distribution_package_has_verified_binary_hashes(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.chdir(tmp_path)
    distribution = tmp_path / ".build/dist"
    release = tmp_path / ".build/release"
    backend_directory = distribution / "backends"
    hidden_directory = distribution / ".metadata"
    backend_directory.mkdir(parents=True)
    hidden_directory.mkdir()
    executable = distribution / "tomiya-code-atlas.exe"
    helper = backend_directory / "helper.dll"
    executable.write_bytes(b"cli test binary")
    helper.write_bytes(b"parser test binary")
    (backend_directory / "manifest.json").write_text("{}", encoding="utf-8")
    hidden_file = hidden_directory / "build-info.json"
    hidden_file.write_text("{}", encoding="utf-8")
    (distribution / "SHA256SUMS.txt").write_text("stale manifest", encoding="ascii")

    archive_path, checksum_path, subjects_path = package_windows_distribution(
        distribution,
        release,
    )

    with zipfile.ZipFile(archive_path) as archive:
        archive_paths = set(archive.namelist())
        assert "tomiya-code-atlas.exe" in archive_paths
        assert "backends/helper.dll" in archive_paths
        assert "backends/manifest.json" in archive_paths
        assert ".metadata/build-info.json" in archive_paths
        binary_checksum = archive.read("SHA256SUMS.txt").decode("ascii").splitlines()
    assert binary_checksum == [
        f"{hashlib.sha256(helper.read_bytes()).hexdigest()} *backends/helper.dll",
        f"{hashlib.sha256(executable.read_bytes()).hexdigest()} *tomiya-code-atlas.exe",
    ]
    assert checksum_path.read_text(encoding="ascii") == (
        f"{hashlib.sha256(archive_path.read_bytes()).hexdigest()} "
        f"*{archive_path.name}\n"
    )
    attested_paths = {
        line.split(" *", maxsplit=1)[1]
        for line in subjects_path.read_text(encoding="ascii").splitlines()
    }
    assert attested_paths == {
        ".build/release/tomiya-code-atlas-windows-x64.zip",
        ".build/release/SHA256SUMS.txt",
        ".build/dist/SHA256SUMS.txt",
        ".build/dist/tomiya-code-atlas.exe",
        ".build/dist/backends/helper.dll",
    }


# {
#   責務: [
#     test_distribution_package_rejects_missing_build_outputs: Go CLIまたはbackend manifestの欠落をエラーにする,
#   ],
# }
def test_distribution_package_rejects_missing_build_outputs(tmp_path: Path) -> None:
    distribution = tmp_path / "dist"
    distribution.mkdir()

    with pytest.raises(ValueError, match="Go CLIがありません"):
        package_windows_distribution(distribution, tmp_path / "release")

    (distribution / "tomiya-code-atlas.exe").write_bytes(b"cli")
    with pytest.raises(ValueError, match="parser backend manifestがありません"):
        package_windows_distribution(distribution, tmp_path / "release")


# {
#   責務: [
#     test_distribution_package_removes_stale_archive_after_failure: 失敗後に古いZIPや途中fileを残さない,
#   ],
# }
def test_distribution_package_removes_stale_archive_after_failure(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    distribution = tmp_path / "dist"
    backend_directory = distribution / "backends"
    backend_directory.mkdir(parents=True)
    (distribution / "tomiya-code-atlas.exe").write_bytes(b"cli")
    (backend_directory / "manifest.json").write_text("{}", encoding="utf-8")
    release = tmp_path / "release"
    release.mkdir()
    archive_path = release / "tomiya-code-atlas-windows-x64.zip"
    archive_path.write_bytes(b"stale archive")

    def fail_to_create_archive(*args: object, **kwargs: object) -> None:
        temporary_archive = Path(args[0])
        temporary_archive.write_bytes(b"partial archive")
        raise OSError("archive write failed")

    monkeypatch.setattr(distribution_package.zipfile, "ZipFile", fail_to_create_archive)
    with pytest.raises(OSError, match="archive write failed"):
        package_windows_distribution(distribution, release)

    assert not archive_path.exists()
    assert not (release / "tomiya-code-atlas-windows-x64.zip.tmp").exists()
