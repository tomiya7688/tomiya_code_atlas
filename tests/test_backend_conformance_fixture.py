from __future__ import annotations

import json
from dataclasses import asdict
from pathlib import Path

from Src.generators.call_graph import build_call_graph
from Src.languages.python_project import PythonProjectLanguageAdapter


FIXTURE_ROOT = Path("tests/fixtures/backend_conformance")
FORBIDDEN_BACKEND_KEYS = {
    "node_type",
    "syntax_kind",
    "roslyn_kind",
    "tree_sitter_type",
    "clang_cursor_kind",
    "token_type",
}


def _load(name: str) -> dict[str, object]:
    return json.loads((FIXTURE_ROOT / name).read_text(encoding="utf-8"))


def _walk_keys(value: object):
    if isinstance(value, dict):
        for key, child in value.items():
            yield str(key)
            yield from _walk_keys(child)
    elif isinstance(value, list):
        for child in value:
            yield from _walk_keys(child)


def test_conformance_manifest_covers_all_six_primary_languages() -> None:
    manifest = _load("manifest.json")
    languages = manifest["languages"]

    assert set(languages) == {"python", "gdscript", "csharp", "cpp", "java", "go"}
    assert set(manifest["required_cases"]) >= {
        "import",
        "class",
        "method",
        "call",
        "duplicate-call",
        "inheritance",
        "generic-probe",
        "async-probe",
        "nested-scope-probe",
    }
    for config in languages.values():
        assert (FIXTURE_ROOT / config["source"]).is_file()
        assert config["generic_probe"]
        assert config["async_probe"]
        assert config["nested_scope_probe"]


def test_golden_contract_contains_only_language_independent_semantics() -> None:
    golden = _load("expected_common_ir_v1.json")

    assert golden["contract_version"] == "1"
    assert golden["projection"] == "common-ir-semantic-subset"
    assert not (set(_walk_keys(golden)) & FORBIDDEN_BACKEND_KEYS)


def test_serialized_common_ir_fixture_covers_wire_and_logical_output_contract() -> None:
    fixture = _load("serialized_common_ir_v1.json")
    module = fixture["common_ir"]

    assert fixture["schema_version"] == "1"
    parsed = PythonProjectLanguageAdapter().parse(fixture["source"])
    assert json.loads(json.dumps(asdict(parsed))) == module
    assert fixture["diagnostic_example"]["line"] >= 1

    graph = fixture["logical_output"]["call_graph"]
    actual_graph = build_call_graph(parsed)
    assert graph["nodes"] == sorted(set(graph["nodes"]))
    assert graph["nodes"] == sorted(actual_graph.nodes)
    assert graph["edges"] == sorted(
        graph["edges"], key=lambda edge: (edge["caller"], edge["callee"], edge["call_type"])
    )
    assert graph["edges"] == [asdict(edge) for edge in actual_graph.edges]
    assert not (set(_walk_keys(fixture)) & FORBIDDEN_BACKEND_KEYS)


def test_go_and_csharp_helpers_share_protocol_conformance_cases() -> None:
    fixture = _load("helper_protocol_cases_v1.json")

    assert fixture["contract_version"] == "1"
    assert fixture["schema_version"] == "1"
    assert set(fixture["cases"]) == {"go", "csharp"}
    for language, case in fixture["cases"].items():
        request = case["request"]
        assert request["operation"] == "parse"
        assert request["language"] == language
        assert request["request_id"]
        assert request["source"]
        assert case["expected_entity_names"]


def test_python_fixture_matches_common_ir_semantic_golden() -> None:
    golden = _load("expected_common_ir_v1.json")
    source = (FIXTURE_ROOT / "python.py").read_text(encoding="utf-8")
    module = PythonProjectLanguageAdapter().parse(source)

    entities = {(entity.kind.value, entity.name, entity.parent): entity for entity in module.entities}
    for expected in golden["required_entities"]:
        key = (expected["kind"], expected["name"], expected.get("parent"))
        assert key in entities
        if bases := expected.get("bases"):
            assert set(bases) <= set(entities[key].bases)

    assert module.imports

    for expected in golden["required_call_counts"]:
        parent, name = expected["owner"].rsplit(".", 1)
        entity = entities[("method", name, parent)]
        assert entity.call_sequence.count(expected["target"]) == expected["count"]


def test_fixture_suite_exercises_generic_async_and_nested_scope_cases() -> None:
    manifest = _load("manifest.json")
    sources = {
        language: (FIXTURE_ROOT / config["source"]).read_text(encoding="utf-8")
        for language, config in manifest["languages"].items()
    }

    assert "Generic[T]" in sources["python"]
    assert "Array[Variant]" in sources["gdscript"]
    assert "Worker<T>" in sources["csharp"]
    assert "template <typename T>" in sources["cpp"]
    assert "Worker<T>" in sources["java"]
    assert "Worker[T any]" in sources["go"]

    assert "async def async_probe" in sources["python"]
    assert "await " in sources["gdscript"]
    assert "async Task<T> async_probe" in sources["csharp"]
    assert "future<T> async_probe" in sources["cpp"]
    assert "CompletableFuture<T> async_probe" in sources["java"]
    assert "go func" in sources["go"]

    assert "def nested(" in sources["python"]
    assert "var nested := func" in sources["gdscript"]
    assert "T nested(T value)" in sources["csharp"]
    assert "auto nested =" in sources["cpp"]
    assert "value -> value" in sources["java"]
    assert "nested := func" in sources["go"]
