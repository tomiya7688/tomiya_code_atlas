from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def test_go_format_check_normalizes_line_endings_and_discovers_sources() -> None:
    workflow = (ROOT / ".github" / "workflows" / "go-exe.yml").read_text(
        encoding="utf-8"
    )

    assert '$goSourceRoots = @("go", "parser-backends\\go")' in workflow
    assert (
        "$goSourceFiles += @(Get-ChildItem -LiteralPath $goSourceRoot -Recurse "
        '-File -Filter "*.go"'
    ) in workflow
    assert 'Join-Path $PWD ".build\\gofmt-check"' in workflow
    assert (
        "$sourceText = $sourceText.Replace($carriageReturn + $lineFeed, $lineFeed).Replace($carriageReturn, $lineFeed)"
        in workflow
    )
    assert "gofmt -l $goSources" in workflow
    assert (
        'Write-Output "gofmt execution failed with exit code $gofmtExitCode."'
        in workflow
    )
    assert workflow.index("$gofmtExitCode = $LASTEXITCODE") < workflow.index(
        "if ($unformattedGo)"
    )
