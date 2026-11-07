"""Windowsのone-dir配布物をZIP化し、provenance用checksumを作成します。"""

from __future__ import annotations

import argparse
import hashlib
from pathlib import Path
import zipfile


BINARY_SUFFIXES = frozenset({".exe", ".dll", ".pyd", ".jar", ".so", ".dylib"})
PACKAGE_NAME = "tomiya-code-atlas-windows-x64.zip"
GENERATED_BINARY_MANIFEST = "SHA256SUMS.txt"
RUNTIME_IMAGE_PATH = Path("backends/java/runtime/lib/modules")


# {
#   責務: [
#     sha256_digest: ファイル全体をメモリへ読み込まずSHA-256を計算する,
#   ],
#   引数: [file_path: hashを計算するfile],
#   戻り値: 小文字hex形式のSHA-256ハッシュ値,
# }
def sha256_digest(file_path: Path) -> str:
    """ファイル全体をメモリへ保持せずSHA-256を返します。"""
    with file_path.open("rb") as file_stream:
        return hashlib.file_digest(file_stream, "sha256").hexdigest()


# {
#   責務: [
#     write_sha256_manifest: 指定ファイル群のSHA-256とルート相対パスをマニフェストへ保存する,
#   ],
#   引数: [root: 相対パスの起点, files: ハッシュを計算するファイル群, manifest: マニフェストの保存先],
#   戻り値: 作成したマニフェストのpath,
# }
def write_sha256_manifest(root: Path, files: list[Path], manifest: Path) -> Path:
    """指定ファイル群のSHA-256とルート相対パスをマニフェストへ保存します。"""
    lines = []
    for file_path in sorted(files, key=lambda item: item.as_posix()):
        relative_path = file_path.relative_to(root).as_posix()
        digest = sha256_digest(file_path)
        lines.append(f"{digest} *{relative_path}")

    write_text_atomically(manifest, "\n".join(lines) + "\n")
    return manifest


# {
#   責務: [
#     write_text_atomically: 既存内容を途中状態にせずテキストファイルを置き換える,
#   ],
#   引数: [file_path: 書き込み先, content: 保存するtext],
#   戻り値: なし,
# }
def write_text_atomically(file_path: Path, content: str) -> None:
    """途中内容を公開せず、LF改行のASCIIテキストでファイルを置き換えます。"""
    temporary_path = file_path.with_name(f"{file_path.name}.tmp")
    try:
        temporary_path.write_text(content, encoding="ascii", newline="\n")
        temporary_path.replace(file_path)
    finally:
        temporary_path.unlink(missing_ok=True)


# {
#   責務: [
#     package_windows_distribution: one-dir配布物、チェックサム、来歴証明対象一覧を作成する,
#   ],
#   引数: [distribution_directory: build済み配布フォルダー, release_directory: ZIPとマニフェストの保存フォルダー],
#   戻り値: 配布ZIP、配布チェックサム、来歴証明対象一覧のpath,
# }
def package_windows_distribution(
    distribution_directory: Path,
    release_directory: Path,
) -> tuple[Path, Path, Path]:
    """配布ZIP、チェックサム、来歴証明対象一覧を作成します。"""
    distribution_root = distribution_directory.resolve(strict=True)
    release_root = release_directory.resolve()
    workspace_root = Path.cwd().resolve()
    if not distribution_root.is_dir():
        raise ValueError(f"配布directoryではありません: {distribution_root}")
    if release_root == distribution_root or distribution_root in release_root.parents:
        raise ValueError("release directoryを配布directoryの内側に作成できません。")
    if not distribution_root.is_relative_to(workspace_root):
        raise ValueError(
            f"配布directoryはrepository内に指定してください: {distribution_root}"
        )
    if not release_root.is_relative_to(workspace_root):
        raise ValueError(
            f"release directoryはrepository内に指定してください: {release_root}"
        )

    release_root.mkdir(parents=True, exist_ok=True)
    archive_path = release_root / PACKAGE_NAME
    release_manifest = release_root / "SHA256SUMS.txt"
    subject_manifest = release_root / "ATTESTED-SUBJECTS.txt"
    binary_manifest = distribution_root / GENERATED_BINARY_MANIFEST
    for generated_path in (archive_path, release_manifest, subject_manifest):
        generated_path.unlink(missing_ok=True)
    binary_manifest.unlink(missing_ok=True)

    product_executable = distribution_root / "tomiya-code-atlas.exe"
    backend_manifest = distribution_root / "backends" / "manifest.json"
    if not product_executable.is_file():
        raise ValueError(f"Go CLIがありません: {product_executable}")
    if not backend_manifest.is_file():
        raise ValueError(f"parser backend manifestがありません: {backend_manifest}")

    distribution_paths = sorted(distribution_root.rglob("*"))
    if any(path.is_symlink() for path in distribution_paths):
        raise ValueError("配布directory内にsymbolic linkを含められません。")
    distribution_files = [
        path
        for path in distribution_paths
        if path.is_file() and path != binary_manifest
    ]
    binary_files = []
    for file_path in distribution_files:
        relative_path = file_path.relative_to(distribution_root)
        if (
            file_path.suffix.lower() in BINARY_SUFFIXES
            or relative_path.as_posix().casefold()
            == RUNTIME_IMAGE_PATH.as_posix().casefold()
        ):
            binary_files.append(file_path)
    if not binary_files:
        raise ValueError("SHA-256 manifestへ記録するbinaryがありません。")

    try:
        write_sha256_manifest(distribution_root, binary_files, binary_manifest)

        temporary_archive = archive_path.with_name(f"{archive_path.name}.tmp")
        archive_files = sorted(
            [*distribution_files, binary_manifest],
            key=lambda item: item.relative_to(distribution_root).as_posix(),
        )
        with zipfile.ZipFile(
            temporary_archive, mode="w", compression=zipfile.ZIP_DEFLATED
        ) as archive:
            for file_path in archive_files:
                archive.write(
                    file_path,
                    file_path.relative_to(distribution_root).as_posix(),
                )
        temporary_archive.replace(archive_path)
        write_sha256_manifest(release_root, [archive_path], release_manifest)

        subject_files = sorted(
            [archive_path, release_manifest, binary_manifest, *binary_files],
            key=lambda item: item.resolve().relative_to(workspace_root).as_posix(),
        )
        subject_lines = []
        for file_path in subject_files:
            relative_path = file_path.resolve().relative_to(workspace_root).as_posix()
            digest = sha256_digest(file_path)
            subject_lines.append(f"{digest} *{relative_path}")
        write_text_atomically(subject_manifest, "\n".join(subject_lines) + "\n")
    except Exception:
        for generated_path in (
            archive_path,
            release_manifest,
            subject_manifest,
            binary_manifest,
        ):
            generated_path.unlink(missing_ok=True)
        raise
    finally:
        archive_path.with_name(f"{archive_path.name}.tmp").unlink(missing_ok=True)
    return archive_path, release_manifest, subject_manifest


def main() -> int:
    """CLI引数から配布物を作成し、出力パスを表示します。"""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--distribution-directory", type=Path, default=Path(".build/dist")
    )
    parser.add_argument(
        "--release-directory", type=Path, default=Path(".build/release")
    )
    arguments = parser.parse_args()
    archive_path, release_manifest, subject_manifest = package_windows_distribution(
        arguments.distribution_directory,
        arguments.release_directory,
    )
    print(f"配布ZIP: {archive_path}")
    print(f"SHA-256 manifest: {release_manifest}")
    print(f"attestation subject一覧: {subject_manifest}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
