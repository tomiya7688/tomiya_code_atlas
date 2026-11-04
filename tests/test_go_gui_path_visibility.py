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
