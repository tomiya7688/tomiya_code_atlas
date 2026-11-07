# UML意味モデルとRenderer能力

## 背景

Tomiya Code Atlasは、解析結果を図の論理modelへ変換してからMermaidまたはPlantUMLへ出力します。Generatorが保持するmodelは、ソース言語にも出力構文にも依存しない製品内の図表現です。ただし、現在のmodelはUML 2.xの完全なmetamodelではなく、各機能Issueで定める利用目的に必要な意味の範囲を表します。

Rendererは論理modelを受け取り、対応する記法へ直列化します。出力先に同種のnative diagram構文がない場合、別の構文へ近似変換します。その場合は下のCapability Matrixで近似と明記し、UML準拠の出力として扱いません。

## 意味modelの境界

```text
言語別Parser
  -> 言語非依存の解析結果
  -> 図の論理model
  -> 系列・共有図へのpresentation partition
  -> Renderer固有の構文
```

- AnalyzerとGeneratorは、MermaidやPlantUMLの構文、node ID、arrow表記をmodelへ格納しません。
- 論理modelは、図の要素・関係・順序・条件など、実装から確認できた意味情報を保持します。推定と確定の区別がある場合はその区別も保持します。
- partitionは表示のためのviewを作ります。元の意味関係を削除・変更せず、図の境界は参照として示します。詳細は[図の系列分割](%E5%9B%B3%E3%81%AE%E7%B3%BB%E5%88%97%E5%88%86%E5%89%B2.md)を参照してください。
- cycle強調、確度、設計警告、high fan-in、shared group、続き図への参照はTomiya Code Atlasのoverlayまたはnavigation metadataです。UML要素や関係の意味へ混ぜません。
- Rendererはmodelの内容を別のUML関係へ読み替えません。出力構文が意味を表せない場合は、Capability Matrixに沿って近似またはunsupportedとして扱います。

## Capability Matrix

状態は現在のGo実装が出力する構文に対する評価です。「忠実」は、現在の論理modelが持つ項目を対応するdiagram構文へ表せることを指し、UML規格全体への完全準拠を意味しません。「近似」は別種のdiagram構文へ写す、または重要なUML意味をmodel / 出力が表さない状態です。「未対応」は該当形式のRendererを実装していない状態です。

