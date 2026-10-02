# Python AST／意味解析バックエンドの評価

Issue #132では、Python parser backendを#131のbackend適合fixtureに照らして評価します。

## 選定

現在のCommon IRでは、主Python parser backendに **CPython標準libraryの `ast` を維持**します。将来のsymbol／scope IRには、標準libraryの `symtable` を意味解析の補助として使います。現時点では主経路をLibCSTやtree-sitter-pythonへ切り替えません。

選定したruntime backend ID:

```text
python-stdlib-ast
```

`PythonStdlibBackend` はTomiyaのversion付き `ParserBackend` 契約を実装し、`ApplicationService` が利用します。

## 候補比較

| 候補 | 構文／AST | symbol／意味情報 | 不完全なソース | 配布 | Common IR化コスト |
| --- | --- | --- | --- | --- | --- |
| 標準 `ast` + `symtable` | 実行中Python versionの高い意味情報忠実度 | `symtable` によるcompiler scope／name binding。project全体のtype推論ではない | 不正構文は失敗 | 追加依存なし | 最小 |
| LibCST | 書式を保つfull-fidelity CST | scopeとqualified nameのmetadata。局所metadataは豊富 | error recovery backendには選定しない | package／native build面が増える | 中程度 |
| tree-sitter-python | 強いincremental concrete syntax parser | 単体ではPython意味層なし | 不完全／editor bufferに適した候補 | native／parser依存 | 中〜高 |

Pythonの `symtable` はcompilerが作るscope tableとidentifier binding情報を公開します。LibCSTはscope／qualified name等のmetadataを提供しますが、任意のattribute／type解決を完全には行いません。tree-sitter-pythonは、意味解析より不完全入力の許容を優先する場合に有用です。

## 主backendを標準libraryに保つ理由

現在のTomiya Common IRが使うのはclass、function、method、継承、call、object、state／timing情報、importです。標準library adapterは追加配布依存なしでそれらを直接対応付けています。現在利用する項目の改善が十分でないままCST backendに切り替えると、変換とpackageの複雑さが増します。

これは標準ASTが常に最良のPython parserだという主張ではありません。**現在のTomiya Common IRと配布デスクトップtool**に対する選定です。

## 既知の制約

1. syntax対応範囲はTomiyaを実行するPython interpreterのversionに依存します。
2. 標準AST／symtableはmoduleをまたぐ完全なtype／call target推論を提供しません。
3. 不正／未完成ソースは拒否され、`PythonStdlibBackend` はこれを `unsupported_syntax` として正規化します。
4. editor buffer／不完全ソース解析が製品要件になる場合、semantic primaryを黙って置き換えずfallbackとしてtree-sitter-pythonを評価します。

## 配布とLicense

- CPython標準library: 同梱interpreterに含まれ、追加runtimeは不要です。
- LibCST: 全体はMITですが、repositoryにはPSF由来fileの記載があります。現在のreleaseにはnative／binary部品があるため、採用前にfrozen buildでの明示的な検証が必要です。
- tree-sitter-python: MITです。採用するとparser／bindingのnative依存が増えます。

parser packageを新たに加えないため、既存のWindows PyInstaller onedir配布方式を変更しません。

## 検証

`tests/test_python_backend_selection.py` は#131適合fixtureを再利用し、選定backend descriptor、Common IR出力、標準library symbol tableのscope probe、正規化されたsyntax error境界を検証します。

2026-09-18時点で確認した情報源:

- Python `symtable`: https://docs.python.org/3/library/symtable.html
- LibCST: https://github.com/Instagram/LibCST
- tree-sitter-python: https://github.com/tree-sitter/tree-sitter-python