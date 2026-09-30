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
