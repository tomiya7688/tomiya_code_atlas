"""Go standard-library parser and type-checker helper adapter."""

from __future__ import annotations

from Src.languages.backend import ParserBackendDescriptor, ParserBackendKind
from Src.languages.helper_backend import JsonHelperBackend


class GoStandardLibraryBackend(JsonHelperBackend):
    """Run the bundled Go AST/type-checker helper through Contract v1."""

    descriptor = ParserBackendDescriptor(
        backend_id="go-stdlib-types-helper",
        language="go",
        kind=ParserBackendKind.HELPER,
    )
    helper_relative_path = "go/tomiya-go-backend.exe"

