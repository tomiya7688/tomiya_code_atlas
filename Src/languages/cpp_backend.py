"""Clang LibTooling-backed C++ parser implementation."""

from __future__ import annotations

from pathlib import Path

from Src.languages.backend import ParserBackendDescriptor, ParserBackendKind
from Src.languages.helper_backend import JsonHelperBackend


class CppClangToolingBackend(JsonHelperBackend):
    """Use the bundled Clang semantic helper behind Common IR."""

    descriptor = ParserBackendDescriptor(
        backend_id="cpp-clang-tooling-helper",
        language="cpp",
        kind=ParserBackendKind.HELPER,
    )
    helper_relative_path = "cpp/kadoka-cpp-backend.exe"

    def __init__(
        self,
        *,
        app_root: Path | None = None,
        timeout_seconds: float = 30.0,
    ) -> None:
        super().__init__(app_root=app_root, timeout_seconds=timeout_seconds)
