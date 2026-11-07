"""Verify Python package and platform trust stores reject the affected emSign root."""

from __future__ import annotations

import argparse
import hashlib
import importlib.metadata
import os
import re
import stat
import ssl
import sys
from pathlib import Path

import certifi
import pip
from pip._vendor.packaging.version import InvalidVersion, Version

if __package__:
    from .pip_tls_configuration import discover_pip_ca_bundles
    from .python_tls_floor import python_tls_floor_error
else:
    from pip_tls_configuration import discover_pip_ca_bundles
    from python_tls_floor import python_tls_floor_error

try:
    import pip._vendor.certifi as pip_certifi
except ImportError as error:
    pip_certifi = None
    PIP_CERTIFI_IMPORT_ERROR = str(error)
else:
    PIP_CERTIFI_IMPORT_ERROR = ""


COMPROMISED_ROOT = "emSign ECC Root CA - G3"
COMPROMISED_ROOT_SHA256 = (
    "86a1ecba089c4a8d3bbe2734c612ba341d813e043cf9e8a862cd5c57a36bbe6b"
)
REQUIRED_VERSIONS = {
    "pip": "26.2.2",
    "certifi": "2026.10.19",
}
IMPORTED_VERSIONS = {
    "pip": pip.__version__,
    "certifi": certifi.__version__,
}
OPENSSL_HASHED_CERTIFICATE = re.compile(r"^[0-9a-fA-F]{8}\.\d+$")


# {
#   責務: [stableVersion: PEP 440 versionを読み込み、prereleaseなどを拒否する,],
#   引数: [value: package version text,],
#   戻り値: [Version: stable releaseまたはpost release。不正ならValueError,],
# }
def stable_version(value: str) -> Version:
    try:
        version = Version(value)
    except InvalidVersion as error:
        raise ValueError(f"{value!r} is not a valid PEP 440 version") from error
    if version.is_prerelease or version.is_devrelease or version.local is not None:
        raise ValueError(f"{value!r} is not a public stable release")
    return version


# {
#   責務: [defaultCaDirectories: OpenSSL既定のCA directoryから証明書を列挙する,],
#   引数: [なし,],
#   戻り値: [tuple[list[dict], list[bytes], list[str]]: 証明書情報、DER、検査error,],
# }
def default_ca_directory_certificates() -> tuple[list[dict], list[bytes], list[str]]:
    # OpenSSLはhash directoryの証明書を必要時に読むため、各hash fileを直接読みます。
    verify_paths = ssl.get_default_verify_paths()
    environment_name = verify_paths.openssl_capath_env
    uses_environment_directory = bool(
        environment_name and environment_name in os.environ
    )
    directory_value = (
        os.environ[environment_name]
        if uses_environment_directory and environment_name
        else verify_paths.capath or ""
    )
    directories = directory_value.split(os.pathsep) if directory_value else []
    certificates: list[dict] = []
    certificate_der: list[bytes] = []
    errors: list[str] = []
    if uses_environment_directory and not directory_value:
        errors.append("SSL_CERT_DIR is configured but empty.")
    if uses_environment_directory and any(not directory for directory in directories):
        errors.append("SSL_CERT_DIR contains an empty directory entry.")
    for directory in filter(None, directories):
        ca_directory = Path(directory)
        try:
            directory_mode = ca_directory.stat().st_mode
        except FileNotFoundError:
            if uses_environment_directory:
                errors.append(f"Configured CA directory does not exist: {ca_directory}")
            continue
        except OSError as error:
            errors.append(f"Could not access CA directory {ca_directory}: {error}")
            continue
        if not stat.S_ISDIR(directory_mode):
            if uses_environment_directory:
                errors.append(f"Configured CA path is not a directory: {ca_directory}")
            continue
        try:
            candidate_files = tuple(ca_directory.iterdir())
        except OSError as error:
            errors.append(f"Could not list CA directory {ca_directory}: {error}")
            continue
        for ca_file in candidate_files:
            if not OPENSSL_HASHED_CERTIFICATE.fullmatch(ca_file.name):
                continue
            try:
                file_mode = ca_file.stat().st_mode
            except FileNotFoundError as error:
                errors.append(f"CA hash file target is missing {ca_file}: {error}")
                continue
            except OSError as error:
                errors.append(f"Could not inspect CA file {ca_file}: {error}")
                continue
            if not stat.S_ISREG(file_mode):
                errors.append(f"CA hash path is not a regular file: {ca_file}")
                continue
            try:
                context = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
                context.load_verify_locations(cafile=str(ca_file))
                certificates.extend(context.get_ca_certs())
                certificate_der.extend(context.get_ca_certs(binary_form=True))
            except (OSError, ValueError) as error:
                errors.append(f"Could not load CA file {ca_file}: {error}")
    return certificates, certificate_der, errors


