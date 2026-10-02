from app import main


def test_static_spec_cli_prints_markdown(capsys) -> None:
    from pathlib import Path

    source = Path(__file__).parent / "fixtures" / "static_specification" / "sample.py"
    assert main(["static-spec", str(source)]) == 0
    content = capsys.readouterr().out

    assert "sample.py" in content
    assert "Small source used to check generated symbol references." in content
    assert "render(value: str) -> str" in content
    assert "同一ファイル内で解決した呼び出し先: `normalize`" in content
    assert "呼び出し元: `render`" in content


def test_static_spec_cli_recursively_documents_python_files(capsys) -> None:
    from pathlib import Path

    root = Path(__file__).parent / "fixtures" / "static_specification"
    assert main(["static-spec", str(root)]) == 0
    content = capsys.readouterr().out

    assert "sample.py" in content
    assert "nested/worker.py" in content
    assert "generated.py" not in content
    assert "method: `Worker.run`" in content
    assert "run(value: int) -> int" in content


def test_static_spec_cli_routes_markdown_to_output_path(monkeypatch) -> None:
    from pathlib import Path

    source = Path(__file__).parent / "fixtures" / "static_specification" / "sample.py"
    written: dict[str, object] = {}

    def capture_write(path: Path, content: str) -> None:
        written["path"] = path
        written["content"] = content

    monkeypatch.setattr("app.write_text", capture_write)

    assert main(["static-spec", str(source), "--output", "sample-spec.md"]) == 0
    assert written["path"] == Path("sample-spec.md")
    assert "同一ファイル内で解決した呼び出し先: `normalize`" in str(written["content"])
