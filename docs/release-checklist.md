# v1.0.0 Release Checklist

- [ ] The release tag matches `version` in `pyproject.toml`.
- [ ] The package Build workflow passes its wheel and source archive install-and-run checks.
- [ ] The Python EXE workflow passes its packaged CLI, backend, and GUI startup checks.
- [ ] The Release workflow passes `verify_build.bat` for the exact wheel, source archive, and Windows one-dir app directory attached to the release.
- [ ] Manual acceptance in Issue #165 is complete before publishing v1.0.0.

The Release workflow creates a GitHub Release only after it builds and verifies the candidate artifacts from the tagged commit. It uploads those same package files and the Windows app directory archived after its E2E checks.