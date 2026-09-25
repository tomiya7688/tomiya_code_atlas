# Class Responsibility Table

## Goal
Generate a compact table describing each class and its primary responsibility.

Example shape:

```text
Class,Responsibility
Tomiya,Holds Tomiya parameters
```

## Analysis
- Extract class/type declarations from the language-independent model.
- Infer responsibility from class name, members, methods, inheritance, dependencies, and available comments/docstrings.
- Keep deterministic structural facts separate from inferred natural-language responsibility descriptions.

## Output
Prefer a simple table representation that can be rendered as Markdown and exported in CSV-compatible form.
