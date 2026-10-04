from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def read_repo_file(relative_path: str) -> str:
    return (ROOT / relative_path).read_text(encoding="utf-8")


def test_python_environment_installs_the_parser_helper_build_dependency() -> None:
    setup = read_repo_file("scripts/build/setup.bat")

    assert "py -3.12" in setup
    assert "sys.version_info >= (3, 11)" in setup
    assert ".venv\\Scripts\\python.exe" in setup
    assert "-m ensurepip --upgrade --default-pip" in setup
    assert 'pip install -e ".[test,parser-build,cpp-parser]"' in setup
    assert "PyInstaller" in read_repo_file("pyproject.toml")


def test_source_and_distribution_launchers_have_distinct_targets() -> None:
    source_launcher = read_repo_file("scripts/build/run_source.bat")
    distribution_launcher = read_repo_file("run_dist.bat")

    assert "Python環境が見つかりません" in source_launcher
    assert "scripts\\build\\setup.bat" in source_launcher
    assert '".venv\\Scripts\\python.exe" app.py %*' in source_launcher
    assert ".build\\dist\\tomiya-code-atlas.exe" in distribution_launcher
    assert '"%APP%" %*' in distribution_launcher


def test_root_build_creates_go_cli_and_cpython_ast_onedir_helper() -> None:
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
    assert "build_python_backend.bat" in go_build
    assert "build_csharp_backend.bat" in go_build
    assert "build_cpp_backend.bat" in go_build
    python_build = read_repo_file("scripts/build/build_python_backend.bat")
    assert "--onedir" in python_build
    assert "--onefile" not in python_build
    assert ".build\\dist\\backends\\tomiya-python-backend\\tomiya-python-backend.exe" in python_build
    assert "PyInstaller" in python_build
    assert "copy_python_runtime_license.py" in python_build
    csharp_build = read_repo_file("scripts/build/build_csharp_backend.bat")
    assert "dotnet publish" in csharp_build
    assert "--self-contained true" in csharp_build
    assert "-r win-x64" in csharp_build
    assert "PublishSingleFile=false" in csharp_build
    assert "PublishTrimmed=false" in csharp_build
    assert "dotnet --list-sdks" in csharp_build
    assert "dotnet-ThirdPartyNotices.txt" in csharp_build


def test_verification_smokes_the_frozen_python_parser_helper() -> None:
    verification = read_repo_file("scripts/build/verify.bat")

    assert "call build_exe.bat" in verification
    assert "call run_dist.bat --version" in verification
    assert "call run_dist.bat --help" in verification
    assert 'findstr /c:"--help, -h"' in verification
    assert '".venv\\Scripts\\python.exe" -m pytest' in verification
    assert "-p no:cacheprovider" in verification
    assert '--ignore-glob="pytest-cache-files-*"' in verification
    assert "policy-check" in verification
    assert "smoke_python_backend.py" in verification
    assert "csharp_backend_smoke.py" in verification
    assert "cpp_backend_smoke.py" in verification
    assert "package.bat" not in verification


def test_only_direct_windows_build_and_distribution_entrypoints_remain_in_root() -> None:
    assert (ROOT / "build_exe.bat").is_file()
    assert (ROOT / "run_dist.bat").is_file()
    for helper in ("setup.bat", "run_source.bat", "verify.bat", "build_go.bat", "build_python_backend.bat", "build_csharp_backend.bat", "build_cpp_backend.bat"):
        assert (ROOT / "scripts" / "build" / helper).is_file()
    for removed_helper in ("package.bat", "build_python_legacy.bat"):
        assert not (ROOT / "scripts" / "build" / removed_helper).exists()
    for old_root_helper in ("setup.bat", "build.bat", "run.bat", "verify_build.bat"):
        assert not (ROOT / old_root_helper).exists()


def test_windows_build_helpers_resolve_paths_from_the_repository_root() -> None:
    for helper in ("setup.bat", "run_source.bat", "verify.bat", "build_go.bat", "build_python_backend.bat", "build_csharp_backend.bat", "build_cpp_backend.bat"):
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


def test_pull_request_runner_skips_local_pytest_cache_artifacts() -> None:
    create_pr = read_repo_file("tools/create_pr.py")

    assert '"no:cacheprovider"' in create_pr
    assert '"--ignore-glob=pytest-cache-files-*"' in create_pr


def test_readme_explains_go_and_python_parser_build_requirements() -> None:
    readme = read_repo_file("README.md")

    for required_text in (
        "Go 1.22以降",
        "build_exe.bat",
        "run_dist.bat",
        "scripts\\build\\setup.bat",
        "scripts\\build\\run_source.bat",
        "scripts\\build\\verify.bat",
        "CPython AST",
        "PyInstaller",
        ".NET 10 SDK",
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


def test_go_distribution_workflow_builds_and_uploads_windows_artifacts() -> None:
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
    assert "tomiya-python-backend\\tomiya-python-backend.exe" in go_workflow
    assert "copy_python_runtime_license.py" in go_workflow or "build_python_backend.bat" in go_workflow
    assert "smoke_python_backend.py" in go_workflow
    assert "build_csharp_backend.bat" in go_workflow
    assert "csharp_backend_smoke.py" in go_workflow
    assert ".build/dist/backends/csharp/" in go_workflow
    assert "build_cpp_backend.bat" in go_workflow
    assert "cpp_backend_smoke.py" in go_workflow
    assert "cpp\\tomiya-cpp-backend\\tomiya-cpp-backend.exe" in go_workflow
    assert ".build/dist/backends/cpp/" in go_workflow
    assert "Go版Windows EXEをartifactとして保存" in go_workflow
    assert not (ROOT / ".github/workflows/python-exe.yml").exists()
    assert not (ROOT / ".github/workflows/build.yml").exists()
    assert not (ROOT / ".github/workflows/release.yml").exists()
