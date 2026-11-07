import os
import ntpath
from pathlib import Path, PureWindowsPath

from tools.pip_tls_configuration import (
    discover_pip_ca_bundles,
    pip_config_files,
)


def _windows_paths(paths: list[Path]) -> set[PureWindowsPath]:
    return {PureWindowsPath(str(path)) for path in paths}


def _expected_windows_pip_config(home: str) -> PureWindowsPath:
    return PureWindowsPath(ntpath.join(home, "pip", "pip.ini"))


def test_windows_pip_config_uses_userprofile_when_home_differs(
    tmp_path: Path,
) -> None:
    user_profile = tmp_path / "user-profile"
    environment = {
        "HOME": str(tmp_path / "home"),
        "USERPROFILE": str(user_profile),
        "HOMEDRIVE": "Z:",
        "HOMEPATH": "\\different-profile",
    }

    paths = pip_config_files(environment, tmp_path / "prefix", "win32")

    windows_paths = _windows_paths(paths)
    assert _expected_windows_pip_config(str(user_profile)) in windows_paths
    assert _expected_windows_pip_config(environment["HOME"]) not in windows_paths
    assert (
        _expected_windows_pip_config(ntpath.join("Z:", "\\different-profile"))
        not in windows_paths
    )


def test_windows_pip_config_uses_home_drive_and_path_without_userprofile(
    tmp_path: Path,
) -> None:
    environment = {
        "HOME": str(tmp_path / "home"),
        "HOMEDRIVE": "Z:",
        "HOMEPATH": "\\user-profile",
    }

    paths = pip_config_files(environment, tmp_path / "prefix", "win32")

    home = ntpath.join("Z:", "\\user-profile")
    windows_paths = _windows_paths(paths)
    assert _expected_windows_pip_config(home) in windows_paths
    assert _expected_windows_pip_config(environment["HOME"]) not in windows_paths


def test_windows_pip_config_matches_hompath_without_home_drive() -> None:
    environment = {"HOMEPATH": "\\user-profile"}

    paths = pip_config_files(environment, Path("prefix"), "win32")

    home = ntpath.join("", "\\user-profile")
    assert _expected_windows_pip_config(home) in _windows_paths(paths)


def test_windows_pip_config_keeps_tilde_when_home_variables_are_missing() -> None:
    paths = pip_config_files({}, Path("prefix"), "win32")

    assert _expected_windows_pip_config("~") in _windows_paths(paths)


def test_windows_pip_config_preserves_empty_userprofile_like_pip() -> None:
    environment = {
        "USERPROFILE": "",
        "HOMEDRIVE": "Z:",
        "HOMEPATH": "\\user-profile",
    }

    paths = pip_config_files(environment, Path("prefix"), "win32")

    assert _expected_windows_pip_config("") in _windows_paths(paths)


def test_discover_pip_ca_bundles_reads_certificate_environment_variables(
    tmp_path: Path,
) -> None:
    environment = {
        "HOME": str(tmp_path / "home"),
        "PROGRAMDATA": str(tmp_path / "program-data"),
        "APPDATA": str(tmp_path / "app-data"),
        "PIP_CERT": "custom/pip.pem",
        "REQUESTS_CA_BUNDLE": "custom/requests.pem",
        "CURL_CA_BUNDLE": "custom/curl.pem",
        "PIP_CONFIG_FILE": os.devnull,
    }

    bundles, errors = discover_pip_ca_bundles(
        environment=environment,
        prefix=tmp_path / "venv",
        platform_name="win32",
    )

    assert bundles == [
        ("PIP_CERT", Path("custom/pip.pem")),
        ("REQUESTS_CA_BUNDLE", Path("custom/requests.pem")),
        ("CURL_CA_BUNDLE", Path("custom/curl.pem")),
    ]
    assert errors == []


def test_discover_pip_ca_bundles_reads_configured_cert_files(tmp_path: Path) -> None:
    config_file = tmp_path / "pip.ini"
    config_file.write_text(
        "[DEFAULT]\ncert = default.pem\n\n"
        "[global]\ncert = global.pem\n\n[install]\ncert = install.pem\n",
        encoding="utf-8",
    )
    environment = {
        "HOME": str(tmp_path / "home"),
        "PROGRAMDATA": str(tmp_path / "program-data"),
        "APPDATA": str(tmp_path / "app-data"),
        "PIP_CONFIG_FILE": str(config_file),
    }

    bundles, errors = discover_pip_ca_bundles(
        environment=environment,
        prefix=tmp_path / "venv",
        platform_name="win32",
    )

    assert bundles == [
        (f"{config_file} [DEFAULT].cert", Path("default.pem")),
        (f"{config_file} [global].cert", Path("global.pem")),
        (f"{config_file} [install].cert", Path("install.pem")),
    ]
    assert errors == []


def test_discover_pip_ca_bundles_uses_pip_config_file_precedence(
    tmp_path: Path,
) -> None:
    program_data = tmp_path / "program-data"
    app_data = tmp_path / "app-data"
    prefix = tmp_path / "prefix"
    global_config = program_data / "pip" / "pip.ini"
    user_config = app_data / "pip" / "pip.ini"
    custom_config = tmp_path / "custom.ini"
    global_config.parent.mkdir(parents=True)
    user_config.parent.mkdir(parents=True)
    global_config.write_text("[global]\ncert = global.pem\n", encoding="utf-8")
    user_config.write_text("[global]\ncert = user.pem\n", encoding="utf-8")
    custom_config.write_text("[global]\ncert = custom.pem\n", encoding="utf-8")
    environment = {
        "HOME": str(tmp_path / "home"),
        "PROGRAMDATA": str(program_data),
        "APPDATA": str(app_data),
        "PIP_CONFIG_FILE": str(custom_config),
    }

    bundles, errors = discover_pip_ca_bundles(
        environment=environment,
        prefix=prefix,
        platform_name="win32",
    )

    assert bundles == [
        (f"{global_config} [global].cert", Path("global.pem")),
        (f"{custom_config} [global].cert", Path("custom.pem")),
    ]
    assert errors == []


