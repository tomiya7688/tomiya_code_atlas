from pathlib import Path

from tools import context_tool


def test_repo_profile_is_bounded_and_ignores_cache(tmp_path: Path) -> None:
    (tmp_path / "Src").mkdir()
    (tmp_path / "Src" / "sample.py").write_text("x = 1\n", encoding="utf-8")
    (tmp_path / "docs").mkdir()
    (tmp_path / "docs" / "guide.md").write_text("# Guide\ntext\n", encoding="utf-8")
    (tmp_path / ".venv").mkdir()
    (tmp_path / ".venv" / "ignored.py").write_text("\n" * 100, encoding="utf-8")
    (tmp_path / ".codex").mkdir()
    (tmp_path / ".codex" / "context_pack.md").write_text("generated\n", encoding="utf-8")

    profile = context_tool.repo_profile(tmp_path)

    assert profile["files"] == 2
    assert profile["text_lines"] == 3
    assert profile["file_types"][".py"] == 1


def test_markdown_index_records_headings_only(tmp_path: Path) -> None:
    docs = tmp_path / "docs"
    docs.mkdir()
    (docs / "guide.md").write_text("# Top\ntext\n## Detail\nmore\n", encoding="utf-8")

    index = context_tool.markdown_index(tmp_path)

    assert index == [
        {
            "path": "docs/guide.md",
            "headings": [
                {"line": 1, "level": 1, "title": "Top"},
                {"line": 3, "level": 2, "title": "Detail"},
            ],
        }
    ]


def test_bounded_search_stops_at_limit_and_ignores_generated_context(tmp_path: Path) -> None:
    (tmp_path / "Src").mkdir()
    (tmp_path / "Src" / "a.py").write_text("needle\nneedle\nneedle\n", encoding="utf-8")
    (tmp_path / ".codex").mkdir()
    (tmp_path / ".codex" / "generated.md").write_text("needle\n", encoding="utf-8")

    result = context_tool.bounded_text_search("needle", tmp_path, max_results=2)

    assert len(result["results"]) == 2
    assert result["truncated"] is True
    assert all(item["path"] == "Src/a.py" for item in result["results"])


def test_path_find_supports_globs_and_is_bounded(tmp_path: Path) -> None:
    (tmp_path / "Src").mkdir()
    (tmp_path / "Src" / "a.py").write_text("", encoding="utf-8")
    (tmp_path / "Src" / "b.py").write_text("", encoding="utf-8")
    (tmp_path / "README.md").write_text("", encoding="utf-8")

    result = context_tool.bounded_path_find("Src/*.py", tmp_path, max_results=1)

    assert len(result["paths"]) == 1
    assert result["paths"][0] in {"Src/a.py", "Src/b.py"}
    assert result["truncated"] is True


def test_python_structure_index_finds_symbols_and_imports(tmp_path: Path) -> None:
    src = tmp_path / "Src"
    src.mkdir()
    (src / "sample.py").write_text(
        "import json\nfrom pathlib import Path\n\nclass Demo:\n    def run(self):\n        return Path('.')\n",
        encoding="utf-8",
    )

    index = context_tool.python_structure_index(tmp_path)

    assert index["parse_errors"] == []
    assert index["imports"]["Src/sample.py"] == ["json", "pathlib"]
    assert [(item["kind"], item["qualified_name"]) for item in index["symbols"]] == [
        ("class", "Demo"),
        ("method", "Demo.run"),
    ]


def test_validation_plan_routes_tools_architecture_and_build() -> None:
    plan = context_tool.validation_plan(
        ["tools/context_tool.py", "docs/jp/構成/UPDコマンダー適用.md", "pyproject.toml"]
    )

    assert "python -m pytest tests/test_next_issue.py tests/test_context_tool.py" in plan
    assert "python tools/context_tool.py policy-check" in plan
    assert "python -m build" in plan
    assert plan[-1] == "python -m pytest"


def test_policy_check_separates_errors_and_warnings(tmp_path: Path) -> None:
    (tmp_path / "Src" / "renderers").mkdir(parents=True)
    (tmp_path / "Src" / "languages").mkdir(parents=True)
    (tmp_path / "Src" / "generators").mkdir(parents=True)
    (tmp_path / "Src" / "renderers" / "bad.py").write_text(
        "from Src.languages import python\nfrom Src.analyzers import call_graph\n",
        encoding="utf-8",
    )
    (tmp_path / "Src" / "generators" / "legacy.py").write_text(
        "from Src.languages import python\n",
        encoding="utf-8",
    )

    findings = context_tool.architecture_findings(tmp_path)

    assert any(item.rule == "KCA101" and item.severity == "error" for item in findings)
    assert any(item.rule == "KCA201" and item.severity == "warning" for item in findings)


def test_exploration_status_requires_goal_required_acceptance_and_working_set() -> None:
    ready = context_tool.exploration_status(
        "# Task\n## Goal\nx\n## Required\ny\n## Acceptance Criteria\nz\n## Working Set\na\n## Out of Scope\nb\n"
    )
    missing = context_tool.exploration_status("## Goal\nx\n## Required\ny\n")

    assert ready["ready"] is True
    assert ready["out_of_scope_present"] is True
    assert missing["ready"] is False
    assert set(missing["missing"]) == {"acceptance", "working_set"}


def test_compact_log_keeps_failures_and_bounded_tail() -> None:
    text = "\n".join(["start", "noise1", "ERROR broken", "noise2", "noise3", "done"])

    result = context_tool.compact_log(text, max_lines=4, tail_lines=2)

    kept = [item["text"] for item in result["kept_lines"]]
    assert "ERROR broken" in kept
    assert "noise3" in kept
    assert "done" in kept
    assert len(kept) <= 4


def test_role_map_and_truth_candidates_are_compact_indexes(tmp_path: Path) -> None:
    (tmp_path / "Src").mkdir()
    (tmp_path / "Src" / "module.py").write_text("", encoding="utf-8")
    (tmp_path / "tests").mkdir()
    (tmp_path / "tests" / "test_module.py").write_text("", encoding="utf-8")
    (tmp_path / "README.md").write_text("", encoding="utf-8")
    (tmp_path / "AI_CONTEXT.md").write_text("", encoding="utf-8")
    (tmp_path / "docs" / "jp" / "機能仕様").mkdir(parents=True)

    roles = context_tool.file_role_map(tmp_path)
    truth = context_tool.source_of_truth_candidates(tmp_path)

    assert roles["source"]["count"] == 1
    assert roles["tests"]["count"] == 1
    assert {item["path"] for item in truth} >= {
        "README.md",
        "AI_CONTEXT.md",
        "docs/jp/機能仕様/",
        "tests/",
    }
