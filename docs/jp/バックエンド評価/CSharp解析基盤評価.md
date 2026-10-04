# C# AST／意味解析バックエンドの評価

Issue #134では、C#解析backendを#131の適合fixtureに照らして評価します。

## 選定

C#の主たる構文／意味解析backendに **Roslyn / Microsoft.CodeAnalysis.CSharp** を選びます。

```text
backend id: csharp-roslyn-helper
hosting: 同梱helper executable
protocol: Parser Backend Contract v1 JSON
runtime: self-contained .NET publish
```

tree-sitter-c-sharpは、編集途中bufferの構文fallback候補として残しますが、意味解析の主backendにはしません。

## Roslynを選ぶ理由

TomiyaのC#解析ではsyntax tree以外に次の情報が必要です。

- symbol identity
- overload解決
- base class／interfaceとの関係
- generic type parameterと置換後の型
- method／call targetのbinding
- accessibility
- 未完成または不正なソースのdiagnostic

RoslynはC# compiler platformそのもので、compiler bindingで使われるsyntax tree、`SemanticModel`、symbol modelを公開します。Python側でC#のoverload／type解決ルールを再実装する必要がありません。

tree-sitter-c-sharpは強力なincremental syntax parserで、将来の復旧／fallbackとして有用です。ただし、semantic情報が必要な曖昧さはそのproject自体でも発生します。確かなcall／class図に必要なのはまさにその情報です。

## Fixtureの追加要素

C# fixtureには次を含めます。

- generic `Worker<T>`
- 継承
- `IWorker<T>` interface実装
- `run` overload
- 同じmethod callの複数回出現
- async／await
- local function／nested scope

backend固有のRoslyn syntax kindやsymbol objectはCommon IR goldenに加えません。

## Helperの構成

RoslynはPython process内ではなく、同梱する.NET helperで実行します。

```text
C# source／project
  -> backends/csharp/tomiya-csharp-backend.exe
      -> Roslyn syntax tree + compilation + SemanticModel
      -> 正規化済みwire DTO／Common IR fact
  -> Python Language Adapter境界
  -> Common IR
```

helperは定義済みのParser Backend Contract v1を使います。stdoutはprotocol message用に予約し、診断とlogはstderrへ出します。timeout、protocol不一致、未対応入力はhost契約に合わせて正規化します。

### Runtimeの配布

単一fileではなく、**self-contained win-x64 folder**としてpublishします。

```text
backends/
  csharp/
    tomiya-csharp-backend.exe
    *.dll
    *.json
    .NET runtime files
```

publish commandは次と同等の設定を使います。

```text
dotnet publish -c Release -r win-x64 --self-contained true
```

初期段階では `PublishSingleFile` とtrimmingを無効にします。これによりRoslynのreflection／loading上の問題を避け、既存のonedir配布方式と揃えます。self-contained .NET publishは必要な.NET runtimeを配布folderに含めるため、対象PCへの.NET runtime別途インストールを不要にします。

## SDK不要でのProject意味解析

初期helperでは、`MSBuildWorkspace` をruntime必須条件にせず `CSharpCompilation` を直接使います。

- Tomiyaのproject探索がC# source fileを集める。
- helperがsyntax treeとcompilationを作る。
- framework metadata referenceはhelper自身のruntimeから取得する。
- project／source referenceは発見済みproject入力から解決する。
- Unityや第三者assemblyはproject情報から解決できる場合に使い、解決できないsymbolは明示的に未解決とする。

これによりruntimeをself-containedに保ち、利用者に.NET SDK／MSBuildの導入を要求しません。

## 候補比較

| 候補 | 構文 | symbol／overload／interface／generic | 不完全なソース | 配布 | 判断 |
| --- | --- | --- | --- | --- | --- |
| Roslyn | compiler基準 | compiler標準のsemantic binding | syntax tree + diagnostic | self-contained .NET helper folder | **選定** |
| tree-sitter-c-sharp | 強力なincremental grammar | 単独ではsemantic bindingなし | 強い | native parser assets | fallback候補 |
| 現行regex adapter | コメント向けheuristic | なし | 該当なし | 依存が軽い | 解析backendではない |

## ライセンス

- Roslyn: MIT。
- tree-sitter-c-sharp: MIT。

## 実装状況

Roslyn helperは旧Python host向けに#150で実装され、#189でGo配布物向けに拡張しました。

- helper source: `backend-src/csharp/`
- Python host: `Src/languages/csharp_backend.py`
- 共通subprocess protocol host: `Src/languages/helper_backend.py`
- runtime path: `backends/csharp/tomiya-csharp-backend.exe`
- build: self-contained .NET 10 `win-x64` folder、single-file／trimmingなし。Roslyn/.NET licenseとthird-party noticesを同梱
- 構文情報: class/interface/struct/record/enum、method/constructor/local function、property/field、型・visibility・location
- 不正なC#構文: Parser Backend Contract v1の`unsupported_syntax`で返す。Unity SDK等を参照できないsemantic情報はdiagnosticに残す。
- Linux CI: Roslyn helper build + .NET/Unity fixtureとprotocol smoke
- Windows Go EXE workflow: helperをself-contained publishし、CPython helperとともにartifactへ含める

利用者向け配布物には.NET SDKやruntimeの別途導入は不要です。

2026-09-18時点で確認した情報源:

- https://github.com/dotnet/roslyn
- https://github.com/tree-sitter/tree-sitter-c-sharp
- https://learn.microsoft.com/dotnet/core/deploying/
