"""Select one highest-priority actionable GitHub issue and build a compact Task Capsule.

Requires GitHub CLI (`gh`) to be installed and authenticated.  Selection, extraction,
and routing are deterministic so task setup does not spend LLM context on broad Issue
or documentation exploration.
"""

from __future__ import annotations

import json
import re
import subprocess
import sys
from pathlib import Path

REPO = "tomiya7688/tomiya_code_atlas"
OUTPUT = Path(".codex") / "next_issue.md"

PRIORITY_LABELS = {
    "p0": 0,
    "priority:p0": 0,
    "priority:critical": 0,
    "critical": 0,
    "p1": 1,
    "priority:p1": 1,
    "priority:high": 1,
    "high priority": 1,
    "p2": 2,
    "priority:p2": 2,
    "priority:medium": 2,
    "medium priority": 2,
    "p3": 3,
    "priority:p3": 3,
    "priority:low": 3,
    "low priority": 3,
}
TITLE_PRIORITY_RE = re.compile(r"^\s*\[P([0-3])\]", re.IGNORECASE)
NON_ACTIONABLE_LABELS = {
    "meta",
    "roadmap",
    "umbrella",
    "index",
    "policy-only",
    "type:meta",
    "type:roadmap",
}

SPEC_LABELS = {
    "spec:comment-generator": "docs/specs/comment_generator.md",
    "spec:comments": "docs/specs/comment_generator.md",
    "spec:diagram": "docs/specs/diagrams.md",
    "spec:diagrams": "docs/specs/diagrams.md",
    "spec:class-diagram": "docs/specs/diagrams.md",
    "spec:object-diagram": "docs/specs/diagrams.md",
    "spec:sequence-diagram": "docs/specs/diagrams.md",
    "spec:package-diagram": "docs/specs/diagrams.md",
    "spec:use-case-diagram": "docs/specs/diagrams.md",
    "spec:communication-diagram": "docs/specs/diagrams.md",
    "spec:activity-diagram": "docs/specs/diagrams.md",
    "spec:component-diagram": "docs/specs/diagrams.md",
    "spec:deployment-diagram": "docs/specs/diagrams.md",
    "spec:state-machine-diagram": "docs/specs/diagrams.md",
    "spec:timing-diagram": "docs/specs/diagrams.md",
    "spec:call-graph": "docs/specs/diagrams.md",
    "spec:class-responsibility-table": "docs/specs/class_responsibility_table.md",
    "spec:responsibility-table": "docs/specs/class_responsibility_table.md",
    "spec:ci-analyzer": "docs/specs/ci_analyzer.md",
    "spec:ci": "docs/specs/ci_analyzer.md",
    "spec:design-evaluation": "docs/specs/design_evaluation.md",
    "spec:evaluation": "docs/specs/design_evaluation.md",
}

# Title/label routing is a fallback for existing Issues that do not yet carry rich labels.
TASK_ROUTES = [
    (
        re.compile(r"comment|コメント", re.IGNORECASE),
        ["Src/generators/", "Src/languages/", "Src/models/"],
        ["tests/test_comment_generator.py", "tests/test_python_comment_generator.py", "tests/test_csharp_comment_generator.py"],
        ["docs/specs/comment_generator.md"],
    ),
    (
        re.compile(r"common ir|中間表現|\bIR\b", re.IGNORECASE),
        ["Src/analyzers/", "Src/models/", "Src/languages/"],
        ["tests/test_public_exports.py", "tests/test_call_graph.py"],
        ["specification/architecture-policy.md"],
    ),
    (
        re.compile(r"call.?graph|class diagram|sequence|communication|package|component|deployment|state|timing|activity|diagram|図", re.IGNORECASE),
        ["Src/analyzers/", "Src/generators/", "Src/renderers/"],
        ["tests/test_call_graph.py", "tests/test_call_graph_generator.py"],
        ["docs/specs/diagrams.md"],
    ),
    (
        re.compile(r"gui|ui|画面", re.IGNORECASE),
        ["app.py", "Src/"],
        ["tests/"],
        ["docs/architecture/upd_commander.md", "specification/architecture-policy.md"],
    ),
    (
        re.compile(r"ci|workflow|build|package|ビルド", re.IGNORECASE),
        [".github/workflows/", "pyproject.toml", "tools/"],
        ["tests/"],
        ["docs/project_operations.md"],
    ),
    (
        re.compile(r"context|運用|commander|architecture|設計基盤", re.IGNORECASE),
        ["tools/", "AI_CONTEXT.md", "AGENTS.md", "docs/", "specification/"],
        ["tests/test_next_issue.py", "tests/test_context_tool.py"],
        ["docs/project_operations.md", "docs/architecture/upd_commander.md", "specification/architecture-policy.md"],
    ),
]


