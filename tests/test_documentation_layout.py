from __future__ import annotations

import re
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
JAPANESE_CHARACTERS = re.compile(r"[ぁ-ゟ゠-ヿ㐀-䶿一-鿿]")
MARKDOWN_LINK = re.compile(r"\[[^\]]*\]\((?:<([^>]+)>|([^\s)]+))(?:\s+[^)]*)?\)")


def test_documentation_is_in_language_roots_and_japanese_names_are_native() -> None:
    docs = ROOT / "docs"
    markdown_files = list(docs.rglob("*.md"))

    assert markdown_files
    for path in markdown_files:
        relative = path.relative_to(docs)
        assert relative.parts[0] in {"jp", "en"}, relative
        if relative.parts[0] == "jp":
            assert JAPANESE_CHARACTERS.search(path.stem), relative
            assert JAPANESE_CHARACTERS.search(path.read_text(encoding="utf-8")), relative


def test_documentation_links_resolve() -> None:
    markdown_files = [
        *ROOT.glob("README.md"),
        *ROOT.glob("AI_CONTEXT.md"),
        *ROOT.glob("AGENTS.md"),
        *(ROOT / "docs").rglob("*.md"),
    ]

    unresolved: list[str] = []
    for source in markdown_files:
        content = source.read_text(encoding="utf-8")
        for match in MARKDOWN_LINK.finditer(content):
            target = (match.group(1) or match.group(2)).split("#", maxsplit=1)[0]
            if not target or "://" in target or target.startswith("mailto:"):
                continue
            if not target.lower().endswith(".md"):
                continue
            destination = (source.parent / target).resolve()
            if not destination.is_file():
                unresolved.append(f"{source.relative_to(ROOT)} -> {target}")

    assert unresolved == []


def test_v1_user_guides_have_english_names_and_reciprocal_japanese_links() -> None:
    docs = ROOT / "docs"
    english_files = {
        "README.md": docs / "en" / "README.md",
        "Building.md": docs / "en" / "Building.md",
        "GUI-Usage.md": docs / "en" / "GUI-Usage.md",
        "CLI-Reference.md": docs / "en" / "CLI-Reference.md",
    }
    for name, path in english_files.items():
        assert path.is_file(), name
        assert not JAPANESE_CHARACTERS.search(path.stem), name

    expected_links = {
        "README.md": "../../README.md",
        "Building.md": "../../README.md",
        "GUI-Usage.md": "../jp/GUI操作ガイド.md",
        "CLI-Reference.md": "../../README.md",
    }
    for name, japanese_source in expected_links.items():
        assert japanese_source in english_files[name].read_text(encoding="utf-8"), name

    japanese_readme = (ROOT / "README.md").read_text(encoding="utf-8")
    for name in ("README.md", "Building.md", "GUI-Usage.md", "CLI-Reference.md"):
        assert f"docs/en/{name}" in japanese_readme
    japanese_gui = (docs / "jp" / "GUI操作ガイド.md").read_text(encoding="utf-8")
    assert "../en/GUI-Usage.md" in japanese_gui


def test_english_cli_guide_keeps_real_entrypoint_and_command_names() -> None:
    guide = (ROOT / "docs" / "en" / "CLI-Reference.md").read_text(encoding="utf-8")
    for command in (
        "python app.py --help",
        "python app.py comment",
        "python app.py ci",
        "python app.py call-graph",
        "python app.py class-diagram",
        "python app.py sequence-diagram",
        "--output-dir",
        "--in-place",
    ):
        assert command in guide


def test_repository_agent_and_normative_guidance_are_japanese() -> None:
    for name in (
        "AGENTS.md",
        "AI_CONTEXT.md",
        "README.md",
        "specification/architecture-policy.md",
        "specification/parser-backend-contract.md",
    ):
        content = (ROOT / name).read_text(encoding="utf-8")
        assert JAPANESE_CHARACTERS.search(content), name
    policy = (ROOT / "specification/architecture-policy.md").read_text(encoding="utf-8")
    assert "Required（必須）" in policy
    assert "Recommended（推奨）" in policy
    assert "Advisory（助言）" in policy
