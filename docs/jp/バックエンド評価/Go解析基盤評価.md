# Go AST／意味解析バックエンドの評価

Issue #137では、Go parser／意味解析backendを#131の共通適合fixtureに照らして評価します。

## 選定

Go標準libraryのparserとtype checkerを用いる**native Go helper**を選びます。

- `go/parser`
- `go/ast`
- `go/token`
- `go/types`

```text
backend id: go-stdlib-types-helper
role: 主構文 + 意味解析
runtime: native helper executable
外部Go導入: 不要
```

## go/packagesを必須にしない理由

`golang.org/x/tools/go/packages` はpackage／workspaceの読み込みに優れ、syntaxと完全な `Types` / `TypesInfo` を取得できます。しかしdocument上の既定build toolは **go command** です。

Tomiyaの配布方針は、利用者へcompiler／SDK／toolchainの導入を暗黙に要求しないことです。そのため `go/packages` はtoolchain対応の任意modeとし、必須runtime backendにはしません。

self-contained経路では発見済みproject source packageをTomiya自身がまとめ、Tomiya管理のimporter／resolverと `go/types` を使います。

## go/typesの意味解析価値

`go/types` は次を解析します。

- identifier／nameの解決
- type推論／検査
- interface実装／型の関係
- generic type checking
- function／method選択情報
- symbolに対するDefs／Uses対応

これはclass／interface関係、call解決、設計解析に必要な情報です。

## 候補比較

| 候補 | 構文 | 意味解析 | package読み込み | 配布 | 判断 |
| --- | --- | --- | --- | --- | --- |
| 標準parser + go/types helper | 公式Go AST | 強いtype checker | Tomiyaが所有するproject resolver | native executable | **選定** |
| go/packages | 公式syntax／types | 強い | module／build対応の読み込みに優れる | 通常go commandを実行 | 強化用の任意mode |
| gopls | 豊富なworkspace解析 | 強い | 高機能なworkspace model | protocol／toolchain面が大きい | 主backendにはしない |
| tree-sitter-go | incremental syntax | 単独ではなし | syntaxのみ | 小さなnative parser | fallback |
| 現行regex comment adapter | 限定的 | なし | なし | 依存が軽い | コメント専用 |

## 配布

helperはCIでcompileし、次のように配布します。

```text
backends/go/
└─ tomiya-go-backend.exe
```

Go binaryにはruntimeが含まれるため、利用者は実行のためにGoを導入しません。helperはstdin／stdout上のParser Backend Contract v1を使います。

## Project／moduleの扱い

self-contained mode:

1. Tomiyaが `.go` source fileを発見する。
2. fileをdirectory／package宣言ごとにまとめる。
3. 発見したsource packageからproject内importを解決する。
4. 利用可能ならvendor／source treeを使う。
5. 不足する外部packageは未解決として残し、symbolを作り出さない。

互換するGo toolchainが意図的に利用可能な場合は、任意toolchain-aware modeで `go/packages` を使えます。アプリケーションが暗黙にself-contained modeからsystem toolchain必須へ切り替わってはいけません。

## 適合fixture

Go fixtureには次を含めます。

- package宣言
- standard／project import
- generic interface
- generic struct
- embedded base struct
- method
- 同じcallの複数回出現
- goroutine／channel／select
- closure／nested function
- context cancel path

## ライセンス

Goとx/toolsはGo BSD-style licenseです。tree-sitter-goはMITです。

## 既知の制約

- build tag、GOOS／GOARCH、正確なmodule選択は意味解析結果に影響します。
- self-contained modeでは、source／export dataを取得できない外部moduleを解決できない場合があります。
- 構文対応範囲はhelperのbuild時に使ったGo versionに従います。
- toolchain-awareなmodule意味解析は任意の `go/packages` modeとして追加できます。

## 今後の作業

Go標準parser/type checker helper、Common IR変換、backend manifest entry、Windows配布とContract v1 adapterはIssue #193で実装しました。project全体の複数file/packageをまとめたimporterは、project scanを扱うIssue #198の範囲で整備します。

2026-09-18時点で確認した情報源:

- https://github.com/golang/go
- https://github.com/golang/tools/tree/master/go/packages
- https://github.com/tree-sitter/tree-sitter-go

