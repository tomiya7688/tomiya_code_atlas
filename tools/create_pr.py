"""Create a pull request with minimal diff reading.

Flow:
1. Read .codex/next_issue.md when available.
2. Ensure work happens on a non-main branch.
3. Run pytest; stop immediately on failure.
4. Stage and commit local changes if needed.
5. Build a compact PR body from file names and diff statistics only.
6. Push and create (or reuse) the pull request with GitHub CLI.

Requires git, Python, pytest, and GitHub CLI (`gh`).
"""

from __future__ import annotations

import re
import shutil
import subprocess
import sys
from pathlib import Path

BASE_BRANCH = "main"
ISSUE_CONTEXT = Path(".codex") / "next_issue.md"


def run(command: list[str], *, capture: bool = True, check: bool = True) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        command,
        check=check,
        capture_output=capture,
        text=True,
        encoding="utf-8",
    )


def require(command: str) -> None:
    if shutil.which(command) is None:
        sys.exit(f"ERROR: Required command was not found: {command}")


def read_issue_context() -> tuple[int | None, str | None]:
    if not ISSUE_CONTEXT.exists():
        return None, None

    text = ISSUE_CONTEXT.read_text(encoding="utf-8")
    match = re.search(r"^Issue:\s*#(\d+)\s*[—-]\s*(.+)$", text, re.MULTILINE)
    if not match:
        return None, None
    return int(match.group(1)), match.group(2).strip()


def current_branch() -> str:
    return run(["git", "branch", "--show-current"]).stdout.strip()


def ensure_work_branch(issue_number: int | None) -> str:
    branch = current_branch()
    if not branch:
        sys.exit("ERROR: Detached HEAD is not supported by scripts/dev/pull_request.bat.")

    if branch != BASE_BRANCH:
        return branch

    suffix = f"issue-{issue_number}" if issue_number is not None else "work"
    candidate = f"codex/{suffix}"

    existing = run(["git", "branch", "--list", candidate]).stdout.strip()
    if existing:
        run(["git", "switch", candidate], capture=False)
    else:
        run(["git", "switch", "-c", candidate], capture=False)
    return candidate


def run_tests() -> None:
    print("== pytest ==")
    base = Path(".pytest-pr-temp")
    base.mkdir(parents=True, exist_ok=True)
    completed = subprocess.run(
        [
            sys.executable,
            "-m",
            "pytest",
            "-p",
            "no:cacheprovider",
            "--ignore-glob=pytest-cache-files-*",
            "--basetemp",
            str(base),
        ],
        text=True,
    )
    if completed.returncode != 0:
        shutil.rmtree(base, ignore_errors=True)
        sys.exit("\nPR aborted: pytest failed.")
    shutil.rmtree(base, ignore_errors=True)
    print("pytest passed.\n")


def has_changes() -> bool:
    return bool(run(["git", "status", "--porcelain"]).stdout.strip())


def commit_changes(issue_number: int | None, issue_title: str | None) -> None:
    if not has_changes():
        return

    run(["git", "add", "-A"], capture=False)
    if issue_number is not None and issue_title:
        message = f"Resolve #{issue_number}: {issue_title}"
    elif issue_title:
        message = issue_title
    else:
        message = "Apply Codex changes"
    run(["git", "commit", "-m", message], capture=False)


def ensure_base_ref() -> str:
    # Prefer origin/main so the summary reflects what GitHub will review.
    fetched = run(["git", "fetch", "origin", BASE_BRANCH], check=False)
    if fetched.returncode == 0:
        return f"origin/{BASE_BRANCH}"
    return BASE_BRANCH


def compact_change_summary(base_ref: str) -> tuple[list[str], str]:
    names_raw = run(["git", "diff", "--name-only", f"{base_ref}...HEAD"]).stdout
    files = [line.strip() for line in names_raw.splitlines() if line.strip()]

    stat = run(["git", "diff", "--shortstat", f"{base_ref}...HEAD"]).stdout.strip()
    return files, stat or "No textual diff statistics available."


def pr_title(issue_number: int | None, issue_title: str | None) -> str:
    if issue_number is not None and issue_title:
        return f"#{issue_number} {issue_title}"
    if issue_title:
        return issue_title
    return run(["git", "log", "-1", "--pretty=%s"]).stdout.strip() or "Codex changes"


def build_body(issue_number: int | None, files: list[str], stat: str) -> str:
    listed = files[:20]
    file_lines = "\n".join(f"- `{name}`" for name in listed) if listed else "- (none)"
    if len(files) > len(listed):
        file_lines += f"\n- … and {len(files) - len(listed)} more file(s)"

    closes = f"\nCloses #{issue_number}\n" if issue_number is not None else ""
    return (
        "## Summary\n"
        "Prepared by the local one-shot PR workflow without loading the full diff into LLM context.\n\n"
        "## Change footprint\n"
        f"{stat}\n\n"
        "### Files changed\n"
        f"{file_lines}\n\n"
        "## Verification\n"
        "- `python -m pytest` passed locally.\n"
        f"{closes}"
    )


def existing_pr_url(branch: str) -> str | None:
    completed = run(
        ["gh", "pr", "view", branch, "--json", "url", "--jq", ".url"],
        check=False,
    )
    url = completed.stdout.strip()
    return url if completed.returncode == 0 and url else None


def main() -> int:
    require("git")
    require("gh")

    issue_number, issue_title = read_issue_context()
    branch = ensure_work_branch(issue_number)

    run_tests()
    commit_changes(issue_number, issue_title)

    base_ref = ensure_base_ref()
    files, stat = compact_change_summary(base_ref)
    if not files:
        print("No changes relative to main; no pull request is needed.")
        return 0

    print("== compact review ==")
    print(stat)
    for name in files[:20]:
        print(f"  {name}")
    if len(files) > 20:
        print(f"  ... and {len(files) - 20} more file(s)")

    run(["git", "push", "-u", "origin", branch], capture=False)

    existing = existing_pr_url(branch)
    if existing:
        print(f"\nPull request already exists: {existing}")
        return 0

    title = pr_title(issue_number, issue_title)
    body = build_body(issue_number, files, stat)
    created = run(
        [
            "gh", "pr", "create",
            "--base", BASE_BRANCH,
            "--head", branch,
            "--title", title,
            "--body", body,
        ]
    )
    print(f"\nPull request created: {created.stdout.strip()}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
