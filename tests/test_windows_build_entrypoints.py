from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def read_repo_file(relative_path: str) -> str:
    return (ROOT / relative_path).read_text(encoding="utf-8")


def test_setup_uses_project_local_supported_python_environment() -> None:
    setup = read_repo_file("scripts/build/setup.bat")

    assert "py -3.12" in setup
    assert "sys.version_info >= (3, 11)" in setup
    assert ".venv\\Scripts\\python.exe" in setup
    assert "-m ensurepip --upgrade --default-pip" in setup
    assert setup.index('mkdir ".build\\metadata"') < setup.index('pip install -e ".[test,exe]" build')
    assert 'pip install -e ".[test,exe]" build' in setup


def test_source_and_distribution_launchers_have_distinct_targets() -> None:
    source_launcher = read_repo_file("scripts/build/run_source.bat")
    distribution_launcher = read_repo_file("run_dist.bat")

    assert "Python環境が見つかりません" in source_launcher
    assert "scripts\\build\\setup.bat" in source_launcher
    assert '".venv\\Scripts\\python.exe" app.py %*' in source_launcher
    assert ".build\\dist\\tomiya-code-atlas\\tomiya-code-atlas.exe" in distribution_launcher
    assert '"%APP%" %*' in distribution_launcher


def test_build_commands_use_the_setup_environment_and_documented_outputs() -> None:
    package_build = read_repo_file("scripts/build/package.bat")
    app_build = read_repo_file("build_exe.bat")
    full_verification = read_repo_file("scripts/build/verify.bat")

    assert '".venv\\Scripts\\python.exe" -m build --outdir .build\\packages' in package_build
    assert 'call scripts\\build\\setup.bat' in app_build
    assert app_build.index('call scripts\\build\\setup.bat') < app_build.index('where dotnet')
    assert "Python環境を準備できませんでした" in app_build
    assert '".venv\\Scripts\\python.exe" -m PyInstaller --onedir' in app_build
    assert "dotnet --list-sdks | findstr /b \"10.\"" in app_build
    assert 'findstr /c:"25."' in app_build
    assert "resolve_maven.ps1" in app_build
    assert "for /f %%V in ('%PYTHON% -c \"from Src.version" in full_verification
    assert "call run_dist.bat --version" in full_verification
    assert "call scripts\\build\\run_source.bat --version" in full_verification


def test_only_direct_windows_build_and_distribution_entrypoints_remain_in_root() -> None:
    assert (ROOT / "build_exe.bat").is_file()
    assert (ROOT / "run_dist.bat").is_file()
    for helper in ("setup.bat", "package.bat", "run_source.bat", "verify.bat"):
        assert (ROOT / "scripts" / "build" / helper).is_file()
    for old_root_helper in ("setup.bat", "build.bat", "run.bat", "verify_build.bat"):
        assert not (ROOT / old_root_helper).exists()


def test_windows_build_entrypoints_resolve_paths_from_the_repository_root() -> None:
    for helper in ("setup.bat", "package.bat", "run_source.bat", "verify.bat"):
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


def test_readme_explains_user_build_and_launch_commands() -> None:
    readme = read_repo_file("README.md")

    for required_text in (
        "Python 3.12",
        "scripts\\build\\setup.bat",
        "scripts\\build\\run_source.bat",
        "scripts\\build\\package.bat",
        "build_exe.bat",
        "run_dist.bat",
        "scripts\\build\\verify.bat",
        ".NET SDK 10",
        "JDK 25",
        "Maven",
        "scripts\\dev\\",
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


def test_github_workflow_names_and_visible_diagnostics_are_japanese() -> None:
    workflows = (
        read_repo_file(".github/workflows/build.yml"),
        read_repo_file(".github/workflows/ci.yml"),
        read_repo_file(".github/workflows/python-exe.yml"),
        read_repo_file(".github/workflows/release.yml"),
    )
    assert "- name: Python packageをbuild" in workflows[0]
    assert "- name: 全testを実行" in workflows[1]
    assert "- name: 配布アプリのsmoke test" in workflows[2]
    assert "- name: release候補をbuild・検証" in workflows[3]
    distribution_workflow = workflows[2]
    assert distribution_workflow.startswith("name: Python EXE")
    assert "EXEが見つかりません" in distribution_workflow
    assert "EXE not found" not in distribution_workflow


def test_maven_bootstrap_uses_verified_official_distribution() -> None:
    resolver = read_repo_file("scripts/build/resolve_maven.ps1")

    assert "repo.maven.apache.org/maven2/org/apache/maven/apache-maven/$version" in resolver
    assert "Security.Cryptography.SHA512" in resolver
    assert "SHA512" in resolver
    assert 'Join-Path $toolsRoot "apache-maven-$version"' in resolver
    assert 'Join-Path $mavenHome "bin\\mvn.cmd"' in resolver


def test_windows_exe_workflow_verifies_the_documented_clean_runner_path() -> None:
    workflow = read_repo_file(".github/workflows/python-exe.yml")

    assert 'python-version: "3.12"' in workflow
    assert "call scripts\\build\\setup.bat" in workflow
    assert "call scripts\\build\\verify.bat" in workflow
