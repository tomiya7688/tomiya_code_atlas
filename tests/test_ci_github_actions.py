from pathlib import Path

from Src.analyzers.ci import parse_github_actions


def test_github_actions_extracts_jobs_dependencies_and_steps() -> None:
    workflow = parse_github_actions("""
name: Build
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Install
        run: python -m pip install -e .
      - name: Lint
        uses: astral-sh/ruff-action@v3
  build:
    needs: [test]
    steps:
      - name: Package
        run: python -m build
""")

    assert workflow.name == "Build"
    assert workflow.trigger == ("push", "pull_request")
    assert workflow.jobs[0].name == "test"
    assert [step.name for step in workflow.jobs[0].steps] == ["Install", "Lint"]
    assert workflow.jobs[0].steps[0].command == "python -m pip install -e ."
    assert workflow.jobs[0].steps[1].action == "astral-sh/ruff-action@v3"
    assert workflow.jobs[1].needs == ("test",)


def test_github_actions_supports_multiline_triggers_commands_needs_and_unnamed_steps() -> None:
    workflow = parse_github_actions("""
name: Standard YAML
on:
  push:
  pull_request:
jobs:
  verify:
    needs:
      - lint
      - test
    steps:
      - run: |
          python -m pytest
          python -m build
      - uses: actions/checkout@v4
  lint:
    steps: []
  test:
    needs: lint
    steps:
      - name: Tests
        run: >
          python -m pytest
          -q
""")

    assert workflow.trigger == ("push", "pull_request")
    assert [job.name for job in workflow.jobs] == ["verify", "lint", "test"]
    assert workflow.jobs[0].needs == ("lint", "test")
    assert workflow.jobs[0].steps[0].name == "run"
    assert "python -m pytest" in (workflow.jobs[0].steps[0].command or "")
    assert "python -m build" in (workflow.jobs[0].steps[0].command or "")
    assert workflow.jobs[0].steps[1].name == "uses"
    assert workflow.jobs[0].steps[1].action == "actions/checkout@v4"
    assert workflow.jobs[2].needs == ("lint",)
    assert "python -m pytest -q" in (workflow.jobs[2].steps[0].command or "")


def test_github_actions_only_treats_jobs_mapping_as_jobs() -> None:
    workflow = parse_github_actions("""
on:
  push:
  workflow_dispatch:
jobs:
  test:
    steps:
      - run: pytest
""")

    assert workflow.trigger == ("push", "workflow_dispatch")
    assert [job.name for job in workflow.jobs] == ["test"]


def test_tomiya_ci_workflow_parses_as_regression_fixture() -> None:
    source = Path(".github/workflows/ci.yml").read_text(encoding="utf-8")
    workflow = parse_github_actions(source)

    assert workflow.name == "CI"
    assert workflow.trigger == ("push", "pull_request")
    jobs = {job.name: job for job in workflow.jobs}
    assert {"test", "oop-design"} <= set(jobs)
    test_steps = {step.name: step for step in jobs["test"].steps}
    install = test_steps["projectとtest用ツールをinstall"]
    assert "python -m pip install --upgrade pip" in (install.command or "")
    assert 'python -m pip install -e ".[test]"' in (install.command or "")
    assert test_steps["全testを実行"].command == "python -m pytest"


def test_v1_release_workflow_is_withheld_until_go_migration_finishes() -> None:
    assert not Path(".github/workflows/release.yml").exists()
    checklist = Path("docs/jp/リリース確認表.md").read_text(encoding="utf-8")
    assert "Issue #27" in checklist
    assert "Go移行完了まで" in checklist


def test_go_windows_artifact_is_smoke_tested_before_upload() -> None:
    source = Path(".github/workflows/go-exe.yml").read_text(encoding="utf-8")
    workflow = parse_github_actions(source)

    assert workflow.name == "Go EXE"
    assert workflow.trigger == ("pull_request", "push", "workflow_dispatch")
    steps = {step.name: step for step in workflow.jobs[0].steps}
    assert "go test ./..." in (steps["Go testを実行"].command or "")
    assert "call build_exe.bat" in (steps["Windows EXEをbuild"].command or "")
    assert "配布EXEのhelpとversionをsmoke test" in steps
    assert "Go版Windows EXEをartifactとして保存" in steps
    assert '$helpText = $help -join "`n"' in source
    assert "cache-dependency-path: go/go.mod" in source
    assert "tomiya-code-atlas-go-windows-x64" in source
    assert not Path(".github/workflows/build.yml").exists()
    assert not Path(".github/workflows/python-exe.yml").exists()
