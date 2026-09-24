"""Dependency-light, bounded context tools for Tomiya Code Atlas.

Outputs are indexes and working aids, never replacements for Issue/source/tests/docs.
"""

from __future__ import annotations

import argparse
import ast
import fnmatch
import json
import re
import subprocess
from collections import Counter
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Iterable

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_BASE = "main"
IGNORED_DIRS = {
    ".git", ".codex", ".venv", "venv", "__pycache__", ".pytest_cache",
    ".mypy_cache", ".ruff_cache", "build", "dist", "htmlcov",
}
TEXT_SUFFIXES = {
    ".py", ".md", ".txt", ".toml", ".yml", ".yaml", ".json", ".cs",
    ".gd", ".cpp", ".cc", ".c", ".h", ".hpp", ".java", ".go", ".bat", ".sh",
}
ERROR_WORDS = re.compile(r"error|warning|fail(?:ed|ure)?|traceback|exception|assert", re.IGNORECASE)


@dataclass(frozen=True)
class Finding:
    path: str
    line: int
    rule: str
    severity: str
    message: str


@dataclass(frozen=True)
class Symbol:
    path: str
    kind: str
    qualified_name: str
    line: int
    end_line: int


def _run_git(args: list[str], *, root: Path = ROOT, check: bool = True) -> str:
    completed = subprocess.run(
        ["git", "-C", str(root), *args], check=check, capture_output=True,
        text=True, encoding="utf-8",
    )
    return completed.stdout.strip()


def iter_repo_files(root: Path = ROOT) -> Iterable[Path]:
    for path in root.rglob("*"):
        if not path.is_file():
            continue
        try:
            rel = path.relative_to(root)
        except ValueError:
            continue
        if any(part in IGNORED_DIRS or part.endswith(".egg-info") for part in rel.parts):
            continue
        yield path


def repo_profile(root: Path = ROOT) -> dict:
    files = list(iter_repo_files(root))
    suffixes: Counter[str] = Counter()
    lines: Counter[str] = Counter()
    largest: list[tuple[int, str]] = []
    text_chars = 0
    for path in files:
        suffix = path.suffix.lower() or "(none)"
        suffixes[suffix] += 1
        if suffix not in TEXT_SUFFIXES:
            continue
        try:
            text = path.read_text(encoding="utf-8")
        except (UnicodeDecodeError, OSError):
            continue
        count = len(text.splitlines())
        lines[suffix] += count
        text_chars += len(text)
        largest.append((count, path.relative_to(root).as_posix()))
    largest.sort(reverse=True)
    return {
        "files": len(files),
        "text_lines": sum(lines.values()),
        "approx_full_read_tokens": text_chars // 4,
        "file_types": dict(suffixes.most_common()),
        "lines_by_type": dict(lines.most_common()),
        "largest_text_files": [{"path": path, "lines": count} for count, path in largest[:12]],
    }


def markdown_index(root: Path = ROOT, directory: str = "docs") -> list[dict]:
    base = root / directory
    result: list[dict] = []
    if not base.exists():
        return result
    for path in sorted(base.rglob("*.md")):
        headings = []
        try:
            for number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
                match = re.match(r"^(#{1,4})\s+(.+?)\s*$", line)
                if match:
                    headings.append({"line": number, "level": len(match.group(1)), "title": match.group(2)})
        except OSError:
            continue
        result.append({"path": path.relative_to(root).as_posix(), "headings": headings})
    return result


