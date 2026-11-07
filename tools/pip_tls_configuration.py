"""Find CA bundles that pip configuration overrides select."""

from __future__ import annotations

import configparser
import os
import sys
from collections.abc import Mapping
from pathlib import Path


PIP_CERT_ENVIRONMENT_VARIABLES = (
    "PIP_CERT",
    "REQUESTS_CA_BUNDLE",
    "CURL_CA_BUNDLE",
)


# {
#   責務: [
#     pip_config_files: pipの優先規則に沿って設定ファイルpathを列挙する,
#   ],
#   引数: [
#     environment: 設定位置を解決する環境変数,
#     prefix: pipが使用するPython環境のprefix,
#     platform_name: 設定pathを選択するOS名,
#   ],
#   戻り値: [
#     paths: 重複を除いた設定ファイルpath,
#   ],
# }
def pip_config_files(
    environment: Mapping[str, str],
    prefix: Path,
    platform_name: str,
) -> list[Path]:
    # Windows版pipはUSERPROFILE、またはHOMEDRIVEとHOMEPATHからホームを決めます。
    if platform_name == "win32":
        if "USERPROFILE" in environment:
            home = Path(environment["USERPROFILE"])
        elif "HOMEPATH" in environment:
            home = Path(environment.get("HOMEDRIVE", "") + environment["HOMEPATH"])
        else:
            home = Path("~")
    else:
        home = Path(environment.get("HOME", str(Path.home())))
    paths: list[Path] = []
    user_paths: list[Path] = []
    config_name = "pip.ini" if platform_name == "win32" else "pip.conf"
    custom_config = environment.get("PIP_CONFIG_FILE")
    if custom_config is not None:
        if not custom_config.strip():
            return []
        custom_path = Path(custom_config)
        if os.path.normcase(os.path.abspath(custom_config)) == os.path.normcase(
            os.path.abspath(os.devnull)
        ):
            return []
        # pip omits user config only when the custom config file exists.
        load_user_config = not custom_path.exists()
    else:
        custom_path = None
        load_user_config = True

    if platform_name == "win32":
        program_data = environment.get("PROGRAMDATA", r"C:\ProgramData")
        app_data = environment.get("APPDATA")
        paths.append(Path(program_data) / "pip" / "pip.ini")
        if app_data:
            user_paths.append(Path(app_data) / "pip" / "pip.ini")
        user_paths.append(home / "pip" / "pip.ini")
    elif platform_name == "darwin":
        data_directories = environment.get("XDG_DATA_DIRS", "")
        if data_directories:
            paths.extend(
                Path(directory) / "pip" / "pip.conf"
                for directory in data_directories.split(os.pathsep)
                if directory
            )
        else:
            paths.append(Path("/Library/Application Support/pip/pip.conf"))

        xdg_data_home = environment.get("XDG_DATA_HOME", "")
        xdg_user_config = Path(xdg_data_home) / "pip" if xdg_data_home else None
        macos_user_config = home / "Library" / "Application Support" / "pip"
        if xdg_user_config and xdg_user_config.is_dir():
            user_paths.append(xdg_user_config / "pip.conf")
        elif not xdg_data_home and macos_user_config.is_dir():
            user_paths.append(macos_user_config / "pip.conf")
        else:
            config_home = environment.get("XDG_CONFIG_HOME")
            user_paths.append(
                (Path(config_home) if config_home else home / ".config")
                / "pip"
                / "pip.conf"
            )
        user_paths.append(home / ".pip" / "pip.conf")
    else:
        config_directories = environment.get("XDG_CONFIG_DIRS", "/etc/xdg")
        paths.extend(
            Path(directory) / "pip" / "pip.conf"
            for directory in config_directories.split(os.pathsep)
            if directory
        )
        paths.append(Path("/etc/pip.conf"))
        config_home = environment.get("XDG_CONFIG_HOME")
        user_paths.append(
            (Path(config_home) if config_home else home / ".config")
            / "pip"
            / "pip.conf"
        )
        user_paths.append(home / ".pip" / "pip.conf")

    if load_user_config:
        paths.extend(user_paths)
    paths.append(prefix / config_name)

    if custom_path is not None:
        paths.append(custom_path)

    return list(dict.fromkeys(paths))


# {
#   責務: [
#     discover_pip_ca_bundles: pipの環境変数と設定ファイルが指定するCA束を検出する,
#   ],
#   引数: [
#     environment: 検査対象の環境変数,
#     prefix: pipが使用するPython環境のprefix,
#     platform_name: 設定pathを選択するOS名,
#   ],
#   戻り値: [
#     bundles: 設定元とCA束path,
#     errors: CA設定を読み取れない場合のエラー,
#   ],
# }
def discover_pip_ca_bundles(
    environment: Mapping[str, str] | None = None,
    prefix: Path | None = None,
    platform_name: str | None = None,
) -> tuple[list[tuple[str, Path]], list[str]]:
    current_environment = os.environ if environment is None else environment
    current_prefix = Path(sys.prefix) if prefix is None else prefix
    current_platform = sys.platform if platform_name is None else platform_name
    bundles: list[tuple[str, Path]] = []
    errors: list[str] = []
    if (
        "PIP_CONFIG_FILE" in current_environment
        and not current_environment["PIP_CONFIG_FILE"].strip()
    ):
        errors.append("PIP_CONFIG_FILE is configured but empty.")

    # pipのHTTPS設定環境変数はcertifiのCA束を上書きします。
    for variable in PIP_CERT_ENVIRONMENT_VARIABLES:
        if variable not in current_environment:
            continue
        value = current_environment[variable].strip()
        if not value:
            errors.append(f"{variable} is configured but empty.")
            continue
        bundles.append((variable, Path(value)))

    for config_path in pip_config_files(
        current_environment, current_prefix, current_platform
    ):
        try:
            with config_path.open(encoding="utf-8-sig") as config_file:
                parser = configparser.ConfigParser(interpolation=None)
                parser.read_file(config_file)
        except FileNotFoundError:
            continue
        except (OSError, configparser.Error) as error:
            errors.append(f"Could not inspect pip configuration {config_path}: {error}")
            continue

        default_certificate = parser.defaults().get("cert", "").strip()
        if default_certificate:
            bundles.append((f"{config_path} [DEFAULT].cert", Path(default_certificate)))
        for section in parser.sections():
            value = parser.get(section, "cert", fallback="").strip()
            if not value:
                continue
            source = f"{config_path} [{section}].cert"
            bundles.append((source, Path(value)))

    return bundles, errors
