from __future__ import annotations

import sys

import pytest

from backends.python.main import handle_request


def _request(source: str) -> dict[str, str]:
    return {
        "contract_version": "1",
        "request_id": "python-grammar-test",
        "operation": "parse",
        "language": "python",
        "source": source,
        "path": "grammar.py",
    }


def test_contract_helper_parses_full_python_grammar_and_normalizes_ir() -> None:
    source = '''"""module docstring"""
from collections.abc import Iterable

def decorate(*, value: int):
    def apply(function):
        return function
    return apply

@decorate(value=1)
class Worker(Base, metaclass=Meta):
    async def run(self, /, values: Iterable[int], *, enabled: bool = True, **options) -> str:
        async with manager() as active:
            match values:
                case [first, *rest] if enabled and (count := len(rest)) >= 0:
                    return f"{first}:{[consume(item) for item in rest]!r}:{count}"
                case _:
                    return "empty"

    def generator(self):
        yield from (item for item in range(10) if item)
'''
    response = handle_request(_request(source))

    assert response["ok"] is True
    assert response["request_id"] == "python-grammar-test"
    assert response["ir"]["schema_version"] == "1"
    assert response["ir"]["language"] == "python"
    assert response["ir"]["module_docstring"] == "module docstring"
    entities = response["ir"]["entities"]
    worker = next(entity for entity in entities if entity["name"] == "Worker")
    run = next(entity for entity in entities if entity["name"] == "run")
    assert worker["decorators"] == ("decorate(value=1)",)
    assert run["is_async"] is True
    assert run["parameters"] == ("self", "values", "enabled", "**options")
    assert "consume" in run["call_sequence"]
    assert response["ir"]["imports"] == ("collections.abc",)


def test_contract_helper_reports_syntax_error_without_partial_ir() -> None:
    response = handle_request(_request("def broken(:\n    pass\n"))

    assert response["ok"] is False
    assert response["error"]["kind"] == "unsupported_syntax"
    assert response["error"]["backend_id"] == "python-stdlib-ast"
    assert "ir" not in response


def test_contract_helper_rejects_invalid_protocol_without_parsing() -> None:
    response = handle_request({"contract_version": "99", "source": "class Valid: pass"})

    assert response["ok"] is False
    assert response["error"]["kind"] == "protocol_error"
    assert "ir" not in response


@pytest.mark.skipif(sys.version_info < (3, 12), reason="PEP 695 syntax requires CPython 3.12+")
def test_contract_helper_parses_pep695_type_parameter_syntax() -> None:
    response = handle_request(
        _request("class Box[T]:\n    def get[U](self, value: U) -> T: ...\n")
    )

    assert response["ok"] is True
    box, get = response["ir"]["entities"]
    assert box["type_parameters"] == ("T",)
    assert get["type_parameters"] == ("U",)
