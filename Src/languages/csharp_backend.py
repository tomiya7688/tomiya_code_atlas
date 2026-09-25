"""Roslyn-backed C# parser implementation."""

from __future__ import annotations

from pathlib import Path

from Src.languages.backend import ParserBackendDescriptor, ParserBackendKind
from Src.languages.helper_backend import JsonHelperBackend


class CSharpRoslynBackend(JsonHelperBackend):
    """Use the bundled self-contained Roslyn helper behind Common IR."""

    descriptor = ParserBackendDescriptor(
        backend_id="csharp-roslyn-helper",
        language="csharp",
        kind=ParserBackendKind.HELPER,
    )
    helper_relative_path = "csharp/tomiya-csharp-backend.exe"

    def __init__(
        self,
        *,
        app_root: Path | None = None,
        timeout_seconds: float = 30.0,
    ) -> None:
        super().__init__(app_root=app_root, timeout_seconds=timeout_seconds)
