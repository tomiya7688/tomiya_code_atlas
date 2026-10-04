"""Clang compiler-AST helper adapter for C++ translation units."""

from __future__ import annotations

from pathlib import Path

from Src.languages.backend import ParserBackendDescriptor, ParserBackendKind
from Src.languages.helper_backend import JsonHelperBackend


class CppClangBackend(JsonHelperBackend):
    """Use the bundled libclang parser and optional compile database."""

    descriptor = ParserBackendDescriptor(
        backend_id="cpp-libclang-helper",
        language="cpp",
        kind=ParserBackendKind.HELPER,
    )
    helper_relative_path = "cpp/tomiya-cpp-backend/tomiya-cpp-backend.exe"

    def __init__(
        self,
        *,
        app_root: Path | None = None,
        timeout_seconds: float = 60.0,
    ) -> None:
        super().__init__(app_root=app_root, timeout_seconds=timeout_seconds)
