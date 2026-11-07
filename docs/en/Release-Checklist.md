# v1.0.0 Release Checklist

The Japanese source is the [v1.0.0 release checklist](../jp/%E3%83%AA%E3%83%AA%E3%83%BC%E3%82%B9%E7%A2%BA%E8%AA%8D%E8%A1%A8.md).

- [x] Inventory artifacts produced by the current workflows and their distribution paths.
- [ ] Release tag, Go CLI version, and GitHub Release artifacts show the same version.
- [ ] Issue #27 and all required Go migration issues are closed.
- [ ] The Go EXE workflow succeeds for the distributed CLI, all analysis features, GUI, and clean-environment output checks.
- [ ] GitHub Release artifacts include the Go Windows executable and do not require a Python runtime.
- [ ] Provenance attestations cover the distributed archive, manifests, and each bundled binary. CI and users verify the ZIP and manifests with GitHub CLI; the manifest records each bundled binary's SHA-256.
- [ ] Existing GitHub Releases and Actions artifacts are audited; unverified artifacts created before this workflow are labeled or replaced with rebuilt artifacts.
- [ ] Public Windows binaries have a trusted Authenticode signature and RFC 3161 timestamp, and their Publisher can be verified.
- [ ] Issue #165 manual acceptance is complete before v1.0.0 publication.

Record manual acceptance in the [manual acceptance record template](../jp/%E6%89%8B%E5%8B%95%E5%8F%97%E3%81%91%E5%85%A5%E3%82%8C%E8%A8%98%E9%8C%B2%E3%83%86%E3%83%B3%E3%83%97%E3%83%AC%E3%83%BC%E3%83%88.md) and attach or link the completed record from Issue #165.

Do not enable the Release workflow or publish v1.0.0 before the Go migration is complete. Add the tag-triggered Go distribution, verification, and publication workflow in the migration issue that owns the release pipeline.

## Artifact inventory

### Purpose and scope

This inventory distinguishes the artifacts generated and stored by repository CI and the Windows distribution workflow, and tells users how to verify downloaded distributions. On 2026-11-07, the GitHub Releases API and `.github/workflows/` plus the distribution package script on `origin/main` were checked. There were no GitHub Releases or Release assets.

The current workflows generate these artifacts:

| Artifact                                                                | Generated or stored at                                        | Distribution role                                                                                                        | Current integrity evidence                                                                                                                                                                                                 |
| ----------------------------------------------------------------------- | ------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Go CLI, language parser helpers, and bundled runtimes                   | Windows build `.build/dist/`                                  | Contents of the one-dir distribution; not released separately.                                                           | Binary files such as `.exe`, `.dll`, `.pyd`, `.jar`, `.so`, and `.dylib`, plus the Java runtime image, appear in `SHA256SUMS.txt` and receive individual GitHub Artifact Attestations. There is no Authenticode signature. |
| License text and other non-binary files, excluding checksum manifests   | Windows build `.build/dist/`                                  | Included in the one-dir distribution ZIP.                                                                                | Verified through the ZIP attestation, not listed in the binary manifest or individually attested.                                                                                                                          |
| Windows one-dir ZIP                                                     | `.build/release/tomiya-code-atlas-windows-x64.zip`            | User-downloadable distribution package                                                                                   | The ZIP has a SHA-256 manifest and a GitHub Artifact Attestation. There is no Authenticode signature.                                                                                                                      |
| SHA-256 manifests                                                       | `.build/dist/SHA256SUMS.txt`, `.build/release/SHA256SUMS.txt` | The dist manifest is inside the ZIP; the release manifest is in the final Actions artifact.                              | Both manifests are attestation subjects. Verify the attestation; do not rely on hashes alone as a trust signal.                                                                                                            |
| Attestation subject manifest                                            | `.build/release/ATTESTED-SUBJECTS.txt`                        | Input to the attestation job and included in the cross-job transfer artifact; not included in the final Actions artifact | Lists subject hashes. It is an input file for attestation verification, not signed provenance by itself.                                                                                                                   |
| `tomiya-code-atlas-windows-x64-attestation-input` Actions artifact      | Transfer from the build job to the attestation job            | Cross-job workflow transfer, not a Release asset; downloadable from GitHub Actions.                                      | The next job attests hashes of the bundled distribution subjects, but the outer transfer package is not an attestation subject. GitHub API exposes an artifact digest, but that does not prove its origin.                 |
| `tomiya-code-atlas-go-windows-x64` Actions artifact                     | Attestation job in `.github/workflows/go-exe.yml`             | Downloadable from the workflow run after verification succeeds                                                           | Verify the ZIP and release manifest attestations separately. The outer Actions artifact package is not an attestation subject. GitHub API exposes an artifact digest, but that does not prove its origin.                  |
| CI tests, self-analysis summary, and intermediate files under `.build/` | `.github/workflows/ci.yml`                                    | Not uploaded as user-facing artifacts by the current CI.                                                                 | Not part of the published distribution.                                                                                                                                                                                    |