def run_gh() -> list[dict]:
    command = [
        "gh", "issue", "list",
        "--repo", REPO,
        "--state", "open",
        "--limit", "100",
        "--json", "number,title,body,labels,url,updatedAt",
    ]
    try:
        completed = subprocess.run(
            command,
            check=True,
            capture_output=True,
            text=True,
            encoding="utf-8",
        )
    except FileNotFoundError:
        sys.exit("ERROR: GitHub CLI (gh) was not found. Install it and run `gh auth login`.")
    except subprocess.CalledProcessError as exc:
        detail = (exc.stderr or exc.stdout or "gh failed").strip()
        sys.exit(f"ERROR: Could not read GitHub issues: {detail}")
    return json.loads(completed.stdout)


def label_names(issue: dict) -> set[str]:
    return {
        label.get("name", "").strip().lower()
        for label in issue.get("labels", [])
        if label.get("name")
    }


def is_actionable(issue: dict) -> bool:
    return not bool(label_names(issue) & NON_ACTIONABLE_LABELS)


def priority(issue: dict) -> tuple[int, int]:
    labels = label_names(issue)
    label_ranks = [PRIORITY_LABELS[name] for name in labels if name in PRIORITY_LABELS]
    if label_ranks:
        rank = min(label_ranks)
    else:
        title_match = TITLE_PRIORITY_RE.match(issue.get("title", ""))
        rank = int(title_match.group(1)) if title_match else 50
    return rank, int(issue["number"])


def select_issue(issues: list[dict]) -> dict:
    actionable = [issue for issue in issues if is_actionable(issue)]
    candidates = actionable or issues
    return min(candidates, key=priority)


def related_specs(issue: dict) -> list[str]:
    paths: list[str] = []
    seen: set[str] = set()
    for name in label_names(issue):
        path = SPEC_LABELS.get(name)
        if path and path not in seen:
            seen.add(path)
            paths.append(path)
    return paths


def _clean_markdown(lines: list[str], max_chars: int = 900) -> str:
    output: list[str] = []
    in_code_block = False
    for raw in lines:
        line = raw.strip()
        if line.startswith("```"):
            in_code_block = not in_code_block
            continue
        if in_code_block or not line:
            continue
        line = re.sub(r"^[-*+]\s+", "", line)
        line = re.sub(r"^\d+[.)]\s+", "", line)
        line = re.sub(r"^- \[[ xX]\]\s*", "", line)
        line = re.sub(r"\[(.*?)\]\([^)]*\)", r"\1", line)
        output.append(line)
    text = re.sub(r"\s+", " ", " ".join(output)).strip()
    if len(text) > max_chars:
        return text[: max_chars - 1].rstrip() + "…"
    return text


def compact_body(body: str, max_chars: int = 900) -> str:
    if not body:
        return "No description provided."
    lines = [re.sub(r"^#{1,6}\s*", "", line) for line in body.splitlines()]
    return _clean_markdown(lines, max_chars) or "No usable description provided."


def extract_sections(body: str) -> dict[str, str]:
    """Extract Goal/Required/Acceptance/Out-of-Scope from common Issue headings."""
    buckets: dict[str, list[str]] = {"goal": [], "required": [], "acceptance": [], "out_of_scope": []}
    current: str | None = None
    for raw in body.splitlines():
        heading = re.match(r"^#{1,6}\s+(.+?)\s*$", raw.strip())
        if heading:
            title = heading.group(1).strip().lower()
            if any(key in title for key in ("goal", "目的", "概要")):
                current = "goal"
            elif any(key in title for key in ("required", "requirement", "要件", "必須", "基本方針")):
                current = "required"
            elif any(key in title for key in ("acceptance", "完了条件", "completion")):
                current = "acceptance"
            elif any(key in title for key in ("out of scope", "deferred", "対象外", "今回はやらない")):
                current = "out_of_scope"
            else:
                current = None
            continue
        if current:
            buckets[current].append(raw)

    return {
        name: _clean_markdown(lines, 700)
        for name, lines in buckets.items()
        if _clean_markdown(lines, 700)
    }