def bounded_text_search(
    query: str,
    root: Path = ROOT,
    *,
    regex: bool = False,
    case_sensitive: bool = False,
    max_results: int = 40,
    max_file_chars: int = 500_000,
) -> dict:
    flags = 0 if case_sensitive else re.IGNORECASE
    pattern = re.compile(query if regex else re.escape(query), flags)
    results: list[dict] = []
    scanned = 0
    for path in iter_repo_files(root):
        if path.suffix.lower() not in TEXT_SUFFIXES:
            continue
        try:
            if path.stat().st_size > max_file_chars:
                continue
            lines = path.read_text(encoding="utf-8").splitlines()
        except (UnicodeDecodeError, OSError):
            continue
        scanned += 1
        for number, line in enumerate(lines, 1):
            if pattern.search(line):
                results.append({
                    "path": path.relative_to(root).as_posix(),
                    "line": number,
                    "text": line.strip()[:240],
                })
                if len(results) >= max_results:
                    return {"query": query, "results": results, "truncated": True, "files_scanned": scanned}
    return {"query": query, "results": results, "truncated": False, "files_scanned": scanned}


def bounded_path_find(pattern: str, root: Path = ROOT, *, max_results: int = 80) -> dict:
    matches: list[str] = []
    lowered = pattern.lower()
    wildcard = any(char in pattern for char in "*?[")
    for path in iter_repo_files(root):
        rel = path.relative_to(root).as_posix()
        hit = fnmatch.fnmatch(rel, pattern) if wildcard else lowered in rel.lower()
        if hit:
            matches.append(rel)
            if len(matches) >= max_results:
                return {"pattern": pattern, "paths": matches, "truncated": True}
    return {"pattern": pattern, "paths": matches, "truncated": False}


def file_role_map(root: Path = ROOT) -> dict:
    roles: dict[str, list[str]] = {
        "source": [], "tests": [], "docs": [], "tooling": [], "config": [], "other": [],
    }
    for path in iter_repo_files(root):
        rel = path.relative_to(root).as_posix()
        if rel.startswith("Src/"):
            role = "source"
        elif rel.startswith("tests/"):
            role = "tests"
        elif rel.startswith(("docs/", "specification/")) or rel in {"README.md", "AI_CONTEXT.md", "AGENTS.md"}:
            role = "docs"
        elif rel.startswith("tools/") or path.suffix.lower() in {".bat", ".sh"}:
            role = "tooling"
        elif rel.startswith(".github/") or path.name in {"pyproject.toml", ".gitignore"}:
            role = "config"
        else:
            role = "other"
        roles[role].append(rel)
    return {key: {"count": len(value), "samples": value[:20]} for key, value in roles.items()}


def source_of_truth_candidates(root: Path = ROOT) -> list[dict]:
    candidates = []
    priority_names = {
        "AI_CONTEXT.md": "AI routing index",
        "AGENTS.md": "agent/coding rules",
        "README.md": "human overview",
        "pyproject.toml": "Python package/build configuration",
    }
    for name, role in priority_names.items():
        if (root / name).exists():
            candidates.append({"path": name, "role": role})
    for directory, role in (("specification", "normative policy"), ("docs/specs", "feature specification"), ("tests", "executable behavior")):
        if (root / directory).exists():
            candidates.append({"path": directory + "/", "role": role})
    return candidates


def python_structure_index(root: Path = ROOT) -> dict:
    symbols: list[Symbol] = []
    imports: dict[str, list[str]] = {}
    parse_errors: list[dict] = []
    for top in (root / "Src", root / "tools"):
        if not top.exists():
            continue
        for path in sorted(top.rglob("*.py")):
            if any(part in IGNORED_DIRS for part in path.parts):
                continue
            rel = path.relative_to(root).as_posix()
            try:
                tree = ast.parse(path.read_text(encoding="utf-8"), filename=rel)
            except (SyntaxError, UnicodeDecodeError, OSError) as exc:
                parse_errors.append({"path": rel, "error": str(exc)})
                continue
            imported: set[str] = set()
            for node in ast.walk(tree):
                if isinstance(node, ast.Import):
                    imported.update(alias.name for alias in node.names)
                elif isinstance(node, ast.ImportFrom) and node.module:
                    imported.add(node.module)
            imports[rel] = sorted(imported)

            def walk(body: list[ast.stmt], parents: tuple[str, ...] = ()) -> None:
                for node in body:
                    if isinstance(node, ast.ClassDef):
                        name = ".".join((*parents, node.name))
                        symbols.append(Symbol(rel, "class", name, node.lineno, getattr(node, "end_lineno", node.lineno)))
                        walk(node.body, (*parents, node.name))
                    elif isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
                        name = ".".join((*parents, node.name))
                        kind = "method" if parents else "function"
                        symbols.append(Symbol(rel, kind, name, node.lineno, getattr(node, "end_lineno", node.lineno)))
                        walk(node.body, (*parents, node.name))
            walk(tree.body)
    return {"symbols": [asdict(item) for item in symbols], "imports": imports, "parse_errors": parse_errors}


