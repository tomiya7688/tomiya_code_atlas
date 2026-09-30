# CI解析とソース解析の連携

Tomiya Code Atlasは、保守的で言語に依存しないpath hintを使ってCI構造とソース解析を結び付けます。CI Analyzerはソースファイルを開いたり、importを解決したり、言語parserを呼び出したりしません。

## 処理の流れ

```text
CI provider設定
  -> provider analyzer
  -> CIWorkflow / CIJob / CIStep
  -> CIStep.path_hints
  -> collect_ci_path_hints()
  -> プロジェクトファイル探索／存在確認
  -> 言語判定
  -> Language Adapter
  -> Common IR / graph / diagram / evaluator
```

## Path hintの契約

各 `CIStep` は0個以上の `path_hints` を持てます。hintはCI設定内で見つかった根拠であり、そのpathが存在することやソースコードであることを保証しません。

例:

- `pytest tests/unit/test_service.py` -> `tests/unit/test_service.py`
- `ruff check Src tests` -> `Src`, `tests`
- `working-directory: ./backend` -> `backend`
- `uses: ./.github/actions/setup` -> `.github/actions/setup`

`collect_ci_path_hints()` は `CIPathHint(job, step, path)` を通じて元のjobとstepを保持します。

## 責務の境界

CI Analyzerの責務:

- provider固有の構文解析
- job、step、`needs` 関係の抽出
- command／action metadataの抽出
- 保守的なpath hint抽出

CI Analyzerの責務ではないもの:

- ファイルシステム上の存在確認
- glob展開
- pathをソース、テスト、生成物、toolingのどれと判断すること
- 言語判定
- ソース解析
- ソース向けCommon IRの構築

これらは既存のプロジェクト探索とLanguage Adapterの境界に置きます。これにより、CI入力の端だけがprovider固有となり、後続の同じソース解析pipelineを再利用できます。

## 将来の逆方向trace

将来的なプロジェクト単位のtraceでは、path hintを選択済みrepositoryに照合して次のような関係を構築できます。

```text
変更されたソース
  -> 対応するCI path hint／project scope
  -> CI step
  -> CI job
  -> downstream needs edge
  -> package／deploy artifact
```

解決できない、または曖昧なhintは、確定した関係を捏造せず未解決／曖昧なまま保持します。