def route_working_set(issue: dict) -> dict[str, list[str]]:
    haystack = issue.get("title", "") + " " + " ".join(label_names(issue))
    source: list[str] = []
    tests: list[str] = []
    docs: list[str] = []
    for pattern, route_source, route_tests, route_docs in TASK_ROUTES:
        if pattern.search(haystack):
            source.extend(route_source)
            tests.extend(route_tests)
            docs.extend(route_docs)
    docs.extend(related_specs(issue))
    return {
        "source": list(dict.fromkeys(source)),
        "tests": list(dict.fromkeys(tests)),
        "docs": list(dict.fromkeys(docs)),
    }


def main() -> int:
    issues = run_gh()
    if not issues:
        print("No open issues.")
        return 0

    issue = select_issue(issues)
    labels = [label.get("name", "") for label in issue.get("labels", [])]
    rank, _ = priority(issue)
    priority_text = f"P{rank}" if rank < 4 else "unlabeled"
    summary = compact_body(issue.get("body") or "")
    sections = extract_sections(issue.get("body") or "")
    route = route_working_set(issue)

    lines = [
        "# Task Capsule",
        "",
        f"Issue: #{issue['number']} — {issue['title']}",
        f"Priority: {priority_text}",
        f"Labels: {', '.join(labels) if labels else '(none)'}",
        f"URL: {issue['url']}",
        "",
        "## Goal",
        sections.get("goal", summary),
        "",
        "## Required",
        sections.get("required", "Read the source of truth and preserve repository architecture constraints."),
        "",
        "## Acceptance",
        sections.get("acceptance", "Use the Issue completion criteria; if they are ambiguous, inspect the original Issue before implementation."),
        "",
        "## Out of Scope",
        sections.get("out_of_scope", "Do not add unrelated refactors or deferred features."),
        "",
        "## Working Set",
        "### Source candidates",
    ]
    lines.extend(f"- {path}" for path in route["source"])
    if not route["source"]:
        lines.append("- Search first; identify the minimum source area from the Issue before reading broadly.")
    lines.append("### Matching tests")
    lines.extend(f"- {path}" for path in route["tests"])
    if not route["tests"]:
        lines.append("- Identify matching targeted tests before implementation.")
    lines.append("### Routed references")
    lines.extend(["- AI_CONTEXT.md", "- AGENTS.md", "- docs/responsibility_map.md", "- specification/architecture-policy.md"])
    lines.extend(f"- {path}" for path in route["docs"])

    lines.extend(
        [
            "",
            "## Exploration Status",
            "- Goal understood: check before broad exploration",
            "- Required known: check before broad exploration",
            "- Acceptance known: check before broad exploration",
            "- Working set identified: use routed candidates, then search",
            "- Stop condition: stop broad exploration when all four are sufficient",
            "",
            "## Validation",
            "- Run targeted checks for the changed area first.",
            "- Use `context.bat validation-plan` / `./context.sh validation-plan` for a deterministic plan after files change.",
            "- Run policy checks for architecture-sensitive changes.",
            "- Record anything not executed as Unverified.",
            "",
            "## Instructions",
            "Search first, read second. Treat this capsule as an index, not source of truth.",
            "Return to the original Issue/source/tests/specification when the capsule is insufficient.",
            "Do not load unrelated Issues/docs/history or a full diff by default.",
            "Use `context.bat remote-delta` when concurrent remote edits are possible.",
            "Use `context.bat context-pack` after a working set exists if a richer temporary packet is useful.",
            "",
        ]
    )

    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    OUTPUT.write_text("\n".join(lines), encoding="utf-8")

    print(f"[{priority_text}] #{issue['number']} {issue['title']}")
    print(sections.get("goal", summary))
    routed = route["source"] + route["tests"] + route["docs"]
    print("Routed: " + (", ".join(routed) if routed else "search-first fallback"))
    print(f"Task capsule written to: {OUTPUT}")
    print(issue["url"])
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
