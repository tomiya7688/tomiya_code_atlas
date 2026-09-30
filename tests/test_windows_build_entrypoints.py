from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def read_repo_file(relative_path: str) -> str:
    return (ROOT / relative_path).read_text(encoding="utf-8")


def test_setup_uses_project_local_python_312_environment() -> None:
    setup = read_repo_file("setup.bat")

    assert "py -3.12" in setup
    assert ".venv\\Scripts\\python.exe" in setup
    assert 'pip install -e ".[test,exe]" build' in setup


def test_source_and_distribution_launchers_have_distinct_targets() -> None:
    source_launcher = read_repo_file("run.bat")
    distribution_launcher = read_repo_file("run_dist.bat")

    assert "Project environment not found. Run setup.bat first." in source_launcher
    assert '".venv\\Scripts\\python.exe" app.py %*' in source_launcher
    assert ".build\\dist\\tomiya-code-atlas\\tomiya-code-atlas.exe" in distribution_launcher
    assert '"%APP%" %*' in distribution_launcher


def test_build_commands_use_the_setup_environment_and_documented_outputs() -> None:
    package_build = read_repo_file("build.bat")
    app_build = read_repo_file("build_exe.bat")
    full_verification = read_repo_file("verify_build.bat")

    assert '".venv\\Scripts\\python.exe" -m build --outdir .build\\packages' in package_build
    assert '".venv\\Scripts\\python.exe" -m PyInstaller --onedir' in app_build
    assert "dotnet --list-sdks | findstr /b \"10.\"" in app_build
    assert 'findstr /c:"25."' in app_build
    assert "call run_dist.bat --version" in full_verification
    assert "call run.bat --version" in full_verification


def test_development_helpers_are_grouped_and_run_from_repository_root() -> None:
    helper_directory = ROOT / "scripts" / "dev"
    assert helper_directory.is_dir()
    assert not (ROOT / "prepare_work.bat").exists()
    assert not (ROOT / "context.bat").exists()
    assert 'cd /d "%~dp0\\..\\.."' in read_repo_file("scripts/dev/context.bat")
    assert 'cd "$(dirname "$0")/../.."' in read_repo_file("scripts/dev/context.sh")
    assert "scripts\\dev\\reducer.bat setup" in read_repo_file("scripts/dev/prepare_work.bat")


def test_readme_explains_user_build_and_launch_commands() -> None:
    readme = read_repo_file("README.md")

    for required_text in (
        "Python 3.12",
        "setup.bat",
        "run.bat",
        "build.bat",
        "build_exe.bat",
        "run_dist.bat",
        "verify_build.bat",
        ".NET SDK 10",
        "JDK 25",
        "Maven",
        "scripts\\dev\\",
    ):
        assert required_text in readme


def test_windows_exe_workflow_verifies_the_documented_clean_runner_path() -> None:
    workflow = read_repo_file(".github/workflows/python-exe.yml")

    assert 'python-version: "3.12"' in workflow
    assert "call setup.bat" in workflow
    assert "call verify_build.bat" in workflow