| 図の種類             | Goの論理model          | Mermaid                    | PlantUML               | 現在表せる意味と境界                                                                                                                                                                                                                                               |
| -------------------- | ---------------------- | -------------------------- | ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| クラス図             | `ClassDiagram`         | 近似: `classDiagram`       | 近似: class diagram    | class / member、visibility、inheritanceを保持します。call由来の`uses`は両Rendererがassociation `-->`で描画するため、UML関係として近似です。他relationは未表現です。[#209](https://github.com/tomiya7688/tomiya_code_atlas/issues/209)で拡張します。                |
| シーケンス図         | `SequenceDiagram`      | 忠実: `sequenceDiagram`    | 忠実: sequence diagram | ordered message、call、return、async、recursive call、深度打切りを保持します。UML interaction fragment、activation、lifeline create/destroy等は未表現です。詳細は[#210](https://github.com/tomiya7688/tomiya_code_atlas/issues/210)を参照します。                  |
| 状態遷移図           | `StateDiagram`         | 近似: `stateDiagram-v2`    | 未対応                 | 初期・終端state、遷移、event、guardを保持します。任意の遷移元を表すため、仮想state「Any current state」を出力します。composite state、region、entry/exit/do action等は未表現です。[#211](https://github.com/tomiya7688/tomiya_code_atlas/issues/211)で拡張します。 |
| アクティビティ図     | `ActivityFlow`         | 近似: `flowchart`          | 未対応                 | action相当node、decision、順序付きedgeをflowchartへ写します。UML activity partition、fork/join、object flow等は保証しません。詳細は[#219](https://github.com/tomiya7688/tomiya_code_atlas/issues/219)で監査します。                                                |
| コミュニケーション図 | `CommunicationDiagram` | 近似: `flowchart`          | 未対応                 | participant間のrelationとmessage orderをedge labelで示します。UML connector / message semanticsとsequence diagram同等の順序表現は保証しません。詳細は[#212](https://github.com/tomiya7688/tomiya_code_atlas/issues/212)で拡張します。                              |
| オブジェクト図       | `ObjectDiagram`        | 近似: `flowchart`          | 未対応                 | instance、type、field value、referenceをnode/edge labelで示します。UML `InstanceSpecification`、slot、classifierの形式的な意味は未保証です。詳細は[#213](https://github.com/tomiya7688/tomiya_code_atlas/issues/213)で拡張します。                                 |
| パッケージ図         | `PackageDiagram`       | 近似: `flowchart`          | 未対応                 | package/moduleとdependency edge、cycle情報を示します。UML package import、merge等は未表現です。詳細は[#214](https://github.com/tomiya7688/tomiya_code_atlas/issues/214)で拡張します。                                                                              |
| ユースケース図       | `UseCaseDiagram`       | 近似: `flowchart`          | 未対応                 | actorとuse caseは図に表示します。handler/call chainはMermaid sourceの`%% handler-chain` commentにのみ含み、描画画像には現れません。`usecase-beta`、include/extendは未対応です。[#215](https://github.com/tomiya7688/tomiya_code_atlas/issues/215)で拡張します。    |
| コンポーネント図     | `ComponentDiagram`     | 近似: `flowchart`          | 未対応                 | componentとdependency、external/boundary、cycleを示します。UML port、provided/required interface、assembly connector等は未表現です。詳細は[#216](https://github.com/tomiya7688/tomiya_code_atlas/issues/216)で拡張します。                                         |
| デプロイメント図     | `DeploymentDiagram`    | 近似: `flowchart`          | 未対応                 | workload、image、database等のnode kind、connection、確度を示します。UML node/deployment specification/communication pathの完全な意味は未保証です。詳細は[#217](https://github.com/tomiya7688/tomiya_code_atlas/issues/217)で拡張します。                           |
| タイミング図         | `TimingChart`          | 近似: `sequenceDiagram`    | 未対応                 | 処理eventの順序、await、parallel、wait等を示しますが、時刻軸・lifeline state/valueの時間変化を持ちません。詳細は[#218](https://github.com/tomiya7688/tomiya_code_atlas/issues/218)で境界を監査します。                                                             |
| コールグラフ         | `CallGraph`            | 製品固有graph: `flowchart` | 未対応                 | caller/callee関係を示すgraphです。UML diagramとしての意味は主張しません。                                                                                                                                                                                          |

「忠実」な行も、表に記載した現在の論理model範囲に限定されます。未表現のUML概念を空欄、別のrelation、任意のlabelであるかのように補いません。

## 出力表示とpartition

CLIが作成するartifact名は図の種類を示しますが、UMLの完全準拠を示すものではありません。出力構文が近似の場合はこのCapability Matrixに近似先と失われる意味を記載します。Rendererが対応外の図を別のsyntaxへ黙って置き換えてはいけません。

分割図の共有node、境界stub、continuation referenceは図をたどるためのpresentation metadataです。それらだけで関係を再構成できない場合も、元の論理modelとpartition metadataに意味関係を保持します。

## 期待値テスト

Goの[共有Renderer golden test](../../../go/internal/renderers/renderers_test.go)は、class、sequence、communication、object、package、component、state、timing、use case、deployment、call graphの代表的な出力をPythonと共有する期待値fixtureと比較します。activity flowchartには専用のRenderer testがあります。これらのgoldenは構文の安定性を確認します。UML規格への完全準拠や、実際に描画した図の可読性は証明しません。

各関連Issueはmodelの意味を広げるとき、対象RendererのCapability Matrix、期待値fixture、利用者向け説明を同時に更新します。受け入れではソース文字列のgoldenに加え、必要な場合は描画結果を確認します。

## 根拠と更新

Rendererの状態はGoの各Rendererとgolden testを基準に記録します。MermaidとPlantUMLが公開するdiagram構文はバージョンによって変わるため、出力syntaxを変更したIssueでは公式構文資料と実際のrenderer versionを確認してください。

- [Mermaid diagram syntax](https://mermaid.js.org/intro/syntax-reference.html)
- [PlantUML diagram overview](https://plantuml.com/)
- [PlantUML class diagram](https://plantuml.com/class-diagram)
- [PlantUML sequence diagram](https://plantuml.com/sequence-diagram)

## 利用方法

開発者は、図のRendererまたはlogical modelを変更した後、repository rootから `go test ./...` を `go/` で実行し、共有goldenの差分を確認してください。UML準拠や画面の可読性が受け入れ条件に含まれる場合は、golden成功だけで判断せず、描画結果も確認してください。
