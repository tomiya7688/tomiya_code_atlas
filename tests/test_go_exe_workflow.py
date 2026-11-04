from pathlib import Path

import yaml


# {
#   責務: [test_go_workflow_checks_formatter_changes_and_limits_token_permissions: formatter設定変更でCIを起動し、workflow tokenを読み取り専用に制限することを検証する,],
# }
def test_go_workflow_checks_formatter_changes_and_limits_token_permissions() -> None:
    workflow_path = Path(__file__).resolve().parents[1] / ".github/workflows/go-exe.yml"
    workflow = yaml.load(
        workflow_path.read_text(encoding="utf-8"), Loader=yaml.BaseLoader
    )

    assert workflow["permissions"] == {"contents": "read"}
    for event_name in ("pull_request", "push"):
        paths = workflow["on"][event_name]["paths"]
        assert "tools/web-format/**" in paths
        assert "tools/verify_distribution.py" in paths
