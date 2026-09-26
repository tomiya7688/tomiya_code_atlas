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
    install = test_steps["Install project and test tooling"]
    assert "python -m pip install --upgrade pip" in (install.command or "")
    assert 'python -m pip install -e ".[test]"' in (install.command or "")
    assert test_steps["Run full test suite"].command == "python -m pytest"


def test_release_workflow_gates_publishing_on_the_verified_tag_candidate() -> None:
    path = Path(".github/workflows/release.yml")
    source = path.read_text(encoding="utf-8")
    workflow = parse_github_actions(source)

    assert workflow.name == "Release"
    assert workflow.trigger == ("push",)
    assert '"v*"' in source
    assert "call verify_build.bat" in source
    assert "gh release create" in source
    assert "--verify-tag" in source


def test_python_artifact_workflows_run_e2e_before_uploading_exact_outputs() -> None:
    build = parse_github_actions(Path(".github/workflows/build.yml").read_text(encoding="utf-8"))
    build_source = Path(".github/workflows/build.yml").read_text(encoding="utf-8")
    exe_source = Path(".github/workflows/python-exe.yml").read_text(encoding="utf-8")

    assert build.name == "Build"
    assert "tools/verify_wheel.py dist/*.whl" in build_source
    assert "tools/verify_wheel.py dist/*.tar.gz" in build_source
    assert "Upload tested Python package artifacts" in build_source
    assert "tools\\verify_distribution.py --exe dist\\tomiya-code-atlas\\tomiya-code-atlas.exe --gui" in exe_source