def compact_diff(root: Path = ROOT, base: str = "origin/main") -> dict:
    changed = _run_git(["diff", "--name-status", f"{base}...HEAD"], root=root, check=False)
    stat = _run_git(["diff", "--shortstat", f"{base}...HEAD"], root=root, check=False)
    commits = _run_git(["log", "--format=%h %s", f"{base}..HEAD", "-n", "12"], root=root, check=False)
    return {
        "base": base,
        "changed_files": changed.splitlines() if changed else [],
        "shortstat": stat or "no textual diff",
        "commits": commits.splitlines() if commits else [],
    }


def remote_delta(
    root: Path = ROOT,
    base: str = DEFAULT_BASE,
    *,
    fetch: bool = True,
    excerpt_lines: int = 100,
    fast_forward: bool = False,
) -> dict:
    remote_ref = f"origin/{base}"
    if fetch:
        subprocess.run(["git", "-C", str(root), "fetch", "origin", base], capture_output=True, text=True)
    local = _run_git(["rev-parse", "HEAD"], root=root)
    remote = _run_git(["rev-parse", remote_ref], root=root, check=False)
    if not remote:
        return {"error": f"remote ref not available: {remote_ref}", "local": local}
    merge_base = _run_git(["merge-base", "HEAD", remote_ref], root=root)
    ahead_text, behind_text = _run_git(["rev-list", "--left-right", "--count", f"HEAD...{remote_ref}"], root=root).split()
    ahead, behind = int(ahead_text), int(behind_text)
    commits = _run_git(["log", "--format=%h %s", f"HEAD..{remote_ref}", "-n", "12"], root=root, check=False)
    files = _run_git(["diff", "--name-status", f"{merge_base}..{remote_ref}"], root=root, check=False)
    stat = _run_git(["diff", "--shortstat", f"{merge_base}..{remote_ref}"], root=root, check=False)
    excerpt = _run_git(["diff", "--unified=1", f"{merge_base}..{remote_ref}"], root=root, check=False).splitlines()
    updated = False
    if fast_forward and behind:
        dirty = bool(_run_git(["status", "--porcelain"], root=root, check=False))
        if dirty:
            raise RuntimeError("refusing fast-forward: worktree is dirty")
        if ahead:
            raise RuntimeError("refusing fast-forward: local branch has unpushed commits")
        subprocess.run(["git", "-C", str(root), "merge", "--ff-only", remote_ref], check=True)
        updated = True
    return {
        "local": local, "remote": remote, "merge_base": merge_base,
        "ahead": ahead, "behind": behind,
        "remote_commits": commits.splitlines() if commits else [],
        "changed_files": files.splitlines() if files else [],
        "shortstat": stat or "no textual diff",
        "diff_excerpt": excerpt[:excerpt_lines],
        "diff_excerpt_truncated": len(excerpt) > excerpt_lines,
        "fast_forwarded": updated,
    }


