# UPD Commander Adoption

Tomiya Code Atlas adopts the useful responsibility rules from `upd-commander-base-design` without making that repository or its naming conventions a runtime dependency. Normative rules live in `specification/architecture-policy.md`; this document explains their mapping.

## Layer mapping

### UI
- GUI / CLI / launch flow
- user input and display settings
- result presentation / save intent

UI does not implement source analysis and does not directly own Data implementation details.

### Process
- application orchestration
- Analyzer / Generator / Evaluator / Renderer flow
- language-independent use of Common IR
- deciding which operation happens next

Process does not know concrete UI presentation or storage-format details.

### Data
- target source/config acquisition
- file I/O, cache, persistence
- external data access

Data does not make UI or analysis-domain decisions.

## Commander
Commander answers **what should run next**.

Allowed:
- receive a request/result
- select an appropriate Processing/service
- ask a Messenger to cross a boundary
- forward results to the next step

Not a Commander responsibility:
- AST parsing
- graph algorithms
- diagram generation
- renderer syntax generation
- file/database/network I/O
- substantial calculations/transforms

A loop/calculation/I/O call inside Commander code is therefore a review signal and may be a policy finding.

## Messenger
Messenger carries requests/results across an application boundary.

Allowed:
- send/receive a boundary message
- translate only the transport/boundary representation required by the contract
- forward the received message to its own Commander/application entry

Not a Messenger responsibility:
- choose domain algorithms
- analyze/evaluate/render
- persist data
- perform business decisions

## Processing
Real work belongs in Processing/services or the existing focused modules: language adapters, analyzers, generators, renderers, evaluators, and data-access implementations. The project does not need to rename every existing module to `Processing`; responsibility is more important than literal class names.

## Dependency flow

Conceptually:

```text
UI <-> Process <-> Data
```

Disallowed shortcuts include direct UI -> Data implementation access and direct calls into another layer's internal Processing when an application boundary should mediate the interaction.

Existing Code Atlas pipeline rules remain authoritative:

```text
Source
 -> Language Adapter / Parser
 -> Common IR / shared models
 -> Analyzer / Generator / Evaluator
 -> Logical Output
 -> Renderer
```

UPD is the application-level boundary around that pipeline; it does not replace language/IR/renderer separation.

## Message contracts
Cross-boundary messages should be small, explicit, and independent of UI frameworks, database clients, parser-library nodes, or renderer-specific syntax. Prefer stable values/records that can be tested without constructing the framework on the other side.

Do not pass a WPF/Tk/Godot control, DB connection, Roslyn node, Python `ast` node, etc. through a shared application contract merely for convenience.

## Error handling
- Detect/handle errors at the layer that owns the operation.
- Convert framework/storage/parser-specific errors at the boundary when higher layers should not know those details.
- Preserve useful cause/context for diagnostics.
- Do not silently swallow errors just to keep a Commander/Messenger path simple.
- UI chooses presentation of an error; Process/Data should return domain/application-relevant failure information rather than UI text formatting.

## Testing model
- Processing / analyzer / generator logic: direct unit/regression tests.
- Commander: orchestration tests proving the correct service/message path is selected.
- Messenger / boundary: contract/integration tests proving request/result transfer without domain logic leakage.
- Layer/import rules: `python tools/context_tool.py policy-check` for mechanically reliable checks.
- Semantic ownership: Responsibility Map + targeted architecture review; do not pretend it is fully machine-checkable.

## Policy strength and exceptions
`specification/architecture-policy.md` separates Required, Recommended, and Advisory rules. Automated checks distinguish confirmed errors from warnings.

A necessary exception is scoped rather than turning the architecture vague. Record the rule, reason, scope, mitigation, and removal/review condition in the relevant Issue/PR.

## Review checklist
- Does UI contain analysis or storage implementation logic?
- Does Process depend on display formatting or storage format details?
- Does Data make UI/domain-analysis decisions?
- Has Commander/Messenger accumulated real processing?
- Are framework/parser-specific types crossing a stable boundary?
- Is Common IR/shared model code acquiring feature orchestration/evaluation/rendering behavior?
- Is a Renderer parsing source or importing a language adapter?
- Could the changed responsibility still be described as one short entry in `docs/responsibility_map.md`?

See also:
- `specification/architecture-policy.md`
- `docs/responsibility_map.md`
- `docs/project_operations.md`