def test_discover_pip_ca_bundles_reports_unreadable_config(
    monkeypatch,
    tmp_path: Path,
) -> None:
    config_file = tmp_path / "pip.ini"
    environment = {
        "HOME": str(tmp_path / "home"),
        "PROGRAMDATA": str(tmp_path / "program-data"),
        "APPDATA": str(tmp_path / "app-data"),
        "PIP_CONFIG_FILE": str(config_file),
    }
    original_open = Path.open

    def reject_custom_config(path: Path, *args, **kwargs):
        if path == config_file:
            raise PermissionError(13, "permission denied", str(path))
        return original_open(path, *args, **kwargs)

    monkeypatch.setattr(Path, "open", reject_custom_config)

    bundles, errors = discover_pip_ca_bundles(
        environment=environment,
        prefix=tmp_path / "venv",
        platform_name="win32",
    )

    assert bundles == []
    assert len(errors) == 1
    assert "permission denied" in errors[0]


def test_pip_config_files_follow_xdg_prefix_and_custom_file_precedence(
    tmp_path: Path,
) -> None:
    xdg_global = tmp_path / "xdg-global"
    xdg_user = tmp_path / "xdg-user"
    python_prefix = tmp_path / "python-prefix"
    unrelated_venv = tmp_path / "unrelated-venv"
    custom_config = tmp_path / "custom.conf"
    custom_config.touch()
    environment = {
        "HOME": str(tmp_path / "home"),
        "XDG_CONFIG_DIRS": str(xdg_global),
        "XDG_CONFIG_HOME": str(xdg_user),
        "VIRTUAL_ENV": str(unrelated_venv),
        "PIP_CONFIG_FILE": str(custom_config),
    }

    paths = pip_config_files(environment, python_prefix, "linux")

    assert xdg_global / "pip" / "pip.conf" in paths
    assert xdg_user / "pip" / "pip.conf" not in paths
    assert python_prefix / "pip.conf" in paths
    assert unrelated_venv / "pip.conf" not in paths
    assert custom_config in paths


def test_pip_config_files_keep_user_config_when_custom_file_is_missing(
    tmp_path: Path,
) -> None:
    home = tmp_path / "home"
    custom_config = tmp_path / "missing-custom.conf"
    environment = {
        "HOME": str(home),
        "XDG_CONFIG_HOME": str(tmp_path / "xdg-user"),
        "PIP_CONFIG_FILE": str(custom_config),
    }

    paths = pip_config_files(environment, tmp_path / "prefix", "linux")

    assert tmp_path / "xdg-user" / "pip" / "pip.conf" in paths
    assert home / ".pip" / "pip.conf" in paths
    assert custom_config in paths


def test_pip_config_files_include_macos_global_and_user_paths(
    tmp_path: Path,
) -> None:
    home = tmp_path / "home"
    xdg_global = tmp_path / "xdg-global"
    macos_user = home / "Library" / "Application Support" / "pip"
    macos_user.mkdir(parents=True)
    environment = {
        "HOME": str(home),
        "XDG_DATA_DIRS": str(xdg_global),
    }

    paths = pip_config_files(environment, tmp_path / "prefix", "darwin")

    assert xdg_global / "pip" / "pip.conf" in paths
    assert macos_user / "pip.conf" in paths
    assert home / ".pip" / "pip.conf" in paths
    assert home / ".config" / "pip" / "pip.conf" not in paths


def test_pip_config_files_use_macos_default_global_directory(
    tmp_path: Path,
) -> None:
    environment = {"HOME": str(tmp_path / "home")}

    paths = pip_config_files(environment, tmp_path / "prefix", "darwin")

    assert Path("/Library/Application Support/pip/pip.conf") in paths


def test_pip_config_files_include_existing_macos_xdg_user_directory(
    tmp_path: Path,
) -> None:
    xdg_data_home = tmp_path / "xdg-data"
    xdg_pip_directory = xdg_data_home / "pip"
    xdg_pip_directory.mkdir(parents=True)
    environment = {
        "HOME": str(tmp_path / "home"),
        "XDG_DATA_HOME": str(xdg_data_home),
    }

    paths = pip_config_files(environment, tmp_path / "prefix", "darwin")

    assert xdg_pip_directory / "pip.conf" in paths
    assert tmp_path / "home" / ".config" / "pip" / "pip.conf" not in paths


def test_discover_pip_ca_bundles_rejects_empty_config_override(
    tmp_path: Path,
) -> None:
    environment = {
        "HOME": str(tmp_path / "home"),
        "PROGRAMDATA": str(tmp_path / "program-data"),
        "APPDATA": str(tmp_path / "app-data"),
        "PIP_CONFIG_FILE": "",
    }

    bundles, errors = discover_pip_ca_bundles(
        environment=environment,
        prefix=tmp_path / "venv",
        platform_name="win32",
    )

    assert bundles == []
    assert errors == ["PIP_CONFIG_FILE is configured but empty."]
