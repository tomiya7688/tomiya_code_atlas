"""Language-specific parsing adapters and backend contracts."""

from .backend import (
    PARSER_BACKEND_CONTRACT_VERSION,
    ParserBackend,
    ParserBackendDescriptor,
    ParserBackendError,
    ParserBackendFailure,
    ParserBackendFailureKind,
    ParserBackendKind,
    normalize_backend_exception,
)
from .base import LanguageAdapter
from .csharp import CSharpAdapter
from .csharp_backend import CSharpRoslynBackend
from .cpp import CppAdapter
from .gdscript import GDScriptAdapter
from .gdscript_backend import GDScriptTreeSitterBackend
from .gdscript_project import FilesystemGDScriptProjectResolver, GDScriptProjectResolver
from .go import GoAdapter
from .java import JavaAdapter
from .java_backend import JavaParserSymbolSolverBackend
from .python import PythonLanguageAdapter
from .python_backend import PythonStdlibBackend
from .python_comments_adapter import PythonAdapter

__all__ = [
    "CSharpAdapter",
    "CSharpRoslynBackend",
    "CppAdapter",
    "FilesystemGDScriptProjectResolver",
    "GDScriptAdapter",
    "GDScriptProjectResolver",
    "GDScriptTreeSitterBackend",
    "GoAdapter",
    "JavaAdapter",
    "JavaParserSymbolSolverBackend",
    "LanguageAdapter",
    "PARSER_BACKEND_CONTRACT_VERSION",
    "ParserBackend",
    "ParserBackendDescriptor",
    "ParserBackendError",
    "ParserBackendFailure",
    "ParserBackendFailureKind",
    "ParserBackendKind",
    "PythonAdapter",
    "PythonLanguageAdapter",
    "PythonStdlibBackend",
    "normalize_backend_exception",
]
