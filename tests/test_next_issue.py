from tools.next_issue import extract_sections, priority, route_working_set, select_issue


def test_priority_uses_label_before_title() -> None:
    issue = {
        "number": 10,
        "title": "[P0] title priority",
        "labels": [{"name": "p2"}],
    }

    assert priority(issue) == (2, 10)


def test_priority_falls_back_to_title_prefix() -> None:
    issue = {
        "number": 11,
        "title": "[P1] title priority",
        "labels": [],
    }

    assert priority(issue) == (1, 11)


def test_priority_title_prefix_is_case_insensitive() -> None:
    issue = {
        "number": 12,
        "title": "[p3] title priority",
        "labels": [],
    }

    assert priority(issue) == (3, 12)


def test_priority_without_label_or_prefix_is_unlabeled() -> None:
    issue = {
        "number": 13,
        "title": "No explicit priority",
        "labels": [],
    }

    assert priority(issue) == (50, 13)


def test_priority_uses_issue_number_as_stable_tiebreaker() -> None:
    older = {"number": 2, "title": "[P1] older", "labels": []}
    newer = {"number": 20, "title": "[P1] newer", "labels": []}

    assert min([newer, older], key=priority) is older


def test_select_issue_filters_explicit_meta_items() -> None:
    meta = {"number": 1, "title": "[P0] roadmap", "labels": [{"name": "meta"}]}
    work = {"number": 2, "title": "[P1] implementation", "labels": []}

    assert select_issue([meta, work]) is work


def test_extract_sections_supports_japanese_issue_headings() -> None:
    body = """## 概要
構造索引を追加する。

## 基本方針
決定論的に生成する。

## 完了条件
- [ ] JSONを生成できる
- [ ] テストが通る

## 対象外
GUIは今回扱わない。
"""

    sections = extract_sections(body)

    assert sections["goal"] == "構造索引を追加する。"
    assert sections["required"] == "決定論的に生成する。"
    assert "JSONを生成できる" in sections["acceptance"]
    assert sections["out_of_scope"] == "GUIは今回扱わない。"


def test_route_working_set_uses_title_for_existing_unlabeled_issues() -> None:
    issue = {
        "number": 20,
        "title": "[P1] コンポーネント図生成機を実装する",
        "labels": [],
    }

    route = route_working_set(issue)

    assert "Src/analyzers/" in route["source"]
    assert "Src/renderers/" in route["source"]
    assert "docs/jp/機能仕様/図表生成.md" in route["docs"]
