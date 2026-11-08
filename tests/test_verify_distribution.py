from pathlib import Path

from tools.verify_distribution import _environment


def test_distribution_appdata_stays_inside_isolated_workspace(
    tmp_path: Path, monkeypatch
) -> None:
    # Windows版GoはAPPDATAを必要とするため、実ユーザー領域でなく検証用workspaceを指定する。
    monkeypatch.setenv("APPDATA", r"C:\Users\example\AppData\Roaming")

    environment = _environment(tmp_path)

    assert Path(environment["APPDATA"]) == tmp_path / "app-data"
    assert environment["TEMP"] == str(tmp_path)
    assert environment["TMP"] == str(tmp_path)
