"""配布ZIPの内容、チェックサム、来歴証明対象を検証します。"""

from __future__ import annotations

import hashlib
from pathlib import Path
import zipfile

import pytest

import scripts.build.package_windows_distribution as distribution_package
from scripts.build.package_windows_distribution import package_windows_distribution


# {
#   責務: [
#     test_distribution_package_has_verified_binary_hashes: ZIP、実行ファイルのチェックサム、来歴証明対象のハッシュが一致することを検証する,
#   ],
# }
def test_distribution_package_has_verified_binary_hashes(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.chdir(tmp_path)
    distribution = tmp_path / ".build/dist"
    release = tmp_path / ".build/release"
    backend_directory = distribution / "backends"
    runtime_image = distribution / "backends/java/runtime/lib/modules"
    hidden_directory = distribution / ".metadata"
    backend_directory.mkdir(parents=True)
    runtime_image.parent.mkdir(parents=True)
    hidden_directory.mkdir()
    executable = distribution / "tomiya-code-atlas.exe"
    helper = backend_directory / "helper.dll"
    executable.write_bytes(b"cli test binary")
    helper.write_bytes(b"parser test binary")
    runtime_image.write_bytes(b"java runtime image")
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
        assert "backends/java/runtime/lib/modules" in archive_paths
        assert "backends/manifest.json" in archive_paths
        assert ".metadata/build-info.json" in archive_paths
        binary_checksum = archive.read("SHA256SUMS.txt").decode("ascii").splitlines()
    assert binary_checksum == [
        f"{hashlib.sha256(helper.read_bytes()).hexdigest()} *backends/helper.dll",
        f"{hashlib.sha256(runtime_image.read_bytes()).hexdigest()} *backends/java/runtime/lib/modules",
        f"{hashlib.sha256(executable.read_bytes()).hexdigest()} *tomiya-code-atlas.exe",
    ]
    assert b"\r" not in (distribution / "SHA256SUMS.txt").read_bytes()
    assert b"\r" not in checksum_path.read_bytes()
    assert checksum_path.read_text(encoding="ascii") == (
        f"{hashlib.sha256(archive_path.read_bytes()).hexdigest()} "
        f"*{archive_path.name}\n"
    )
    assert b"\r" not in subjects_path.read_bytes()
    attested_subjects = {}
    for line in subjects_path.read_text(encoding="ascii").splitlines():
        digest, subject_path = line.split(" *", maxsplit=1)
        assert subject_path not in attested_subjects
        attested_subjects[subject_path] = digest
    expected_subject_files = {
        ".build/release/tomiya-code-atlas-windows-x64.zip": archive_path,
        ".build/release/SHA256SUMS.txt": checksum_path,
        ".build/dist/SHA256SUMS.txt": distribution / "SHA256SUMS.txt",
        ".build/dist/tomiya-code-atlas.exe": executable,
        ".build/dist/backends/helper.dll": helper,
        ".build/dist/backends/java/runtime/lib/modules": runtime_image,
    }
    assert set(attested_subjects) == set(expected_subject_files)
    assert attested_subjects == {
        subject_path: hashlib.sha256(file_path.read_bytes()).hexdigest()
        for subject_path, file_path in expected_subject_files.items()
    }


# {
#   責務: [
#     test_distribution_package_rejects_missing_build_outputs: Go CLIまたはparser manifestの欠落をエラーにする,
#   ],
# }
def test_distribution_package_rejects_missing_build_outputs(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.chdir(tmp_path)
    distribution = tmp_path / "dist"
    distribution.mkdir()
    release = tmp_path / "release"
    release.mkdir()
    stale_archive = release / "tomiya-code-atlas-windows-x64.zip"
    stale_archive.write_bytes(b"stale archive")
    stale_binary_manifest = distribution / "SHA256SUMS.txt"
    stale_binary_manifest.write_text("stale manifest", encoding="ascii")

    with pytest.raises(ValueError, match="Go CLIがありません"):
        package_windows_distribution(distribution, release)
    assert not stale_archive.exists()
    assert not stale_binary_manifest.exists()

    (distribution / "tomiya-code-atlas.exe").write_bytes(b"cli")
    with pytest.raises(ValueError, match="parser backend manifestがありません"):
        package_windows_distribution(distribution, release)


# {
#   責務: [
#     test_distribution_package_rejects_external_paths: repository外のbuild folderを変更前に拒否する,
#   ],
# }
def test_distribution_package_rejects_external_paths(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.chdir(tmp_path)
    external_distribution = tmp_path.parent / f"{tmp_path.name}-dist"
    external_distribution.mkdir()

    with pytest.raises(ValueError, match="配布directoryはrepository内"):
        package_windows_distribution(external_distribution, tmp_path / "release")

    assert not (external_distribution / "SHA256SUMS.txt").exists()


# {
#   責務: [
#     test_distribution_package_removes_stale_archive_after_failure: 失敗後に古いZIPや途中ファイルを残さない,
#   ],
# }
def test_distribution_package_removes_stale_archive_after_failure(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.chdir(tmp_path)
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
