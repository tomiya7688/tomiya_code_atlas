# Manual Acceptance Record

Use one copy of this record for each release candidate tested under [Issue #165](https://github.com/tomiya7688/tomiya_code_atlas/issues/165). The Issue is the authoritative test specification; record observations here and link the completed record from the Issue before release sign-off.

Do not treat unchecked or untested items as passing. If a check does not apply, explain why.

## Candidate and environment

- Test date:
- Tester:
- Release version / tag:
- Source commit SHA:
- Downloaded artifact name:
- Artifact SHA-256:
- Windows edition and version:
- Hardware / available memory:
- Clean-environment setup:
- Configuration used:

## Results

Use `Not tested`, `Pass`, `Issue found`, or `Not applicable` in the result column. Add a short observation for every result.

### First launch and GUI

| Check | Result | Observation |
| --- | --- | --- |
| First launch in the recorded environment; window appears without an error | Not tested | |
| Input or project selection and navigation through the main features | Not tested | |
| Progress, completion, errors, output location, and recovery are understandable | Not tested | |
| Repeated operation and rerun after changing an input | Not tested | |

### Languages and published features

Record the actual project or sample used. For each language, note which applicable features were run; use the full feature list in Issue #165.

| Language / input | Features exercised | Result | Observation |
| --- | --- | --- | --- |
| Python | | Not tested | |
| GDScript | | Not tested | |
| C# (Unity / .NET) | | Not tested | |
| C++ | | Not tested | |
| Java | | Not tested | |
| Go | | Not tested | |

### Output readability and README

| Check | Result | Observation |
| --- | --- | --- |
| Mermaid and PlantUML output is readable and useful | Not tested | |
| Tables, filenames, and output folder layout are understandable | Not tested | |
| README installation, launch, operation, and CLI steps work as written | Not tested | |

### CLI and error handling

| Check | Result | Observation |
| --- | --- | --- |
| Help, version, subcommands, renderer selection, and output options | Not tested | |
| Normal user mistakes and malformed / empty input | Not tested | |
| Missing backend, unwritable output, and large input behavior | Not tested | |
| Errors are actionable and the application can recover where expected | Not tested | |

## Findings and retests

For each issue, record the GitHub issue number, severity, whether it blocks v1.0.0, and the retest result after a fix.

| Finding / issue | Release blocker? | Retest candidate SHA | Retest result |
| --- | --- | --- | --- |
| | | | |

## Sign-off

- [ ] All applicable Issue #165 checks were performed on the recorded release candidate.
- [ ] Every release-blocking finding is fixed and retested on the final candidate.
- [ ] The completed record is linked from Issue #165.

Tester notes:

```text

```
