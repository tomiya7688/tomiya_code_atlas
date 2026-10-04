from __future__ import annotations

import json
from pathlib import Path
import symtable

import pytest

from Src.languages import (
    ParserBackendError,
    ParserBackendFailureKind,
    ParserBackendKind,
    PythonStdlibBackend,
)


FIXTURE_ROOT = Path("tests/fixtures/backend_conformance")


def _golden() -> dict[str, object]:
    return json.loads(
        (FIXTURE_ROOT / "expected_common_ir_v1.json").read_text(encoding="utf-8")
    )


def test_selected_python_backend_has_stable_descriptor() -> None:
    descriptor = PythonStdlibBackend.descriptor

    assert descriptor.backend_id == "python-stdlib-ast"
    assert descriptor.language == "python"
    assert descriptor.kind is ParserBackendKind.IN_PROCESS


def test_selected_python_backend_matches_conformance_fixture() -> None:
    source = (FIXTURE_ROOT / "python.py").read_text(encoding="utf-8")
    module = PythonStdlibBackend().parse(source, str(FIXTURE_ROOT / "python.py"))
    golden = _golden()

    entities = {
        (entity.kind.value, entity.name, entity.parent): entity
        for entity in module.entities
    }
    for expected in golden["required_entities"]:
        key = (expected["kind"], expected["name"], expected.get("parent"))
        assert key in entities
        if bases := expected.get("bases"):
            assert set(bases) <= set(entities[key].bases)

    assert module.imports
    assert entities[("method", "async_probe", "Worker")].is_async is True

    for expected in golden["required_call_counts"]:
        parent, name = expected["owner"].rsplit(".", 1)
        entity = entities[("method", name, parent)]
        assert entity.call_sequence.count(expected["target"]) == expected["count"]


def test_stdlib_symtable_companion_sees_fixture_scopes() -> None:
    source = (FIXTURE_ROOT / "python.py").read_text(encoding="utf-8")
    table = symtable.symtable(source, "python.py", "exec")
    worker = next(child for child in table.get_children() if child.get_name() == "Worker")

    method_names = {child.get_name() for child in worker.get_children()}
    assert {"run", "helper", "async_probe", "nested_probe"} <= method_names


def test_selected_python_backend_normalizes_syntax_failure() -> None:
    with pytest.raises(ParserBackendError) as captured:
        PythonStdlibBackend().parse("def broken(:\n    pass\n", "broken.py")

    assert captured.value.failure.kind is ParserBackendFailureKind.UNSUPPORTED_SYNTAX
    assert captured.value.failure.backend_id == "python-stdlib-ast"
