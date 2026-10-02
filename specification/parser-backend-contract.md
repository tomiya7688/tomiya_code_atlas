# Parser Backend Contract

この仕様は、Tomiya Code Atlasのlanguage adapterが使用するparser backendの安定した境界を定義します。in-process library、native binding、同梱helper executable、subprocess parserに適用します。

## 契約version

現在の契約versionは **`1`** です。

backendは安定した`backend_id`、対象`language`、実行形態`kind`、`contract_version`を公開しなければなりません。hostは対応していない契約versionを、互換性を推測して受け入れず、`protocol_error`として拒否します。

## 境界ルール

交換可能なparser backendは、backend固有のAST、syntax tree、compiler、FFI、例外の型をlanguage adapterより上の層へ公開してはいけません。

```text
parser library / helper / native code
            ↓
       ParserBackend
            ↓
    Common IR (ModuleIR)
            ↓
Analyzer / Generator / Evaluator
```

backend内部では任意のparser実装言語・libraryを利用できます。backend固有の表現からCommon IRへの変換は、この境界の内側で行います。

## 正規化されたfailure種別

version 1では、次の安定した種別だけを定義します。

- `failure` — より具体的な種別に該当しない、一般的なbackend実行・解析失敗
- `timeout` — hostまたはbackendの制限時間を超過
- `unsupported_syntax` — backendが意図的に未対応としている構文・言語機能
- `unsupported_language` — 要求されたソース言語にbackendが未対応
- `protocol_error` — 不正なmessage、非互換version、必須field不足、不正な応答、transport契約違反

正規化されたfailureには次の情報を含めます。

```json
{
  "kind": "failure",
  "message": "human-readable detail",
  "backend_id": "example-python-backend",
  "retryable": false
}
```

adapter境界より上のcallerは、backend library固有の例外名、stack trace、compiler object、FFI handleに依存してはいけません。安全で有用な場合は、境界内で診断情報をlogに記録できます。

## subprocess / helper間のwire protocol

parser backendをhelper processとして動かす場合、version 1ではUTF-8のJSON messageを使います。transportは、process起動ごとに1回のrequest/responseを行う方式、または永続helper向けのnewline-delimited JSONを利用できます。どちらもmessage envelopeは共通です。

### request

```json
{
  "contract_version": "1",
  "request_id": "opaque-request-id",
  "operation": "parse",
  "language": "python",
  "source": "def main(): pass\n",
  "path": "optional/source.py"
}
```

必須field:
- `contract_version`
- `request_id`
- `operation`（version 1では`parse`）
- `language`
- `source`

`path`は任意metadataです。ソース文字列の解析に必須としてはいけません。

### 成功応答

```json
{
  "contract_version": "1",
  "request_id": "opaque-request-id",
  "ok": true,
  "ir": {}
}
```

`ir`はserializeされたCommon IR payloadです。helperは応答前にparser固有の構造を言語非依存のIR schemaへ変換します。

### failure応答

```json
{
  "contract_version": "1",
  "request_id": "opaque-request-id",
  "ok": false,
  "error": {
    "kind": "unsupported_syntax",
    "message": "pattern matching is not supported by this backend",
    "backend_id": "example-python-backend",
    "retryable": false
  }
}
```

応答には成功時の`ir`か正規化された`error`のどちらか一方だけを含めます。

## transportルール

- protocol payloadはUTF-8です。
- stdoutをtransportに使う場合、helperのstdoutはprotocol応答専用です。診断出力はstderrへ送ります。
- processのtimeoutを管理するのはhostです。timeoutは正規化された`timeout`種別へ変換します。
- 有効な正規化応答を伴わないnon-zero終了は、実行失敗かprotocol違反かに応じて`failure`または`protocol_error`にします。
- 前方互換性のため、未知の任意fieldは無視できます。
- version 1の必須fieldが欠落している場合は`protocol_error`です。
- 永続helperが応答を対応付けられるよう、request IDは変更せず返します。

## native / in-process backend

native libraryやin-process parserは、内部でJSONをserializeする必要はありません。ただしlanguage adapterへ、同一の論理契約を公開します。

- 安定したbackend識別子とversion metadata
- Common IR出力
- 正規化されたfailure種別
- 境界より上へparser固有型を漏らさない

## 配布

Windowsアプリには、helper executableまたはnative libraryとしてbackendを同梱できます。任意の別連携として明示されていない限り、利用者にcompiler SDK/runtime/toolchainの別途導入を要求してはいけません。

同梱binary/libraryの場所を特定するのはpackaging層です。Analyzer / Generator / Evaluatorはfile path、実行file名、FFI handle、transport方式を知ってはいけません。

## 互換性

必須fieldの意味の変更、failure種別の削除、wire envelopeの変更には新しい契約versionが必要です。古いhostが安全に無視できる任意fieldの追加は、version 1のまま行えます。
