from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def read_repo_file(relative_path: str) -> str:
    return (ROOT / relative_path).read_text(encoding="utf-8")


def test_python_environment_is_only_for_source_tests_and_does_not_install_pyinstaller() -> None:
    setup = read_repo_file("scripts/build/setup.bat")

    assert "py -3.12" in setup
    assert "sys.version_info >= (3, 11)" in setup
    assert ".venv\\Scripts\\python.exe" in setup
    assert "-m ensurepip --upgrade --default-pip" in setup
    assert 'pip install -e ".[test]"' in setup
    assert ".[test,exe]" not in setup
    assert "pyinstaller" not in setup.lower()


def test_source_and_distribution_launchers_have_distinct_targets() -> None:
    source_launcher = read_repo_file("scripts/build/run_source.bat")
    distribution_launcher = read_repo_file("run_dist.bat")

    assert "Python環境が見つかりません" in source_launcher
    assert "scripts\\build\\setup.bat" in source_launcher
    assert '".venv\\Scripts\\python.exe" app.py %*' in source_launcher
    assert ".build\\dist\\tomiya-code-atlas.exe" in distribution_launcher
    assert '"%APP%" %*' in distribution_launcher


def test_root_build_creates_only_the_go_distribution_executable() -> None:
    app_build = read_repo_file("build_exe.bat")
    go_build = read_repo_file("scripts/build/build_go.bat")

    assert "call scripts\\build\\build_go.bat" in app_build
    assert 'cd /d "%~dp0\\..\\.."' in go_build
    assert 'set "GOCACHE=%CD%\\.build\\go-cache"' in go_build
    assert 'set "GOTMPDIR=%CD%\\.build\\go-temp"' in go_build
    assert "pushd go" in go_build
    assert "go test ./..." in go_build
    assert "go build -trimpath" in go_build
    assert ".build\\dist\\tomiya-code-atlas.exe" in go_build
    assert "pip install" not in go_build
    assert "PyInstaller" not in go_build


def test_verification_tests_sources_and_go_exe_without_building_python_artifacts() -> None:
    verification = read_repo_file("scripts/build/verify.bat")

    assert "call build_exe.bat" in verification
    assert "call run_dist.bat --version" in verification
    assert "call run_dist.bat --help" in verification
    assert '".venv\\Scripts\\python.exe" -m pytest' in verification
    assert "policy-check" in verification
    assert "build_python" not in verification
    assert "PyInstaller" not in verification
    assert "package.bat" not in verification


def test_only_direct_windows_build_and_distribution_entrypoints_remain_in_root() -> None:
    assert (ROOT / "build_exe.bat").is_file()
    assert (ROOT / "run_dist.bat").is_file()
    for helper in ("setup.bat", "run_source.bat", "verify.bat", "build_go.bat"):
        assert (ROOT / "scripts" / "build" / helper).is_file()
    for removed_helper in ("package.bat", "build_python_legacy.bat"):
        assert not (ROOT / "scripts" / "build" / removed_helper).exists()
    for old_root_helper in ("setup.bat", "build.bat", "run.bat", "verify_build.bat"):
        assert not (ROOT / old_root_helper).exists()


def test_windows_build_helpers_resolve_paths_from_the_repository_root() -> None:
    for helper in ("setup.bat", "run_source.bat", "verify.bat", "build_go.bat"):
        source = read_repo_file(f"scripts/build/{helper}")
        assert 'cd /d "%~dp0\\..\\.."' in source
    for entrypoint in ("build_exe.bat", "run_dist.bat"):
        assert 'cd /d "%~dp0"' in read_repo_file(entrypoint)


def test_development_helpers_are_grouped_and_run_from_repository_root() -> None:
    helper_directory = ROOT / "scripts" / "dev"
    assert helper_directory.is_dir()
    assert not (ROOT / "prepare_work.bat").exists()
    assert not (ROOT / "context.bat").exists()
    assert 'cd /d "%~dp0\\..\\.."' in read_repo_file("scripts/dev/context.bat")
    assert 'cd "$(dirname "$0")/../.."' in read_repo_file("scripts/dev/context.sh")
    assert "scripts\\dev\\reducer.bat setup" in read_repo_file("scripts/dev/prepare_work.bat")


def test_readme_explains_go_build_and_python_test_only_usage() -> None:
    readme = read_repo_file("README.md")

    for required_text in (
        "Go 1.22以降",
        "build_exe.bat",
        "run_dist.bat",
        "scripts\\build\\setup.bat",
        "scripts\\build\\run_source.bat",
        "scripts\\build\\verify.bat",
        "Python/PyInstallerのEXE buildは行いません",
        "Go EXE",
    ):
        assert required_text in readme


def test_windows_user_entrypoint_errors_are_japanese() -> None:
    launcher = read_repo_file("run_dist.bat")
    verifier = read_repo_file("scripts/build/verify.bat")
    source_launcher = read_repo_file("scripts/build/run_source.bat")

    assert "[エラー] ビルド済みアプリが見つかりません" in launcher
    assert "先に build_exe.bat を実行" in launcher
    assert "[エラー]" in verifier
    assert "did not launch" not in verifier
    assert "verification failed" not in verifier
    assert "[エラー] Python環境が見つかりません" in source_launcher
    assert "Project environment not found" not in source_launcher


def test_only_go_distribution_workflow_builds_and_uploads_windows_exe() -> None:
    ci_workflow = read_repo_file(".github/workflows/ci.yml")
    assert "- name: 全testを実行" in ci_workflow
    go_workflow = read_repo_file(".github/workflows/go-exe.yml")
    assert go_workflow.startswith("name: Go EXE")
    assert 'go-version: "1.27.x"' in go_workflow
    assert '"go/**"' in go_workflow
    assert "go test ./..." in go_workflow
    assert "call build_exe.bat" in go_workflow
    assert "--version" in go_workflow
    assert "--help" in go_workflow
    assert "tomiya-code-atlas-go-windows-x64" in go_workflow
    assert "Go版Windows EXEをartifactとして保存" in go_workflow
    assert not (ROOT / ".github/workflows/python-exe.yml").exists()
    assert not (ROOT / ".github/workflows/build.yml").exists()
    assert not (ROOT / ".github/workflows/release.yml").exists()
