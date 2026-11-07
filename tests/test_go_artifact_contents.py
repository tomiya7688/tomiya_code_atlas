import json
from pathlib import Path

import yaml


# {
#   責務: [
#     test_windows_distribution_is_attested_after_smoke_checks: one-dir配布ZIP、checksum、attestationを全smoke成功後に保存することを検証する,
#   ],
# }
def test_windows_distribution_is_attested_after_smoke_checks() -> None:
    workflow = yaml.safe_load(
        (
            Path(__file__).resolve().parents[1] / ".github/workflows/go-exe.yml"
        ).read_text(encoding="utf-8")
    )
    steps = workflow["jobs"]["build-windows-exe"]["steps"]
    upload_index = next(
        index
        for index, step in enumerate(steps)
        if step.get("with", {}).get("name") == "tomiya-code-atlas-go-windows-x64"
    )
    package_index = next(
        index
        for index, step in enumerate(steps)
        if step.get("name") == "Windows one-dir配布ZIPとSHA-256 manifestを作成"
    )
    attestation_index = next(
        index
        for index, step in enumerate(steps)
        if step.get("uses") == "actions/attest@v4"
    )
    verification_index = next(
        index
        for index, step in enumerate(steps)
        if step.get("name") == "配布ZIPのprovenance attestationを検証"
    )
    gui_smoke_index = next(
        index
        for index, step in enumerate(steps)
        if step.get("name") == "Go GUIの配布起動をsmoke test"
    )
    distribution_e2e_index = next(
        index
        for index, step in enumerate(steps)
        if step.get("name") == "one-dir配布物のE2E検証"
    )
    artifact_paths = {
        line.strip()
        for line in steps[upload_index]["with"]["path"].splitlines()
        if line.strip()
    }
    assert artifact_paths == {
        ".build/release/tomiya-code-atlas-windows-x64.zip",
        ".build/release/SHA256SUMS.txt",
    }
    assert (
        distribution_e2e_index
        < package_index
        < attestation_index
        < verification_index
        < upload_index
    )
    assert gui_smoke_index < distribution_e2e_index
    assert steps[package_index]["run"] == "python tools/package_windows_distribution.py"
    assert steps[package_index]["env"] == {"PYTHONIOENCODING": "utf-8"}
    assert steps[attestation_index]["if"] == "github.event_name != 'pull_request'"
    assert steps[verification_index]["if"] == "github.event_name != 'pull_request'"
    assert steps[upload_index]["if"] == "github.event_name != 'pull_request'"
    assert "gh attestation verify" in steps[verification_index]["run"]
    assert "--signer-workflow" in steps[verification_index]["run"]
    assert steps[verification_index]["env"] == {"GH_TOKEN": "${{ github.token }}"}
    job_permissions = workflow["jobs"]["build-windows-exe"]["permissions"]
    assert job_permissions == {
        "contents": "read",
        "id-token": "write",
        "attestations": "write",
    }
    assert steps[attestation_index]["with"]["subject-checksums"] == (
        ".build/release/ATTESTED-SUBJECTS.txt"
    )

    build_script = (
        Path(__file__).resolve().parents[1] / "scripts/build/build_go.bat"
    ).read_text(encoding="utf-8")
    assert "generate_app_icon.ps1" in build_script
    assert '".build\\dist\\assets\\atlas-kun.ico"' in build_script
    assert '".build\\dist\\assets\\atlas-kun-256.png"' in build_script
    assert '".build\\dist\\assets\\atlas-kun-64.png"' in build_script
    assert '".build\\dist\\assets\\LICENSE.md"' in build_script

    # manifestが全言語を登録し、配布folder全体をZIP化する処理があることを確認する。
    manifest_path = (
        Path(__file__).resolve().parents[1] / "parser-backends/manifest.json"
    )
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    assert {"python", "gdscript", "csharp", "cpp", "java", "go"} <= {
        backend["language"] for backend in manifest["backends"]
    }
    for backend in manifest["backends"]:
        assert backend["path"], (
            f"{backend['backend_id']}: parser helper pathがありません"
        )
        for license_path in backend.get("licenses", []):
            assert license_path
        if backend["language"] == "java":
            assert backend.get("runtime", "").startswith("java/"), (
                "Java runtimeの配布先が不正です"
            )
