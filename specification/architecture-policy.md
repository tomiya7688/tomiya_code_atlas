# Tomiya Code Atlas Architecture Policy

This document contains normative project rules. Explanatory background lives in `docs/`.

## Rule strength

- **Required**: violation is not acceptable unless an explicit scoped exception exists.
- **Recommended**: default choice; deviation needs a concrete reason when it affects architecture or maintenance.
- **Advisory**: review signal, not an automatic violation.

Automated checks must distinguish confirmed violations from warnings/review candidates.

## Required: language and IR boundaries

1. Language-specific parser / AST / semantic-library types must not leak above the language-adapter boundary.
2. Shared IR / models must remain language-neutral enough for another adapter to produce equivalent information.
3. Analyzer / evaluator logic must not generate Mermaid, PlantUML, or other renderer syntax directly.
4. Renderer code must not parse source languages or depend on language adapters.
5. External parser libraries may be used inside adapters, but replacing one must not require rewriting unrelated generators/evaluators/renderers.
6. Machine-obtainable structural information should use deterministic analysis before LLM inference where practical.

## Required: UPD application boundary

Tomiya applies the UI / Process / Data separation at application level.

1. UI handles user input, presentation, launch flow, and display decisions.
2. Process handles orchestration and language-independent analysis/generation/evaluation flow.
3. Data handles source/config/file/external-data access and persistence details.
4. UI must not directly depend on Data implementation details.
5. Process must not know UI presentation details or storage-format details.
6. Data must not make UI or analysis-domain decisions.
7. A Commander selects/routs work; real analysis, transformation, I/O, rendering, or calculation must live in Processing/services.
8. A Messenger carries requests/results across a boundary; it must not select domain algorithms or perform real processing.

Equivalent explicit boundaries may be used without literally naming every module `Commander` or `Messenger`; the responsibility rule is normative, the naming convention is not.

## Required: information responsibilities

1. `README.md` is a human-facing overview and entry point, not the full specification.
2. `AI_CONTEXT.md` is the compact AI routing index, not a duplicate specification.
3. `docs/current_state.md` describes current capabilities / blockers only; it is not a changelog.
4. `docs/` explains architecture and feature design.
5. `specification/` contains normative project rules.
6. GitHub Issues are the source of truth for requested work, priority, discussion, and incomplete tasks.
7. Source and tests are the source of truth for implemented behavior.
8. Generated Context Packs, indexes, diagrams, and reports are derived artifacts and must not silently become authoritative specifications.

## Required: task / context discipline

1. Work should normally be anchored to one Issue; `1 Issue ~= 1 PR` is the default.
2. Search / metadata / indexes should narrow the working set before broad file reading.
3. Stop broad exploration when Goal, Required constraints, Acceptance, and the working set are sufficient.
4. Do not mix unrelated refactors into the active Issue.
5. Full diff, full logs, all Issues, and all docs must not be loaded by default merely for completeness.
6. A compact summary must preserve a path/ID back to its source of truth.
7. Unknown or unvalidated areas must be reported as `Unverified` rather than hidden by speculative broad reading.
8. Remote changes should be inspected as a compact delta before broad re-reading when concurrent edits are possible.

## Required: validation evidence

1. Validation must inspect the behavior/property claimed by the change; `0 tests` or empty scans are not sufficient evidence.
2. Start with targeted evidence and expand only as required by the change type / completion gate.
3. GUI/visual acceptance cannot be inferred solely from unit tests when visual correctness is part of Acceptance.
4. Package/distribution changes require artifact-level validation when source-only checks cannot prove the packaged result.
5. Successful logs should stay compact; failure logs may expand only around relevant evidence.
6. CI/policy automation must not convert uncertain semantic design judgments into hard errors without sufficient confidence.

## Recommended

- Reuse one structural analysis result for multiple diagrams/evaluators instead of reparsing.
- Prefer bounded graph traversal around target symbols.
- Use fan-in / fan-out / cycle information to prioritize impact review, not as automatic quality verdicts.
- Keep Responsibility Map entries current with architectural ownership changes.
- Use disposable temporary workspaces for validation that generates files.
- Prefer reproducible transformations and dry-run modes for bulk changes.
- Keep messages/results small, explicit, and framework-neutral across boundaries.
- Keep error conversion at the owning boundary instead of leaking framework/storage exceptions across layers.

## Advisory review signals

- A file/module responsibility requires several unrelated clauses to explain.
- Commander/Messenger code contains loops, calculations, direct file/network/database calls, or format conversion.
- Generator code directly imports a language adapter.
- Renderer code directly consumes low-level analysis objects rather than a stable logical output contract.
- A Context Pack keeps growing instead of linking to source-of-truth files.

These are review signals; existing transitional code may legitimately trigger warnings while migration work is tracked.

## Exception record

A Required-rule exception must record, in the Issue/PR or a dedicated exception document:

- rule
- reason
- scope
- mitigation / alternative
- removal condition or future review point
- source-of-truth reference

An exception applies only to the stated scope and must not silently become a general policy change.
