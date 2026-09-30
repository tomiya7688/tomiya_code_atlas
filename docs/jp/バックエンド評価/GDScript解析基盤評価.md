# GDScript AST／意味解析バックエンドの評価

Issue #133では、GDScript解析backendを#131の共通適合fixtureに照らして評価します。

## 選定

GDScriptの主たる構文backendに **`tree-sitter-gdscript`** を選びます。

選定記録で参照する正規package／repository:

```text
package: tree-sitter-gdscript
repository: PrestonKnopp/tree-sitter-gdscript
backend id: gdscript-tree-sitter
role: 主構文backend
```

tree-sitterを完全な意味解析backendとして扱ってはいけません。Godot projectの意味情報はLanguage Adapterの背後に別層として追加します。

## tree-sitter-gdscriptを選ぶ理由

このgrammarは、構造解析backendで必要な次のGDScript構文を明示的にモデル化します。

- `signal`
- `class_name`
- `extends`
- `@export` などのannotation
- 型付き変数とcontainer
- `await`
- lambda
- match構文
- Godot固有のStringName、NodePath、node取得構文

`preload()` と `load()` は構文上は通常のcallです。adapterがstring／path引数を抽出し、後段のGodot project resolverで解決できます。

評価時に確認したreleaseは **v6.1.0** で、repositoryでは2026-07-13までのmaintenance commitが確認されました。Python packageにはnative tree-sitter grammarが含まれ、利用者にGodot editor／engineの導入を求めません。

## 候補比較

| 候補 | 構文 | 意味解析／project情報 | error recovery | 配布 | 判断 |
| --- | --- | --- | --- | --- | --- |
| PrestonKnopp/tree-sitter-gdscript | Godot固有のgrammarが強い | 構文のみ | tolerant parsing | Python／native assetsを同梱可能 | **主構文backend** |
| GDQuest/tree-sitter-gdscript | 同じgrammar系統でGodot指向の開発が活発 | 構文のみ | 同じ方式 | 同じnative binding方式 | fork／mirrorとして追跡 |
| Godot `GDScriptParser` | 権威性が高く情報も豊富 | Godot core／projectに統合されたparser・type model | native diagnostic | engine core、cache、resource classと強く結合 | 初期runtimeには採用しない |
| 現行regex comment adapter | 構造認識は非常に限定的 | なし | 該当なし | 依存が軽い | 置換まではコメント生成専用 |

## Godot parserを採用する場合のトレードオフ

Godot本体のparserは権威性があり、`PreloadNode`、`SignalNode`、`AwaitNode`、`ClassNode` や豊富な `DataType` modelを持ちます。意味解析面では魅力的です。

一方、internal C++ parserはGodot cache、Resource、ScriptLanguage、Variantなどengine coreの型と結合しています。Tomiya helperとして配布するには大きなGodot由来の実行物／libraryを保守するか、Godot runtimeを利用者に要求する必要があり、self-contained Windows onedir方針に合いません。

後の解析で、project fileから妥当に再現できない正確なGodot compiler意味解析が必要になった場合は、version付きparser backend IPC contractの背後でGodot由来helperを再評価できます。

## 意味情報の層

選定する構成:

```text
.gd source
  -> tree-sitter-gdscript
  -> GDScript Language Adapter
       + Godot project resolver
         - res:// path
         - preload/load target
         - class_name registry
         - script inheritance
         - Node/Resource type catalogue
         - 取得可能なstatic signal connection
  -> Common IR
```

parser固有のtree nodeや将来のGodot API objectはLanguage Adapter境界より下に留めます。

## 適合fixture

GDScript fixtureには次を含めます。

- `class_name`
- `signal`
- `preload()`
- `load()`
- `Node` / `Resource` type reference
- `@export`
- 継承
- 同じcallの複数回出現
- `await`
- lambda
- 型付きcontainer

fixtureはbackendに依存させず、tree-sitter node名をCommon IR goldenに加えません。

## Licenseと再配布

- tree-sitter-gdscript: MIT。
- Godot Engine parser source: MIT。
- 選定したtree-sitter経路ではGodotの導入は不要です。

実装時には、既存PyInstaller Windows **onedir** artifact内でPython／native grammarとtree-sitter runtimeが動くことを検証します。

## 今後の作業

#133は選定のみです。現在のregex-only GDScript comment adapterを解析backendとして置き換えるには、別の実装Issueが必要です。その実装ではversion付き `ParserBackend` 境界と#131適合fixtureを使います。

2026-09-18時点で確認した情報源:

- https://github.com/PrestonKnopp/tree-sitter-gdscript
- https://github.com/GDQuest/tree-sitter-gdscript
- https://github.com/godotengine/godot/blob/master/modules/gdscript/gdscript_parser.h