import json
from pathlib import Path

import yaml


# {
#   責務: [
#     test_windows_distribution_is_attested_after_smoke_checks: PR検証ジョブと署名ジョブの権限分離、ZIP生成・検証順を検証する,
#   ],
# }
def test_windows_distribution_is_attested_after_smoke_checks() -> None:
    workflow = yaml.safe_load(
        (
            Path(__file__).resolve().parents[1] / ".github/workflows/go-exe.yml"
        ).read_text(encoding="utf-8")
    )
    build_job = workflow["jobs"]["build-windows-exe"]
    build_steps = build_job["steps"]
    attest_job = workflow["jobs"]["attest-windows-distribution"]
    attest_steps = attest_job["steps"]
    intermediate_upload_index = next(
        index
        for index, step in enumerate(build_steps)
        if step.get("with", {}).get("name")
        == "tomiya-code-atlas-windows-x64-attestation-input"
    )
    intermediate_upload = build_steps[intermediate_upload_index]
    pr_upload_index = next(
        index
        for index, step in enumerate(build_steps)
        if step.get("name") == "PR検証用Windows配布artifactを保存"
    )
    pr_upload = build_steps[pr_upload_index]
    package_index = next(
        index
        for index, step in enumerate(build_steps)
        if step.get("name") == "Windows one-dir配布ZIPとSHA-256 manifestを作成"
    )
    attestation_index = next(
        index
        for index, step in enumerate(attest_steps)
        if step.get("uses") == "actions/attest@v4"
    )
    verification_index = next(
        index
        for index, step in enumerate(attest_steps)
        if step.get("name") == "配布ZIPとchecksum manifestのprovenanceを検証"
    )
    final_upload_index = next(
        index
        for index, step in enumerate(attest_steps)
        if step.get("with", {}).get("name") == "tomiya-code-atlas-go-windows-x64"
    )
    gui_smoke_index = next(
        index
        for index, step in enumerate(build_steps)
        if step.get("name") == "Go GUIの配布起動をsmoke test"
    )
    distribution_e2e_index = next(
        index
        for index, step in enumerate(build_steps)
        if step.get("name") == "one-dir配布物のE2E検証"
    )
    artifact_paths = {
        line.strip()
        for line in attest_steps[final_upload_index]["with"]["path"].splitlines()
        if line.strip()
    }
    assert artifact_paths == {
        ".build/release/tomiya-code-atlas-windows-x64.zip",
        ".build/release/SHA256SUMS.txt",
    }
    assert distribution_e2e_index < package_index < intermediate_upload_index
    assert package_index < pr_upload_index < intermediate_upload_index
    assert gui_smoke_index < distribution_e2e_index
    assert pr_upload["if"] == (
        "github.event_name == 'pull_request' && "
        "github.event.pull_request.head.repo.full_name == 'tomiya7688/tomiya_code_atlas'"
    )
    assert pr_upload["uses"] == "actions/upload-artifact@v7"
    assert pr_upload["with"]["name"] == (
        "tomiya-code-atlas-windows-x64-pr-${{ github.event.pull_request.number }}"
    )
    assert pr_upload["with"]["path"].splitlines() == [
        ".build/release/tomiya-code-atlas-windows-x64.zip",
        ".build/release/SHA256SUMS.txt",
    ]
    assert pr_upload["with"]["retention-days"] == 7
    assert build_steps[package_index]["run"] == (
        "python scripts/build/package_windows_distribution.py"
    )
    assert build_steps[package_index]["env"] == {"PYTHONIOENCODING": "utf-8"}
    assert build_job["permissions"] == {"contents": "read"}
    assert attest_job["needs"] == "build-windows-exe"
    assert attest_job["if"] == (
        "github.event_name != 'pull_request' "
        "&& github.repository == 'tomiya7688/tomiya_code_atlas'"
    )
    assert attest_job["permissions"] == {
        "contents": "read",
        "id-token": "write",
        "attestations": "write",
    }
    assert intermediate_upload["if"] == attest_job["if"]
    assert intermediate_upload["with"]["path"].splitlines() == [
        ".build/dist/",
        ".build/release/",
    ]
    cli_smoke = (
        Path(__file__).resolve().parents[1] / "tools/go_cli_smoke.py"
    ).read_text(encoding="utf-8")
    expected_operations = {
        "call-graph",
        "class-diagram",
        "sequence-diagram",
        "object-diagram",
        "communication-diagram",
        "state-diagram",
        "timing-chart",
        "use-case-diagram",
        "responsibility-table",
        "package-diagram",
        "component-diagram",
        "deployment-diagram",
        "ci-graph",
        "comments",
        "static-spec",
        "design-quality",
    }
    assert all(
        f'"--operation",\n        "{operation}"' in cli_smoke
        for operation in expected_operations
    )
    assert all(
        f'"go_project_{operation}.md"' in cli_smoke
        for operation in expected_operations - {"comments"}
    )
    download_step = attest_steps[0]
    assert download_step["uses"] == "actions/download-artifact@v8"
    assert download_step["with"]["path"] == ".build"
    assert attest_steps[attestation_index]["with"]["subject-checksums"] == (
        ".build/release/ATTESTED-SUBJECTS.txt"
    )
    assert attestation_index < verification_index < final_upload_index
    assert "gh attestation verify" in attest_steps[verification_index]["run"]
    assert (
        "Test-Path -LiteralPath $path -PathType Leaf"
        in attest_steps[verification_index]["run"]
    )
    assert "--signer-workflow" in attest_steps[verification_index]["run"]
    assert (
        '--source-digest "${{ github.sha }}"' in attest_steps[verification_index]["run"]
    )
    assert '--source-ref "${{ github.ref }}"' in attest_steps[verification_index]["run"]
    assert '".build/release/SHA256SUMS.txt"' in attest_steps[verification_index]["run"]
    assert '".build/dist/SHA256SUMS.txt"' in attest_steps[verification_index]["run"]
    assert attest_steps[verification_index]["env"] == {
        "GH_TOKEN": "${{ github.token }}"
    }

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