def validation_plan(paths: Iterable[str]) -> list[str]:
    files = [path.replace("\\", "/") for path in paths]
    plan: list[str] = []
    docs_only = bool(files) and all(path.endswith(".md") for path in files)
    if any(path.startswith("tools/") or path.endswith((".bat", ".sh")) for path in files):
        plan.append("python -m pytest tests/test_next_issue.py tests/test_context_tool.py")
    if any(path.startswith("Src/languages/") for path in files):
        plan.append("python -m pytest -k 'python or csharp or language or comment'")
    if any(path.startswith(("Src/analyzers/", "Src/generators/", "Src/renderers/")) for path in files):
        plan.append("python -m pytest -k 'call_graph or generator or renderer'")
    if any(path.startswith(("specification/", "docs/architecture/", "Src/")) for path in files):
        plan.append("python tools/context_tool.py policy-check")
    if any(path in {"app.py", "run.bat"} for path in files):
        plan.append("python app.py")
    if any(path == "pyproject.toml" or path.startswith(".github/workflows/") for path in files):
        plan.extend(["python app.py", "python -m build"])
    if not docs_only:
        plan.append("python -m pytest")
    if not plan:
        plan.append("documentation/path consistency review")
    return list(dict.fromkeys(plan))


def compact_log(text: str, *, max_lines: int = 80, tail_lines: int = 20) -> dict:
    lines = text.splitlines()
    selected: list[tuple[int, str]] = []
    seen: set[int] = set()
    for index, line in enumerate(lines):
        if ERROR_WORDS.search(line):
            selected.append((index + 1, line[:400]))
            seen.add(index)
    tail_start = max(0, len(lines) - max(0, tail_lines))
    for index in range(tail_start, len(lines)):
        if index not in seen:
            selected.append((index + 1, lines[index][:400]))
    selected.sort(key=lambda item: item[0])
    truncated = len(selected) > max_lines
    selected = selected[:max_lines]
    return {
        "input_lines": len(lines),
        "kept_lines": [{"line": number, "text": line} for number, line in selected],
        "truncated": truncated,
    }


def exploration_status(text: str) -> dict:
    headings = {match.group(1).strip().lower() for match in re.finditer(r"^##\s+(.+?)\s*$", text, re.MULTILINE)}
    aliases = {
        "goal": {"goal"},
        "required": {"required", "required constraints"},
        "acceptance": {"acceptance", "acceptance criteria"},
        "working_set": {"working set", "required context"},
    }
    status = {key: bool(names & headings) for key, names in aliases.items()}
    missing = [key for key, present in status.items() if not present]
    return {"ready": not missing, "status": status, "missing": missing, "out_of_scope_present": bool({"out of scope", "deferred"} & headings)}


def _module_name(path: Path, root: Path) -> str:
    return ".".join(path.relative_to(root).with_suffix("").parts)


def architecture_findings(root: Path = ROOT) -> list[Finding]:
    findings: list[Finding] = []
    src = root / "Src"
    if not src.exists():
        return findings
    forbidden_errors = {
        "Src.languages": ("Src.renderers", "Src.evaluators"),
        "Src.renderers": ("Src.languages",),
        "Src.analyzers": ("Src.renderers",),
        "Src.models": ("Src.languages", "Src.generators", "Src.renderers", "Src.evaluators"),
    }
    warning_dependencies = {"Src.generators": ("Src.languages",), "Src.renderers": ("Src.analyzers",)}
    for path in sorted(src.rglob("*.py")):
        rel = path.relative_to(root).as_posix()
        source_module = _module_name(path, root)
        try:
            tree = ast.parse(path.read_text(encoding="utf-8"), filename=rel)
        except (SyntaxError, OSError, UnicodeDecodeError) as exc:
            findings.append(Finding(rel, 1, "KCA001", "error", f"cannot parse module: {exc}"))
            continue
        imports: list[tuple[int, str]] = []
        for node in ast.walk(tree):
            if isinstance(node, ast.Import):
                imports.extend((node.lineno, alias.name) for alias in node.names)
            elif isinstance(node, ast.ImportFrom) and node.module:
                imports.append((node.lineno, node.module))
        for prefix, targets in forbidden_errors.items():
            if source_module.startswith(prefix):
                for line, imported in imports:
                    if any(imported.startswith(target) for target in targets):
                        findings.append(Finding(rel, line, "KCA101", "error", f"forbidden dependency: {source_module} -> {imported}"))
        for prefix, targets in warning_dependencies.items():
            if source_module.startswith(prefix):
                for line, imported in imports:
                    if any(imported.startswith(target) for target in targets):
                        findings.append(Finding(rel, line, "KCA201", "warning", f"boundary coupling to review: {source_module} -> {imported}"))
        if "commander" in path.stem.lower():
            for node in ast.walk(tree):
                if isinstance(node, (ast.For, ast.While, ast.BinOp)):
                    findings.append(Finding(rel, getattr(node, "lineno", 1), "UPD201", "warning", "Commander contains processing-like logic"))
                if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and node.func.id == "open":
                    findings.append(Finding(rel, node.lineno, "UPD203", "error", "Commander performs direct file I/O"))
    return findings


