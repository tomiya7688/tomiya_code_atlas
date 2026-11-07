import re
import shutil
import subprocess
from pathlib import Path

import pytest


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
#   責務: [test_gui_interactions_validate_and_dispatch_requests: 実際のGUIイベントで未入力拒否、再入防止、複数図の送信、結果path欠損を検証する,],
# }
def test_gui_interactions_validate_and_dispatch_requests() -> None:
    node = shutil.which("node")
    if node is None:
        pytest.skip("Node.js is required for GUI JavaScript behavior tests")

    test_script = Path(__file__).resolve().parent / "go_gui_frontend_behavior.test.cjs"
    result = subprocess.run(
        [node, str(test_script)],
        cwd=test_script.parent,
        capture_output=True,
        check=False,
        text=True,
        encoding="utf-8",
        errors="backslashreplace",
        timeout=30,
    )
    assert result.returncode == 0, f"{result.stdout}\n{result.stderr}"
