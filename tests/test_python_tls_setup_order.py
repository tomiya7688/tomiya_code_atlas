from pathlib import Path


REPOSITORY_ROOT = Path(__file__).resolve().parents[1]


def test_windows_setup_checks_cpython_before_venv_and_pip_network_access() -> None:
    setup_lines = (
        (REPOSITORY_ROOT / "scripts" / "build" / "setup.bat")
        .read_text(encoding="utf-8")
        .splitlines()
    )
    python_floor_checks = [
        index
        for index, line in enumerate(setup_lines)
        if "tools\\python_tls_floor.py" in line
    ]
    virtual_environment_creation = next(
        index for index, line in enumerate(setup_lines) if "-m venv .venv" in line
    )
    pip_update = next(
        index
        for index, line in enumerate(setup_lines)
        if "-m pip install --upgrade --require-hashes" in line
    )

    assert len(python_floor_checks) == 3
    assert python_floor_checks[0] < virtual_environment_creation
    assert python_floor_checks[1] < virtual_environment_creation
    assert python_floor_checks[2] < pip_update


def test_ci_checks_cpython_before_hashed_pip_downloads() -> None:
    workflow_lines = (
        (REPOSITORY_ROOT / ".github" / "workflows" / "ci.yml")
        .read_text(encoding="utf-8")
        .splitlines()
    )
    floor_check_count = sum(
        "python tools/python_tls_floor.py" in line for line in workflow_lines
    )
    bootstrap_count = sum(
        "python -m pip install --upgrade --require-hashes -r "
        "tools/requirements-pip-bootstrap.txt" in line
        for line in workflow_lines
    )

    assert floor_check_count == bootstrap_count
    for index, line in enumerate(workflow_lines):
        if "python -m pip install --upgrade --require-hashes" not in line:
            continue
        assert any(
            "python tools/python_tls_floor.py" in prior_line
            for prior_line in workflow_lines[max(0, index - 4) : index]
        )


def test_python_backend_build_keeps_temporary_and_pyinstaller_files_in_repo() -> None:
    build_lines = (
        (REPOSITORY_ROOT / "scripts" / "build" / "build_python_backend.bat")
        .read_text(encoding="utf-8")
        .splitlines()
    )
    pyinstaller_invocation = next(
        index for index, line in enumerate(build_lines) if "-m PyInstaller" in line
    )
    required_local_paths = (
        'set "TEMP=%CD%\\.build\\temp"',
        'set "TMP=%TEMP%"',
        'set "PYINSTALLER_CONFIG_DIR=%CD%\\.build\\pyinstaller\\config"',
    )

    for setting in required_local_paths:
        setting_index = next(
            index for index, line in enumerate(build_lines) if line == setting
        )
        assert setting_index < pyinstaller_invocation

    required_directories = (
        ".build\\temp",
        ".build\\pyinstaller\\config",
        ".build\\pyinstaller\\work",
        ".build\\pyinstaller\\spec",
        ".build\\dist\\backends",
        ".build\\dist\\backends\\tomiya-python-backend\\licenses",
    )
    for directory in required_directories:
        create_command = f'if not exist "{directory}" mkdir "{directory}"'
        create_index = next(
            index for index, line in enumerate(build_lines) if line == create_command
        )
        assert build_lines[create_index + 1] == "if errorlevel 1 exit /b 1"
