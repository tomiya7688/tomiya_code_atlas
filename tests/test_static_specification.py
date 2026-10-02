from Src.generators.static_specification import build_static_specification
from Src.analyzers.call_sequence import resolve_call_sequences
from Src.languages.python import PythonLanguageAdapter
from Src.renderers.static_specification_markdown import render_static_specification


def test_static_specification_lists_signatures_inheritance_and_local_calls() -> None:
    module = PythonLanguageAdapter().parse(
        '"""計算処理の概要。"""\n'
        "class Base: pass\n"
        "class Worker(Base):\n"
        "    def run(self, value: int) -> str:\n"
        "        return self.format(value)\n"
        "    def format(self, value: int) -> str:\n"
        "        return str(value)\n"
    )

    document = build_static_specification(
        module,
        source_name="sample.py",
        resolved_calls=resolve_call_sequences(module),
    )
    rendered = render_static_specification(document)

    assert document.module_docstring == "計算処理の概要。"
    assert "class: `Worker`" in rendered
    assert "継承・実装: `Base`" in rendered
    assert "run(value: int) -> str" in rendered
    assert "同一ファイル内で解決した呼び出し先: `Worker.format`" in rendered
    assert "呼び出し元: `Worker.run`" in rendered
    assert "定義位置: 4行目" in rendered


def test_static_specification_handles_empty_module() -> None:
    module = PythonLanguageAdapter().parse('"""空のファイル。"""\n')
    rendered = render_static_specification(
        build_static_specification(
            module,
            source_name="empty.py",
            resolved_calls=resolve_call_sequences(module),
        )
    )

    assert "空のファイル。" in rendered
    assert "宣言されたクラス、関数、メソッドはありません。" in rendered
