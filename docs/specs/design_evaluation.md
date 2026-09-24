# Design Evaluation

## Goal
Evaluate software design quality using structural information produced by Tomiya Code Atlas.

A core project insight is that difficult-to-read generated diagrams can themselves reveal design problems. Evaluation should therefore use measurable graph/structure properties rather than visual appearance alone.

## Candidate signals
- Coupling and dependency concentration
- Fan-in / fan-out
- Cyclic dependencies
- Class/package responsibility spread
- Oversized hubs or god objects
- Call-graph complexity
- Excessively tangled diagram clusters
- Isolation of classes used by only one subsystem
- Highly shared infrastructure classes

## Rules
- Keep raw metrics separate from judgments/scores.
- Prefer explainable scoring: every warning should point to the structural evidence that caused it.
- Do not treat a single metric as definitive proof of bad design.
- Reuse analyzer output rather than implementing evaluator-specific parsers.

## Output
A set of metrics, findings, and optional scores with references to affected classes/modules/edges.
