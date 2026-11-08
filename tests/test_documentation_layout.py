from __future__ import annotations

import re
from pathlib import Path
from urllib.parse import unquote, urlsplit


ROOT = Path(__file__).resolve().parents[1]
JAPANESE_CHARACTERS = re.compile(r"[ぁ-ゟ゠-ヿ㐀-䶿一-鿿]")
MARKDOWN_LINK = re.compile(r"\[[^\]]*\]\((?:<([^>]+)>|([^\s)]+))(?:\s+[^)]*)?\)")


# {
#   責務: [local_markdown_path: Markdownリンクから検証対象のlocal pathを取り出す]
#   処理: [1: URL成分を分離する; 2: 外部URLを除外する; 3: pathだけをURL decodeする]
#   引数: [target: Markdownリンクの参照先]
#   戻り値: [path: query/fragmentを除いたlocal path、または外部URLなら空文字]
# }
def local_markdown_path(target: str) -> str:
    parsed = urlsplit(target)
    if parsed.scheme or parsed.netloc:
        return ""
    return unquote(parsed.path)


def test_documentation_is_in_language_roots_and_japanese_names_are_native() -> None:
    docs = ROOT / "docs"
    markdown_files = list(docs.rglob("*.md"))

    assert markdown_files
    for path in markdown_files:
        relative = path.relative_to(docs)
        assert relative.parts[0] in {"jp", "en"}, relative
        if relative.parts[0] == "jp":
            assert JAPANESE_CHARACTERS.search(path.stem), relative
            assert JAPANESE_CHARACTERS.search(path.read_text(encoding="utf-8")), (
                relative
            )


# {
#   責務: [test_documentation_links_resolve: 文書間のlocal Markdownリンクが存在するfileを指すことを確認する]
#   処理: [1: 文書を列挙する; 2: local pathを解決する; 3: 未解決リンクがないことを確認する]
#   引数: [なし]
#   戻り値: [なし: pytest assertionで検証する]
# }
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
            target = local_markdown_path(match.group(1) or match.group(2))
            if not target or "://" in target or target.startswith("mailto:"):
                continue
            if not target.lower().endswith(".md"):
                continue
            destination = (source.parent / target).resolve()
            if not destination.is_file():
                unresolved.append(f"{source.relative_to(ROOT)} -> {target}")

    assert unresolved == []


# {
#   責務: [test_local_markdown_path_decodes_only_the_path_component: path成分だけをdecodeしquery/fragmentを除外することを確認する]
#   処理: [1: 日本語pathとquery/fragmentを入力する; 2: 外部URLを入力する; 3: 期待値と比較する]
#   引数: [なし]
#   戻り値: [なし: pytest assertionで検証する]
# }
def test_local_markdown_path_decodes_only_the_path_component() -> None:
    assert local_markdown_path("../jp/%E5%9B%B3.md?view=1#details") == "../jp/図.md"
    assert local_markdown_path("https://example.com/diagram.md") == ""


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
        "CLI-Reference.md": "../jp/CLI利用ガイド.md",
    }
    for name, japanese_source in expected_links.items():
        guide = english_files[name].read_text(encoding="utf-8")
        local_targets = {
            local_markdown_path(target)
            for match in MARKDOWN_LINK.finditer(guide)
            for target in match.groups()
            if target
        }
        assert japanese_source in local_targets, name

    japanese_readme = (ROOT / "README.md").read_text(encoding="utf-8")
    for name in ("README.md", "Building.md", "GUI-Usage.md", "CLI-Reference.md"):
        assert f"docs/en/{name}" in japanese_readme
    japanese_gui = (docs / "jp" / "GUI操作ガイド.md").read_text(encoding="utf-8")
    assert "../en/GUI-Usage.md" in japanese_gui


def test_english_cli_guide_keeps_real_entrypoint_and_command_names() -> None:
    guide = (ROOT / "docs" / "en" / "CLI-Reference.md").read_text(encoding="utf-8")
    for command in (
        "python -m Src.app --help",
        "python -m Src.app comment",
        "python -m Src.app ci",
        "python -m Src.app call-graph",
        "python -m Src.app class-diagram",
        "python -m Src.app sequence-diagram",
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
