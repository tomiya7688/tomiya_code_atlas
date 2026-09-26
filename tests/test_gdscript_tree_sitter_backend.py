from __future__ import annotations

import json
from pathlib import Path

from app import main
from Src.languages import (
    FilesystemGDScriptProjectResolver,
    GDScriptTreeSitterBackend,
    ParserBackendKind,
)


FIXTURE = Path("tests/fixtures/backend_conformance/gdscript.gd")


def test_gdscript_tree_sitter_backend_descriptor() -> None:
    descriptor = GDScriptTreeSitterBackend.descriptor

    assert descriptor.backend_id == "gdscript-tree-sitter"
    assert descriptor.language == "gdscript"
    assert descriptor.kind is ParserBackendKind.NATIVE


def test_gdscript_fixture_normalizes_to_common_ir() -> None:
    source = FIXTURE.read_text(encoding="utf-8")
    module = GDScriptTreeSitterBackend().parse(source, str(FIXTURE))

    entities = {
        (entity.kind.value, entity.name, entity.parent): entity
        for entity in module.entities
    }

    assert ("class", "FixtureWorker", None) in entities
    assert entities[("class", "FixtureWorker", None)].bases == ("RefCounted",)
    assert ("class", "BaseWorker", None) in entities
    assert ("class", "Worker", None) in entities

    run = entities[("method", "run", "Worker")]
    assert run.call_sequence.count("helper") == 2
    assert run.parameters == ("item",)

    assert ("method", "helper", "Worker") in entities
    assert ("method", "async_probe", "Worker") in entities
    assert ("method", "nested_probe", "Worker") in entities

    assert "res://support.gd" in module.imports
    assert "res://other_resource.tres" in module.imports

    assert len(module.signals) == 1
    assert module.signals[0].name == "completed"
    assert module.signals[0].owner == "FixtureWorker"
    assert module.signals[0].parameters == ("value",)

    assert module.diagnostics == []


def test_gdscript_backend_keeps_partial_ir_with_syntax_diagnostic() -> None:
    module = GDScriptTreeSitterBackend().parse(
        "class_name Broken\nfunc broken(:\n    pass\n",
        "broken.gd",
    )

    assert any(entity.name == "Broken" for entity in module.entities)
    assert module.diagnostics
    assert {item.kind for item in module.diagnostics} <= {
        "syntax_error",
        "missing_syntax",
    }


def test_gdscript_project_resolver_resolves_res_paths_and_class_names(tmp_path: Path) -> None:
    root = tmp_path / "game"
    scripts = root / "scripts"
    scripts.mkdir(parents=True)
    support = scripts / "support.gd"
    support.write_text("extends RefCounted\n", encoding="utf-8")

    resolver = FilesystemGDScriptProjectResolver(root)
    resolver.register_class_name("Support", support)

    assert resolver.resolve_resource("res://scripts/support.gd") == support.resolve()
    assert resolver.resolve_class_name("Support") == support.resolve()
    assert resolver.resolve_resource("res://../outside.gd") is None


def test_backend_manifest_registers_gdscript_tree_sitter() -> None:
    manifest = json.loads(Path("backends/manifest.json").read_text(encoding="utf-8"))
    backend = next(
        item
        for item in manifest["backends"]
        if item["backend_id"] == "gdscript-tree-sitter"
    )

    assert backend["language"] == "gdscript"
    assert backend["kind"] == "native"
    assert backend["contract_version"] == "1"
    assert "tree-sitter-gdscript" in backend["packages"]


def test_backend_smoke_cli_loads_gdscript_native_parser(capsys) -> None:
    assert main(["backend-smoke", "gdscript"]) == 0
    assert "gdscript-tree-sitter" in capsys.readouterr().out


def test_backend_smoke_cli_parses_supplied_gdscript_fixture(tmp_path, capsys) -> None:
    source = tmp_path / "fixture.gd"
    source.write_text("class_name LoadConfig\nextends RefCounted\nfunc load():\n    return true\n", encoding="utf-8")

    assert main(["backend-smoke", "gdscript", "--source", str(source)]) == 0
    assert "gdscript-tree-sitter" in capsys.readouterr().out