# {
#   責務: [trustedCertificates: PEM bundleまたはsystem storeのCA nameとSHA-256を取得する,],
#   引数: [bundle_path: PEM bundleのpath。Noneの場合はPythonのsystem storeを使う,],
#   戻り値: [tuple[set[str], set[str], list[str]]: CA名、fingerprint、検査error,],
# }
def trusted_certificates(
    bundle_path: Path | None,
) -> tuple[set[str], set[str], list[str]]:
    # 明示されたbundleとOS既定storeを同じTLS parserで読み取ります。
    context = ssl.create_default_context(
        cafile=str(bundle_path) if bundle_path else None
    )
    certificates = context.get_ca_certs()
    certificate_der = context.get_ca_certs(binary_form=True)
    errors: list[str] = []
    if bundle_path is None:
        directory_certificates, directory_der, errors = (
            default_ca_directory_certificates()
        )
        certificates.extend(directory_certificates)
        certificate_der.extend(directory_der)
    if not certificates or not certificate_der:
        raise ssl.SSLError("trust store contains no CA certificates")
    names: set[str] = set()
    for certificate in certificates:
        for distinguished_name in certificate.get("subject", ()):
            names.update(
                value
                for attribute, value in distinguished_name
                if attribute == "commonName"
            )
    fingerprints = {
        hashlib.sha256(certificate).hexdigest() for certificate in certificate_der
    }
    return names, fingerprints, errors


# {
#   責務: [_is_windows_reparse_point: Windowsの再解析ポイントを識別する,],
#   引数: [path_status: pathのlstat結果,],
#   戻り値: [bool: Windowsの再解析ポイントならTrue,],
# }
def _is_windows_reparse_point(path_status: os.stat_result) -> bool:
    # Windowsのjunctionも再解析ポイントflagで拒否し、bundle外への走査を防ぎます。
    reparse_flag = getattr(stat, "FILE_ATTRIBUTE_REPARSE_POINT", 0)
    attributes = getattr(path_status, "st_file_attributes", 0)
    return bool(reparse_flag and attributes & reparse_flag)


# {
#   責務: [distributionCertifiBundles: 配布物のcertifi CA束を漏れなく列挙する,],
#   引数: [bundle_root: PyInstaller配布物のroot path,],
#   戻り値: [tuple[list[Path], list[str]]: CA束pathと走査error,],
# }
def distribution_certifi_bundles(
    bundle_root: Path,
) -> tuple[list[Path], list[str]]:
    # pathlib.rglobは権限errorを抑制するため、os.walkで走査失敗を記録します。
    bundles: list[Path] = []
    errors: list[str] = []
    # os.walkは開始pathのリンクをたどるため、走査起点を先に検査します。
    try:
        root_status = bundle_root.lstat()
    except OSError as error:
        return bundles, [f"Could not inspect bundle root {bundle_root}: {error}"]
    if stat.S_ISLNK(root_status.st_mode) or _is_windows_reparse_point(root_status):
        return bundles, [f"Bundle root is a symlink or reparse point: {bundle_root}"]
    if not stat.S_ISDIR(root_status.st_mode):
        return bundles, [f"Bundle root is not a directory: {bundle_root}"]

    scan_errors: list[OSError] = []
    for directory, subdirectories, filenames in os.walk(
        bundle_root,
        onerror=scan_errors.append,
        followlinks=False,
    ):
        directory_path = Path(directory)
        for subdirectory in tuple(subdirectories):
            child_directory = directory_path / subdirectory
            try:
                child_status = child_directory.lstat()
            except OSError as error:
                errors.append(
                    f"Could not inspect bundle directory {child_directory}: {error}"
                )
                subdirectories.remove(subdirectory)
                continue
            if stat.S_ISLNK(child_status.st_mode) or _is_windows_reparse_point(
                child_status
            ):
                errors.append(
                    f"Bundle directory is a symlink or reparse point: {child_directory}"
                )
                subdirectories.remove(subdirectory)

        if "cacert.pem" not in filenames or "certifi" not in directory_path.parts:
            continue
        bundle_path = directory_path / "cacert.pem"
        try:
            bundle_status = bundle_path.lstat()
        except OSError as error:
            errors.append(f"Could not inspect bundled CA file {bundle_path}: {error}")
            continue
        if _is_windows_reparse_point(bundle_status) or not stat.S_ISREG(
            bundle_status.st_mode
        ):
            errors.append(f"Bundled CA path is not a regular file: {bundle_path}")
            continue
        bundles.append(bundle_path)

    errors.extend(
        f"Could not scan PyInstaller bundle directory {error.filename}: {error}"
        for error in scan_errors
    )
    return sorted(bundles), errors