def current_changed_paths(root: Path = ROOT, base: str = "origin/main") -> list[str]:
    text = _run_git(["diff", "--name-only", f"{base}...HEAD"], root=root, check=False)
    return [line for line in text.splitlines() if line]


def build_context_pack(root: Path = ROOT, base: str = "origin/main") -> str:
    task_file = root / ".codex" / "next_issue.md"
    task = task_file.read_text(encoding="utf-8") if task_file.exists() else "No generated task capsule. Run next_issue.bat first."
    paths = current_changed_paths(root, base)
    plan = validation_plan(paths)
    diff = compact_diff(root, base)
    remote = remote_delta(root, DEFAULT_BASE, fetch=False, excerpt_lines=30)
    stop = exploration_status(task)
    lines = [
        "# Context Pack", "",
        "> Generated working index; source-of-truth remains Issue/source/tests/docs.", "",
        "## Task Capsule", task.strip(), "",
        "## Working Set", *(f"- `{path}`" for path in paths[:30]),
        "- No changed files yet." if not paths else "",
        "## Validation Plan", *(f"- `{item}`" for item in plan), "",
        "## Compact Change Summary", f"- {diff['shortstat']}",
        *(f"- {item}" for item in diff["changed_files"][:30]), "", "## Remote Delta",
    ]
    if "error" in remote:
        lines.append(f"- {remote['error']}")
    else:
        lines.extend([f"- ahead: {remote['ahead']}", f"- behind: {remote['behind']}", f"- {remote['shortstat']}"])
    lines.extend([
        "", "## Exploration Status",
        f"- ready to stop broad exploration: {'yes' if stop['ready'] else 'no'}",
        f"- missing: {', '.join(stop['missing']) if stop['missing'] else 'none'}",
        "", "## Unverified", "- Fill during implementation / validation.", "",
    ])
    return "\n".join(lines)


