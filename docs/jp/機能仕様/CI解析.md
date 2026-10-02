# CI解析器

## 目標

CI pipelineで利用でき、アーキテクチャやコード品質のregressionを自動検出できる解析を提供します。

## 責務

- 決定的なanalyzerを非対話で実行する。
- CIで使える安定したmachine-readable結果を作る。
- UI codeと結合せずpass／fail thresholdを設定できるようにする。
- localの図表／表生成と同じ中間解析modelを再利用する。

## 候補となるcheck

- dependency／call graphの変化
- 過剰なcouplingやfan-in／fan-out
- 新しく導入されたcycle
- 構造的複雑度の増加
- 設計評価scoreのregression
- 対応言語のparser／analyzer失敗

## 出力

人が読めるsummaryとmachine-readable結果を出します。exit statusは設定済みのCI失敗条件を反映します。

## 境界

CIでは決定的なcheckを既定とします。LLM支援評価は任意とし、再現可能なCIを保てるよう明確に区別します。