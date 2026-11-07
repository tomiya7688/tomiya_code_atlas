import hashlib
import stat
from collections.abc import Callable
from pathlib import Path
from types import SimpleNamespace

from tools import verify_python_tls


def test_trusted_certificates_combines_system_store_and_ssl_cert_directory(
    monkeypatch,
) -> None:
    system_certificate = {
        "subject": ((("commonName", "System Root"),),),
    }
    capath_certificate = {
        "subject": ((("commonName", "OpenSSL Directory Root"),),),
    }

    class CertificateContext:
        def get_ca_certs(self, binary_form: bool = False):
            if binary_form:
                return [b"system der"]
            return [system_certificate]

    monkeypatch.setattr(
        verify_python_tls.ssl,
        "create_default_context",
        lambda cafile=None: CertificateContext(),
    )
    monkeypatch.setattr(
        verify_python_tls,
        "default_ca_directory_certificates",
        lambda: ([capath_certificate], [b"directory der"], []),
    )

    names, fingerprints, errors = verify_python_tls.trusted_certificates(None)

    assert names == {"System Root", "OpenSSL Directory Root"}
    assert fingerprints == {
        hashlib.sha256(b"system der").hexdigest(),
        hashlib.sha256(b"directory der").hexdigest(),
    }
    assert errors == []


def test_distribution_certifi_bundles_finds_ca_file(tmp_path: Path) -> None:
    bundle_root = tmp_path / "distribution"
    certifi_directory = bundle_root / "_internal" / "certifi"
    certifi_directory.mkdir(parents=True)
    ca_bundle = certifi_directory / "cacert.pem"
    ca_bundle.write_text("test data", encoding="utf-8")

    bundles, errors = verify_python_tls.distribution_certifi_bundles(bundle_root)

    assert bundles == [ca_bundle]
    assert errors == []


def test_distribution_certifi_bundles_reports_inaccessible_subdirectory(
    monkeypatch,
    tmp_path: Path,
) -> None:
    bundle_root = tmp_path / "distribution"
    certifi_directory = bundle_root / "certifi"
    certifi_directory.mkdir(parents=True)
    ca_bundle = certifi_directory / "cacert.pem"
    ca_bundle.write_text("test data", encoding="utf-8")

    def walk_with_permission_error(
        root: Path,
        onerror: Callable[[OSError], None],
        followlinks: bool,
    ):
        assert followlinks is False
        onerror(PermissionError(13, "permission denied", str(root / "blocked")))
        yield str(root), ["certifi"], []
        yield str(certifi_directory), [], ["cacert.pem"]

    monkeypatch.setattr(verify_python_tls.os, "walk", walk_with_permission_error)

    bundles, errors = verify_python_tls.distribution_certifi_bundles(bundle_root)

    assert bundles == [ca_bundle]
    assert len(errors) == 1
    assert "permission denied" in errors[0]


def test_distribution_certifi_bundles_rejects_symlink_directory(
    monkeypatch,
    tmp_path: Path,
) -> None:
    bundle_root = tmp_path / "distribution"
    certifi_directory = bundle_root / "certifi"
    certifi_directory.mkdir(parents=True)
    (certifi_directory / "cacert.pem").write_text("test data", encoding="utf-8")
    original_lstat = Path.lstat

    def report_symlink(path: Path):
        if path == certifi_directory:
            return SimpleNamespace(st_mode=stat.S_IFLNK)
        return original_lstat(path)

    monkeypatch.setattr(Path, "lstat", report_symlink)

    bundles, errors = verify_python_tls.distribution_certifi_bundles(bundle_root)

    assert bundles == []
    assert len(errors) == 1
    assert "symlink" in errors[0]


def test_distribution_certifi_bundles_rejects_symlink_ca_file(
    monkeypatch,
    tmp_path: Path,
) -> None:
    bundle_root = tmp_path / "distribution"
    certifi_directory = bundle_root / "certifi"
    certifi_directory.mkdir(parents=True)
    ca_bundle = certifi_directory / "cacert.pem"
    ca_bundle.write_text("test data", encoding="utf-8")
    original_lstat = Path.lstat

    def report_symlink(path: Path):
        if path == ca_bundle:
            return SimpleNamespace(st_mode=stat.S_IFLNK)
        return original_lstat(path)

    monkeypatch.setattr(Path, "lstat", report_symlink)

    bundles, errors = verify_python_tls.distribution_certifi_bundles(bundle_root)

    assert bundles == []
    assert len(errors) == 1
    assert "regular file" in errors[0]