The push workflow run associated with PR #270 saved these two Actions artifacts. Their GitHub API digests are SHA-256 checks for the downloaded packages; the digests are not workflow provenance attestations.

| Artifact                                          |            ID | GitHub API digest                                                         | Purpose                                                                      |
| ------------------------------------------------- | ------------: | ------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `tomiya-code-atlas-windows-x64-attestation-input` | `13569716553` | `sha256:e5024b665d730e33cca8f82cbb698ebf07f6fdbe46529e4c4853de37683fc4cc` | Cross-job transfer. Contains the ZIP, manifests, and distribution directory. |
| `tomiya-code-atlas-go-windows-x64`                | `13570215655` | `sha256:e29ba9a7852e292c31f4a871cc32e5ae7cdba2f4221ea956462ce52139ce2231` | User-facing Actions artifact after attestation verification.                 |

See [workflow run 42360603029](https://github.com/tomiya7688/tomiya_code_atlas/actions/runs/42360603029), [final artifact 13570215655](https://github.com/tomiya7688/tomiya_code_atlas/actions/runs/42360603029/artifacts/13570215655), and its [provenance attestation](https://github.com/tomiya7688/tomiya_code_atlas/attestations/64814883). The recorded expiry date for both Actions artifacts is 2027-02-05. These are Actions artifacts, not GitHub Release assets.

### Historical artifact audit status

No GitHub Releases or Release assets existed when checked through the GitHub Releases API. This environment could not retrieve a complete listing of historical Actions artifacts, so the historical artifact audit is not complete. Do not assume that artifacts created before Artifact Attestations have provenance or direct users to treat them as verified. Before retrieving or redistributing a historical artifact, check its workflow run retention and attestation individually. Stop using an unverified artifact or replace it with a verifiable build.

### User verification

Download `tomiya-code-atlas-go-windows-x64` from its Actions run and use the verification commands in the Japanese [README](../../README.md) or the English [Windows build guide](Building.md) on the ZIP and manifests. After extracting the ZIP, use `SHA256SUMS.txt` to compare the SHA-256 hash of each bundled binary. Each listed binary is also an attestation subject; verify its provenance with `gh attestation verify` using the same repository, signer workflow, source ref, and commit SHA as the ZIP command. GitHub CLI fetches up to 30 attestations by default ([official manual](https://cli.github.com/manual/gh_attestation_verify)). If the expected run falls outside that result window, repeat verification with a `--limit` large enough to include the target, while keeping the source ref and commit SHA checks. Do not run an artifact if attestation verification fails or a checksum differs. Attestation identifies build provenance and detects modification; it does not guarantee a Windows SmartScreen Publisher identity. Trusted Authenticode signing and an RFC 3161 timestamp remain incomplete until a trusted certificate is provisioned.
