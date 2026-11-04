import json
from pathlib import Path

import yaml


# {
#   責務: [test_windows_gui_artifact_includes_runtime_assets_after_smoke: ダウンロードした配布物がGUI assetとparserを含み、E2E確認の後に保存されることを検証する,],
# }
def test_windows_gui_artifact_includes_runtime_assets_after_smoke() -> None:
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
    # 除外やglobを使う場合は照合方法を見直し、誤った成功を防ぐ。
    assert not any(any(char in path for char in "!*?[]") for path in artifact_paths)
    assert ".build/dist/backends/manifest.json" in artifact_paths
    assert ".build/dist/assets/" in artifact_paths
    assert gui_smoke_index < distribution_e2e_index < upload_index

    build_script = (
        Path(__file__).resolve().parents[1] / "scripts/build/build_go.bat"
    ).read_text(encoding="utf-8")
    assert "generate_app_icon.ps1" in build_script
    assert '".build\\dist\\assets\\atlas-kun.ico"' in build_script
    assert '".build\\dist\\assets\\atlas-kun-256.png"' in build_script
    assert '".build\\dist\\assets\\atlas-kun-64.png"' in build_script
    assert '".build\\dist\\assets\\LICENSE.md"' in build_script

    # manifestに登録されたhelperとライセンスが、保存対象のフォルダーに含まれるか確認する。
    manifest_path = (
        Path(__file__).resolve().parents[1] / "parser-backends/manifest.json"
    )
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    assert {"python", "gdscript", "csharp", "cpp", "java", "go"} <= {
        backend["language"] for backend in manifest["backends"]
    }
    for backend in manifest["backends"]:
        relative_paths = [backend["path"], *backend.get("licenses", [])]
        if backend["language"] == "java":
            runtime = backend.get("runtime", "")
            assert runtime.startswith("java/"), "Java runtimeの配布先が不正です"
            relative_paths.append(runtime)
        for relative_path in relative_paths:
            artifact_file = f"{manifest['build_output_root']}/{relative_path}"
            assert any(
                artifact_file == path
                or (path.endswith("/") and artifact_file.startswith(path))
                for path in artifact_paths
            ), f"{backend['backend_id']}: artifact対象外のパス {artifact_file}"
