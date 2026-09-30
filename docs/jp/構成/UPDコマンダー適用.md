# UPD Commanderの適用

Tomiya Code Atlasは `upd-commander-base-design` の有用な責務ルールを採用しますが、実行時にそのrepositoryや命名規則へ依存しません。規範ルールは `specification/architecture-policy.md` にあり、この文書では対応関係を説明します。

## 層の対応

### UI

- GUI / CLI / 起動の流れ
- 利用者入力と表示設定
- 結果の提示と保存指示

UIはソース解析を実装せず、Dataの実装詳細も直接所有しません。

### Process

- アプリケーションのオーケストレーション
- Analyzer / Generator / Evaluator / Rendererの処理の流れ
- Common IRの言語非依存な利用
- 次に実行する操作の決定

ProcessはUIの具体的な表示や保存形式を知りません。

### Data

- 対象ソース／設定の取得
- ファイルI/O、cache、永続化
- 外部データへのアクセス

DataはUIや解析ドメインの判断をしません。

## Commander

Commanderが答えるのは、**次に何を実行するか**です。

許可されること:

- request／resultを受け取る
- 適切なProcessing／serviceを選ぶ
- Messengerに境界越しの通信を依頼する
- resultを次の処理へ渡す

Commanderの責務ではないもの:

- AST解析
- graph algorithm
- 図表生成
- Renderer構文の生成
- file／database／network I/O
- 大規模な計算や変換

したがってCommander内部のloop、計算、I/O呼び出しはレビュー対象となり、policy上の指摘になる場合があります。

## Messenger

Messengerはアプリケーション境界を越えてrequest／resultを運びます。

許可されること:

- 境界messageの送受信
- contractに必要なtransport／境界表現だけを変換する
- 受信したmessageを自身のCommander／アプリケーション入口へ渡す

Messengerの責務ではないもの:

- ドメインalgorithmの選択
- 解析／評価／描画
- データの永続化
- 業務判断

## Processing

実処理はProcessing／serviceまたは既存の専門module（言語adapter、analyzer、generator、renderer、evaluator、data access実装）に置きます。責務が適切なら、すべての既存moduleを文字通り `Processing` に改名する必要はありません。

## 依存の流れ

概念上の境界:

```text
UI <-> Process <-> Data
```

避けるべき近道には、UIからDataの実装へ直接アクセスすることや、アプリケーション境界が仲介すべき別層の内部Processingを直接呼ぶことがあります。

既存のCode Atlas pipelineのルールも引き続き有効です。

```text
ソース
 -> Language Adapter / Parser
 -> Common IR / 共有モデル
 -> Analyzer / Generator / Evaluator
 -> 論理出力
 -> Renderer
```

UPDはそのpipelineを囲むアプリケーション層の境界です。言語／IR／Rendererの分離を置き換えるものではありません。

## Message契約

層をまたぐmessageは、小さく明示的で、UI framework、database client、parser library node、Renderer固有の構文に依存させません。反対側のframeworkを構築せずにテストできる安定した値／recordを優先します。

都合だけを理由に、WPF/Tk/Godot control、DB connection、Roslyn node、Python `ast` nodeなどを共有application contractに渡してはいけません。

## エラー処理

- 操作を所有する層でエラーを検出・処理します。
- 上位層に詳細を知らせない場合は、境界でframework／storage／parser固有エラーを変換します。
- 原因と診断に必要なcontextを保持します。
- Commander／Messengerの経路を単純にするためだけに、エラーを黙って無視しません。
- エラーの見せ方はUIが決めます。Process／DataはUI表示用の文言整形ではなく、domain／applicationに関係する失敗情報を返します。

## テスト方針

- Processing / analyzer / generatorの処理: 直接のunit／regression test。
- Commander: 適切なservice／message経路を選ぶことを確かめるorchestration test。
- Messenger／境界: ドメイン処理を含まずrequest／resultを渡すcontract／integration test。
- 層／importの機械的ルール: `python tools/context_tool.py policy-check`。
- 意味上の責務分担: 責務マップと対象を絞ったアーキテクチャレビュー。全てを機械判定できるように見せかけない。

## ルールの強さと例外

`specification/architecture-policy.md` はRequired、Recommended、Advisoryを区別します。自動チェックも確定したerrorと警告を分けます。

必要な例外は範囲を限定します。該当ルール、理由、範囲、軽減策、削除／再確認の条件を、関係するIssue／PRに記録します。

## レビューチェックリスト

- UIに解析やstorageの実装処理が入っていないか。
- Processが表示形式や保存形式の詳細に依存していないか。
- DataがUIや解析ドメインの判断をしていないか。
- Commander／Messengerに実処理が入り込んでいないか。
- framework／parser固有の型が安定した境界を越えていないか。
- Common IR／共有modelに機能のオーケストレーション・評価・描画処理が入っていないか。
- Rendererがソースを解析したりLanguage Adapterをimportしたりしていないか。
- 変更した責務を `docs/jp/責務マップ.md` の短い一行で説明できるか。

関連文書:

- `specification/architecture-policy.md`
- `docs/jp/責務マップ.md`
- `docs/jp/開発運用.md`