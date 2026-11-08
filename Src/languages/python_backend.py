"""Selected Python parser backend behind the versioned backend contract."""

from __future__ import annotations

from Src.analyzers.ir import ModuleIR
from Src.languages.backend import (
    ParserBackendDescriptor,
    ParserBackendError,
    ParserBackendFailure,
    ParserBackendFailureKind,
    ParserBackendKind,
    normalize_backend_exception,
)
from Src.languages.python_timing import PythonTimingLanguageAdapter
from Src.languages.python_usecase import PythonUseCaseLanguageAdapter
from Src.languages.python_project import PythonProjectLanguageAdapter


class PythonStdlibBackend:
    """Use CPython's stdlib AST-based adapter as the primary Python backend."""

    descriptor = ParserBackendDescriptor(
        backend_id="python-stdlib-ast",
        language="python",
        kind=ParserBackendKind.IN_PROCESS,
    )

    def parse(self, source: str, path: str | None = None) -> ModuleIR:
        try:
            module = PythonProjectLanguageAdapter().parse(source)
            module.timing_flows = (
                PythonTimingLanguageAdapter().parse(source).timing_flows
            )
            module.input_events = (
                PythonUseCaseLanguageAdapter().parse(source).input_events
            )
            return module
        except SyntaxError as error:
            raise ParserBackendError(
                ParserBackendFailure(
                    ParserBackendFailureKind.UNSUPPORTED_SYNTAX,
                    str(error),
                    self.descriptor.backend_id,
                )
            ) from error
        except BaseException as error:
            raise normalize_backend_exception(self.descriptor, error) from error