def _print_json(value: object) -> None:
    print(json.dumps(value, ensure_ascii=False, indent=2))


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Tomiya Code Atlas low-context toolbox")
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("profile", help="compact repository profile and context budget")
    docs = sub.add_parser("doc-index", help="compact Markdown heading index")
    docs.add_argument("directory", nargs="?", default="docs")
    sub.add_parser("structure-index", help="Python symbol/import structure index")
    sub.add_parser("role-map", help="classify repository files by information role")
    sub.add_parser("truth-candidates", help="list likely source-of-truth locations without asserting authority")

    search = sub.add_parser("search", help="bounded dependency-free text search")
    search.add_argument("query")
    search.add_argument("--regex", action="store_true")
    search.add_argument("--case-sensitive", action="store_true")
    search.add_argument("--max-results", type=int, default=40)

    find = sub.add_parser("path-find", help="bounded dependency-free path search")
    find.add_argument("pattern")
    find.add_argument("--max-results", type=int, default=80)

    remote = sub.add_parser("remote-delta", help="bounded summary of remote changes")
    remote.add_argument("--base", default=DEFAULT_BASE)
    remote.add_argument("--no-fetch", action="store_true")
    remote.add_argument("--ff", action="store_true", help="safe fast-forward only; refuses dirty/diverged state")
    remote.add_argument("--excerpt-lines", type=int, default=100)

    diff = sub.add_parser("compact-diff", help="changed files, shortstat and commit subjects")
    diff.add_argument("--base", default="origin/main")
    validate = sub.add_parser("validation-plan", help="smallest sufficient validation plan from changed files")
    validate.add_argument("--base", default="origin/main")
    policy = sub.add_parser("policy-check", help="compact architecture/UPD boundary check")
    policy.add_argument("--fail-on", choices=("error", "warning", "never"), default="error")
    pack = sub.add_parser("context-pack", help="generate .codex/context_pack.md")
    pack.add_argument("--base", default="origin/main")

    stop = sub.add_parser("exploration-stop", help="check whether task capsule has enough information to stop broad exploration")
    stop.add_argument("file", nargs="?", default=".codex/next_issue.md")
    log = sub.add_parser("compact-log", help="keep failures/warnings plus bounded log tail")
    log.add_argument("file")
    log.add_argument("--max-lines", type=int, default=80)
    log.add_argument("--tail-lines", type=int, default=20)

    args = parser.parse_args(argv)
    if args.command == "profile":
        _print_json(repo_profile())
    elif args.command == "doc-index":
        _print_json(markdown_index(directory=args.directory))
    elif args.command == "structure-index":
        _print_json(python_structure_index())
    elif args.command == "role-map":
        _print_json(file_role_map())
    elif args.command == "truth-candidates":
        _print_json(source_of_truth_candidates())
    elif args.command == "search":
        _print_json(bounded_text_search(args.query, regex=args.regex, case_sensitive=args.case_sensitive, max_results=max(1, args.max_results)))
    elif args.command == "path-find":
        _print_json(bounded_path_find(args.pattern, max_results=max(1, args.max_results)))
    elif args.command == "remote-delta":
        _print_json(remote_delta(base=args.base, fetch=not args.no_fetch, excerpt_lines=max(0, args.excerpt_lines), fast_forward=args.ff))
    elif args.command == "compact-diff":
        _print_json(compact_diff(base=args.base))
    elif args.command == "validation-plan":
        paths = current_changed_paths(base=args.base)
        _print_json({"changed_files": paths, "plan": validation_plan(paths)})
    elif args.command == "policy-check":
        findings = architecture_findings()
        for finding in findings:
            print(f"{finding.severity[0].upper()} {finding.path}:{finding.line} {finding.rule} {finding.message}")
        errors = sum(item.severity == "error" for item in findings)
        warnings = sum(item.severity == "warning" for item in findings)
        print(f"policy-check: {errors} error(s), {warnings} warning(s)")
        if args.fail_on == "error" and errors:
            return 1
        if args.fail_on == "warning" and (errors or warnings):
            return 1
    elif args.command == "context-pack":
        output = ROOT / ".codex" / "context_pack.md"
        output.parent.mkdir(exist_ok=True)
        output.write_text(build_context_pack(base=args.base), encoding="utf-8")
        print(output.relative_to(ROOT).as_posix())
    elif args.command == "exploration-stop":
        path = ROOT / args.file
        if not path.exists():
            _print_json({"ready": False, "missing": ["task capsule"], "file": args.file})
            return 1
        status = exploration_status(path.read_text(encoding="utf-8"))
        _print_json(status)
        return 0 if status["ready"] else 1
    elif args.command == "compact-log":
        path = Path(args.file)
        if not path.is_absolute():
            path = ROOT / path
        _print_json(compact_log(path.read_text(encoding="utf-8", errors="replace"), max_lines=max(1, args.max_lines), tail_lines=max(0, args.tail_lines)))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