# {
#   責務: [main: pip、certifi、Pythonと配布物のtrust storeから影響rootが除外されたことを確認する,],
#   引数: [arguments: CLI arguments,],
#   戻り値: [int: 全確認成功時は0、失敗時は1,],
# }
def main(arguments: list[str] | None = None) -> int:
    # 配布物検査を指定した場合は、その中のcertifi CA bundleも検証します。
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--bundle-root",
        type=Path,
        help=(
            "PyInstaller one-dir root whose bundled certifi CA bundle must be checked."
        ),
    )
    options = parser.parse_args(arguments)

    errors: list[str] = []
    python_version = tuple(sys.version_info[:3])
    print(f"python={'.'.join(map(str, python_version))}")
    floor_error = python_tls_floor_error(
        sys.version_info,
        sys.version_info.releaselevel,
        sys.implementation.name,
    )
    if floor_error:
        errors.append(floor_error)
    print(f"certifi bundle={certifi.where()}")

    for package, required in REQUIRED_VERSIONS.items():
        try:
            package_version = importlib.metadata.version(package)
        except importlib.metadata.PackageNotFoundError as error:
            errors.append(f"{package} package metadata is missing: {error}.")
            continue
        print(f"{package}={package_version}")
        try:
            installed = stable_version(package_version)
            imported = stable_version(IMPORTED_VERSIONS[package])
        except ValueError as error:
            errors.append(f"{package} version cannot be checked: {error}.")
            continue
        if installed != imported:
            errors.append(
                f"{package} imported version {imported} does not match "
                f"metadata version {installed}."
            )
            continue
        if installed < Version(required):
            errors.append(f"{package} must be >= {required}.")

    if pip_certifi is not None:
        try:
            pip_certifi_version = stable_version(pip_certifi.__version__)
        except (AttributeError, ValueError) as error:
            errors.append(f"pip vendored certifi version cannot be checked: {error}.")
        else:
            print(f"pip vendored certifi={pip_certifi_version}")
            certifi_floor = Version(REQUIRED_VERSIONS["certifi"])
            if pip_certifi_version < certifi_floor:
                errors.append(
                    f"pip vendored certifi must be >= {REQUIRED_VERSIONS['certifi']}."
                )

    bundles: list[tuple[str, Path | None]] = [("certifi", Path(certifi.where()))]
    if pip_certifi is None:
        errors.append(
            f"Could not import pip vendored certifi: {PIP_CERTIFI_IMPORT_ERROR}"
        )
    else:
        bundles.append(("pip vendored certifi", Path(pip_certifi.where())))
    bundles.append(("Python system trust store", None))
    if options.bundle_root is not None:
        if not options.bundle_root.is_dir():
            errors.append(
                f"PyInstaller bundle root is not a directory: {options.bundle_root}."
            )
        else:
            distribution_bundles, scan_errors = distribution_certifi_bundles(
                options.bundle_root
            )
            errors.extend(scan_errors)
            if len(distribution_bundles) != 1:
                errors.append(
                    "Expected one bundled certifi CA file under "
                    f"{options.bundle_root}; found {len(distribution_bundles)}."
                )
            else:
                path = distribution_bundles[0]
                bundles.append((f"Python distribution certifi ({path})", path))

    pip_bundles, pip_configuration_errors = discover_pip_ca_bundles()
    errors.extend(pip_configuration_errors)
    bundles.extend(pip_bundles)

    for name, path in bundles:
        source = str(path) if path is not None else "OS default trust store"
        print(f"{name} source={source}")
        try:
            common_names, fingerprints, inspection_errors = trusted_certificates(path)
            errors.extend(
                f"Could not inspect {name}: {error}" for error in inspection_errors
            )
            affected_root_found = (
                COMPROMISED_ROOT in common_names
                or COMPROMISED_ROOT_SHA256 in fingerprints
            )
            status = (
                "affected root present"
                if affected_root_found
                else "affected root absent"
            )
            print(f"{name}: {len(common_names)} CA names; {status}")
            if affected_root_found:
                errors.append(f"{name} still trusts {COMPROMISED_ROOT}.")
        except (OSError, ssl.SSLError) as error:
            errors.append(f"Could not inspect {name}: {error}")

    if errors:
        for error in errors:
            print(f"[ERROR] {error}", file=sys.stderr)
        return 1

    print("Python versions and CA trust stores meet the emSign security floor.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
