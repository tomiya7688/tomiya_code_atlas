import re
from pathlib import Path


# {
#   責務: [test_gui_path_labels_wrap_without_truncation: GUIが選択済み入力・出力pathの全文を折り返して表示するstyleを保つことを検証する,],
# }
def test_gui_path_labels_wrap_without_truncation() -> None:
    stylesheet = (
        Path(__file__).resolve().parents[1] / "go/internal/ui/web/app.css"
    ).read_text(encoding="utf-8")
    path_rule = re.search(r"(?ms)^\.path\s*\{([^}]*)\}", stylesheet)

    assert path_rule is not None
    declarations = {
        name.strip(): value.strip()
        for declaration in path_rule.group(1).split(";")
        if ":" in declaration
        for name, value in (declaration.split(":", maxsplit=1),)
    }

    assert declarations.get("overflow-wrap") == "anywhere"
    assert declarations.get("white-space") == "normal"
    assert declarations.get("text-overflow") != "ellipsis"
    assert declarations.get("overflow") != "hidden"


# {
#   責務: [test_gui_controls_expose_names_and_live_status: GUIの主要操作名と解析状態をHTMLアクセシビリティtreeへ公開する契約を守る,],
# }
def test_gui_controls_expose_names_and_live_status() -> None:
    markup = (
        Path(__file__).resolve().parents[1] / "go/internal/ui/web/index.html"
    ).read_text(encoding="utf-8")

    for control_id in ("choose-source", "choose-output", "run-analysis"):
        assert f'id="{control_id}"' in markup
    for visible_name in (
        "ファイル",
        "プロジェクトフォルダー",
        "呼び出し図",
        "クラス図",
        "保存先を選ぶ",
        "解析して図を作る",
    ):
        assert visible_name in markup
    assert 'role="status" aria-live="polite"' in markup


# {
#   責務: [test_gui_rejects_incomplete_requests_before_analysis: GUIが未選択の解析要求を止め、入力中の二重起動と結果pathの欠損を安全に扱うことを確認する,],
# }
def test_gui_rejects_incomplete_requests_before_analysis() -> None:
    script = (
        Path(__file__).resolve().parents[1] / "go/internal/ui/web/app.js"
    ).read_text(encoding="utf-8")

    assert "if (analyzeButton.disabled) return;" in script
    assert "if (!selected.source)" in script
    assert "if (operations.length === 0)" in script
    assert "if (!selected.output)" in script
    assert "Array.isArray(response?.outputPaths)" in script
    assert 'if (!["o", "p", "r"].includes(key)) return;' in script
    assert "event.preventDefault();" in script
    assert 'if (key === "r" && !analyzeButton.disabled)' in script
