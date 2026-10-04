from __future__ import annotations

import json
from pathlib import Path

import pytest

pytest.importorskip("clang")

from backends.cpp.main import handle_request  # noqa: E402
from Src.languages import CppClangBackend, ParserBackendKind  # noqa: E402


FIXTURE_ROOT = Path("tests/fixtures/backend_conformance")


def _request(source: str, path: str | None = None) -> dict[str, object]:
    request: dict[str, object] = {
        "contract_version": "1",
        "request_id": "cpp-parser-test",
        "operation": "parse",
        "language": "cpp",
        "source": source,
    }
    if path is not None:
        request["path"] = path
    return request


def test_cpp_helper_parses_complete_translation_unit_and_normalizes_common_ir() -> None:
    path = (FIXTURE_ROOT / "cpp.cpp").resolve()
    response = handle_request(_request(path.read_text(encoding="utf-8"), str(path)))

    assert response["ok"] is True
    assert response["request_id"] == "cpp-parser-test"
    ir = response["ir"]
    assert ir["schema_version"] == "1"
    assert ir["language"] == "cpp"
    entities = ir["entities"]
    namespaces = [item for item in entities if item["kind"] == "namespace"]
    assert any(item["name"] == "tomiya" and item["line"] == 5 for item in namespaces)
    worker = next(item for item in entities if item["kind"] == "class" and item["name"] == "Worker")
    assert worker["parent"] == "tomiya"
    assert worker["bases"] == ("BaseWorker",)
    assert worker["type_parameters"] == ("T",)
    methods = [item for item in entities if item["kind"] == "method" and item["name"] == "helper"]
    assert len(methods) == 2
    run = next(item for item in entities if item["kind"] == "method" and item["name"] == "run")
    assert run["call_sequence"].count("helper") == 2
    assert run["line"] == 11
    assert "support.hpp" in ir["imports"]
    assert ir["diagnostics"] == []


def test_cpp_missing_sdk_header_is_a_diagnostic_and_keeps_known_entities() -> None:
    source = "#include <missing_vendor/sdk.hpp>\nstruct VisibleType {};\n"
    response = handle_request(_request(source))

    assert response["ok"] is True
    assert any(item["name"] == "VisibleType" for item in response["ir"]["entities"])
    diagnostic = response["ir"]["diagnostics"][0]
    assert diagnostic["kind"] == "missing_header"
    assert diagnostic["line"] == 1


def test_cpp_syntax_error_returns_normalized_failure_without_partial_ir() -> None:
    response = handle_request(_request("struct Broken { void run( };\n"))

    assert response["ok"] is False
    assert response["error"]["kind"] == "unsupported_syntax"
    assert response["error"]["backend_id"] == "cpp-libclang-helper"
    assert "ir" not in response


def test_cpp_compile_database_arguments_are_applied(tmp_path: Path) -> None:
    source_path = tmp_path / "configured.cpp"
    source = "#ifdef CPP_FROM_COMPILE_DB\nstruct ConfiguredType {};\n#endif\n"
    source_path.write_text(source, encoding="utf-8")
    database = [
        {
            "directory": str(tmp_path),
            "file": str(source_path),
            "arguments": [
                "clang++",
                "-std=c++20",
                "-DCPP_FROM_COMPILE_DB=1",
                "-c",
                str(source_path),
                "-o",
                "configured.o",
            ],
        }
    ]
    (tmp_path / "compile_commands.json").write_text(
        json.dumps(database), encoding="utf-8"
    )

    response = handle_request(_request(source, str(source_path)))

    assert response["ok"] is True
    assert any(item["name"] == "ConfiguredType" for item in response["ir"]["entities"])


def test_cpp_adapter_descriptor_uses_bundled_clang_helper() -> None:
    assert CppClangBackend.descriptor.backend_id == "cpp-libclang-helper"
    assert CppClangBackend.descriptor.language == "cpp"
    assert CppClangBackend.descriptor.kind is ParserBackendKind.HELPER
