"""JavaParser + JavaSymbolSolver backend integration."""

from __future__ import annotations

from pathlib import Path

from Src.data.backend_assets import resolve_backend_asset
from Src.languages.backend import ParserBackendDescriptor, ParserBackendFailure, ParserBackendFailureKind, ParserBackendError, ParserBackendKind
from Src.languages.helper_backend import JsonHelperBackend


class JavaParserSymbolSolverBackend(JsonHelperBackend):
    """Run the bundled Java helper with its private runtime."""

    descriptor = ParserBackendDescriptor(
        backend_id="java-javaparser-symbol-solver-helper",
        language="java",
        kind=ParserBackendKind.HELPER,
    )
    helper_relative_path = "java/kadoka-java-backend.jar"

    def helper_command(self, helper: Path) -> list[str]:
        runtime = resolve_backend_asset("java/runtime/bin/java.exe", self._app_root)
        if not runtime.is_file():
            raise ParserBackendError(
                ParserBackendFailure(
                    ParserBackendFailureKind.FAILURE,
                    f"Parser backend runtime not found: {runtime}",
                    self.descriptor.backend_id,
                )
            )
        return [str(runtime), "-jar", str(helper)]
