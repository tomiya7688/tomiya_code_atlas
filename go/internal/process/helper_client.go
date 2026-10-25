package process

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

const (
	// {
	//   責務: [ParserBackendContractVersion: 言語parser helperとの標準入出力契約のversionを固定する]
	//   値: [1: JSON requestとresponseの現在の契約]
	// }
	ParserBackendContractVersion = "1"
	defaultHelperTimeout         = 30 * time.Second
	defaultHelperStdoutLimit     = 8 * 1024 * 1024
	defaultHelperStderrLimit     = 64 * 1024
)

//	{
//	  責務: [BackendFailure: helper processまたはJSON protocol境界で正規化した失敗を保持する]
//	  状態: [Kind: 安定した失敗分類, Message: 利用者向け原因説明, BackendID: 失敗したhelper, Retryable: 再試行可能性]
//	}
type BackendFailure struct {
	Kind      string
	Message   string
	BackendID string
	Retryable bool
}

//	{
//	  責務: [BackendError: helper失敗をGo errorとしてProcess呼出元へ運ぶ]
//	  状態: [Failure: process・protocol詳細から切り離した失敗情報]
//	}
type BackendError struct {
	Failure BackendFailure
}

//	{
//	  責務: [HelperClient: versioned JSON requestをhelper processへ送りCommon IR responseを検証する]
//	  状態: [timeout: helper一回あたりの実行時間上限, maxStdoutBytes/maxStderrBytes: helper出力のmemory上限, createCommand: shellを介さずhelper processを起動する関数]
//	}
type HelperClient struct {
	timeout         time.Duration
	maxStdoutBytes  int
	maxStderrBytes  int
	createCommand   func(context.Context, string) *exec.Cmd
}

//	{
//	  責務: [boundedOutputBuffer: helper出力を上限まで保持し、超過を記録しながらpipe自体はdrainする]
//	  状態: [buffer: 保持済みbytes, limit: 最大保持byte数, exceeded: 上限超過の有無]
//	}
type boundedOutputBuffer struct {
	buffer   bytes.Buffer
	limit    int
	exceeded bool
}

//	{
//	  責務: [Write: helper出力のうち上限内だけを保持し、callerには全byte受理として返してprocessをblockさせない]
//	  処理: [1: 残容量だけbufferへ書く, 2: 超過分があればexceededを立てる, 3: 入力全長を返す]
//	  引数: [data: helper processから受け取ったbytes]
//	  戻り値: [count: 受理した入力全長, error: 常にnil]
//	}
func (output *boundedOutputBuffer) Write(data []byte) (int, error) {
	remaining := output.limit - output.buffer.Len()
	if remaining > 0 {
		keep := len(data)
		if keep > remaining {
			keep = remaining
		}
		_, _ = output.buffer.Write(data[:keep])
	}
	if len(data) > remaining {
		output.exceeded = true
	}
	return len(data), nil
}

//	{
//	  責務: [Bytes: 上限内で保持したhelper出力を返す]
//	  処理: [1: 内部bufferのbytesを返す]
//	  引数: []
//	  戻り値: [bytes: 保持済み出力]
//	}
func (output *boundedOutputBuffer) Bytes() []byte {
	return output.buffer.Bytes()
}

//	{
//	  責務: [String: 上限内で保持したhelper出力を文字列として返す]
//	  処理: [1: 内部bufferを文字列化する]
//	  引数: []
//	  戻り値: [text: 保持済み出力]
//	}
func (output *boundedOutputBuffer) String() string {
	return output.buffer.String()
}

//	{
//	  責務: [parseRequest: Parser Backend Contract v1のparse要求をhelperへ渡す]
//	  状態: [RequestID: response対応付けID, Operation: 実行する処理, Language: parser対象言語, Source: 解析対象source, Path: 任意のsource path]
//	}
type parseRequest struct {
	ContractVersion string  `json:"contract_version"`
	RequestID       string  `json:"request_id"`
	Operation       string  `json:"operation"`
	Language        string  `json:"language"`
	Source          string  `json:"source"`
	Path            *string `json:"path,omitempty"`
}

//	{
//	  責務: [helperResponse: helperから返ったversion・request対応・解析結果または失敗を受け取る]
//	  状態: [ContractVersion: helper契約version, RequestID: 対応するrequest ID, OK: 成功状態, IR: serialized Common IR, Error: helper側の正規化error]
//	}
type helperResponse struct {
	ContractVersion string          `json:"contract_version"`
	RequestID       string          `json:"request_id"`
	OK              bool            `json:"ok"`
	IR              json.RawMessage `json:"ir"`
	Error           *helperError    `json:"error"`
}

//	{
//	  責務: [helperError: helperが報告する正規化された失敗分類と説明を保持する]
//	  状態: [Kind: 失敗分類, Message: 失敗内容, Retryable: 再試行可能性]
//	}
type helperError struct {
	Kind      string `json:"kind"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

//	{
//	  責務: [Error: backend失敗分類から日本語の原因と対処を示すerror文字列を作る]
//	  処理: [1: failure kindに対応する日本語説明を選ぶ, 2: backend IDと詳細を付けて返す]
//	  引数: [failure: 正規化失敗を持つerror]
//	  戻り値: [message: 利用者向け日本語説明とbackend詳細]
//	}
func (failure *BackendError) Error() string {
	message := backendFailureSummary(failure.Failure.Kind)
	if failure.Failure.BackendID != "" {
		message += fmt.Sprintf("（parser: %s）", failure.Failure.BackendID)
	}
	if details := strings.TrimSpace(failure.Failure.Message); details != "" {
		message += " 詳細: " + details
	}
	return message
}

//	{
//	  責務: [backendFailureSummary: backend failure kindを利用者向けの日本語原因と対処へ変換する]
//	  処理: [1: 安定したfailure kindを照合する, 2: 原因と次の確認手順を返す]
//	  引数: [kind: helper processまたはprotocol境界の失敗分類]
//	  戻り値: [string: 日本語で書いた原因と利用者の確認手順]
//	}
func backendFailureSummary(kind string) string {
	switch kind {
	case "timeout":
		return "parserが制限時間内に完了しませんでした。入力fileを確認して再実行してください。"
	case "unsupported_syntax":
		return "sourceに現在のparserが対応していない構文があります。対象言語とversionを確認してください。"
	case "unsupported_language":
		return "この言語のparserを利用できません。対応parserを含む配布物を使ってください。"
	case "protocol_error":
		return "parserとの通信結果を読み取れませんでした。配布物を再取得して再実行してください。"
	case "failure":
		return "parserの実行に失敗しました。sourceとparserの環境を確認してください。"
	default:
		return "parserで解析できませんでした。入力fileと配布物を確認してください。"
	}
}

//	{
//	  責務: [NewHelperClient: timeoutを固定しhelper用subprocess factoryを持つclientを作る]
//	  処理: [1: 0以下のtimeoutに30秒のdefaultを適用する, 2: helperを直接起動するcommand factoryを設定する, 3: clientを返す]
//	  引数: [timeout: helperの最大実行時間]
//	  戻り値: [client: Parser Backend Contract v1 helper client]
//	}
func NewHelperClient(timeout time.Duration) HelperClient {
	if timeout <= 0 {
		timeout = defaultHelperTimeout
	}
	return HelperClient{
		timeout:        timeout,
		maxStdoutBytes: defaultHelperStdoutLimit,
		maxStderrBytes: defaultHelperStderrLimit,
		createCommand: func(ctx context.Context, executablePath string) *exec.Cmd {
			return exec.CommandContext(ctx, executablePath)
		},
	}
}

//	{
//	  責務: [Parse: helperを起動し、契約を検証したCommon IRまたは正規化失敗を返す]
//	  処理: [1: request IDとJSON requestを作る, 2: timeout付きでhelperを直接起動する, 3: process失敗を分類する, 4: response versionとrequest IDを検証する, 5: helper errorまたはCommon IRを検証して返す]
//	  引数: [parentContext: 呼出元のcancel・deadline, executablePath: manifestから解決したhelper path, backendID: registry上の識別子, language: 解析対象language ID, source: parserへ渡す内容, sourcePath: 任意のsource path]
//	  戻り値: [payload: version検証済みCommon IR, error: processまたはprotocol failure]
//	}
func (client HelperClient) Parse(parentContext context.Context, executablePath string, backendID string, language string, source string, sourcePath *string) (commonir.Payload, error) {
	requestID, err := newRequestID()
	if err != nil {
		return commonir.Payload{}, newBackendError("failure", fmt.Sprintf("create parser request id: %v", err), backendID, false)
	}
	request := parseRequest{
		ContractVersion: ParserBackendContractVersion,
		RequestID:       requestID,
		Operation:       "parse",
		Language:        language,
		Source:          source,
		Path:            sourcePath,
	}
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return commonir.Payload{}, newBackendError("failure", fmt.Sprintf("encode parser request: %v", err), backendID, false)
	}

	// Timeoutはhelper process全体に適用し、呼出元の早いcancelもそのまま反映する。
	timeout := client.timeout
	if timeout <= 0 {
		timeout = defaultHelperTimeout
	}
	createCommand := client.createCommand
	if createCommand == nil {
		createCommand = func(ctx context.Context, helperPath string) *exec.Cmd {
			return exec.CommandContext(ctx, helperPath)
		}
	}
	runContext, cancel := context.WithTimeout(parentContext, timeout)
	defer cancel()
	command := createCommand(runContext, executablePath)
	command.Stdin = bytes.NewReader(append(requestBytes, '\n'))
	stdoutLimit := client.maxStdoutBytes
	if stdoutLimit <= 0 {
		stdoutLimit = defaultHelperStdoutLimit
	}
	stderrLimit := client.maxStderrBytes
	if stderrLimit <= 0 {
		stderrLimit = defaultHelperStderrLimit
	}
	standardOutput := boundedOutputBuffer{limit: stdoutLimit}
	standardError := boundedOutputBuffer{limit: stderrLimit}
	command.Stdout = &standardOutput
	command.Stderr = &standardError
	if err := command.Run(); err != nil {
		return commonir.Payload{}, classifyProcessError(runContext, err, standardError.String(), backendID)
	}
	if standardOutput.exceeded {
		return commonir.Payload{}, newBackendError(
			"protocol_error",
			fmt.Sprintf("Parser helper stdout exceeded the %d-byte limit.", stdoutLimit),
			backendID,
			false,
		)
	}

	response, err := decodeHelperResponse(standardOutput.Bytes(), requestID, backendID)
	if err != nil {
		return commonir.Payload{}, err
	}
	if !response.OK {
		return commonir.Payload{}, normalizeHelperError(response.Error, backendID)
	}
	return decodeHelperIR(response.IR, language, backendID)
}

//	{
//	  責務: [newRequestID: helper responseとの対応に使う一意なrequest IDを作る]
//	  処理: [1: 暗号学的乱数16 byteを生成する, 2: lowercase hexadecimalへ変換して返す]
//	  引数: []
//	  戻り値: [requestID: 32文字の一意ID, error: 乱数生成に失敗した場合の内容]
//	}
func newRequestID() (string, error) {
	var identifierBytes [16]byte
	if _, err := rand.Read(identifierBytes[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(identifierBytes[:]), nil
}

//	{
//	  責務: [classifyProcessError: helper起動・実行・終了失敗をbackend failureへ正規化する]
//	  処理: [1: deadline超過をretryable timeoutにする, 2: stderrまたはOS errorからmessageを作る, 3: failureを返す]
//	  引数: [runContext: 実行時context, processError: command.Runのerror, standardError: helperの標準error出力, backendID: 登録済みhelper ID]
//	  戻り値: [failure: 安定した分類とmessageを持つBackendError]
//	}
func classifyProcessError(runContext context.Context, processError error, standardError string, backendID string) *BackendError {
	if errors.Is(runContext.Err(), context.DeadlineExceeded) {
		return newBackendError("timeout", "Parser backend timed out.", backendID, true)
	}
	message := strings.TrimSpace(standardError)
	if message == "" {
		message = processError.Error()
	}
	return newBackendError("failure", message, backendID, false)
}

//	{
//	  責務: [decodeHelperResponse: stdoutを一つのParser Backend Contract v1 responseとして検証する]
//	  処理: [1: JSON objectをdecodeする, 2: contract versionとrequest IDを一致確認する, 3: responseを返す]
//	  引数: [responseBytes: helper stdout, expectedRequestID: 起動時に送ったrequest ID, backendID: 登録済みhelper ID]
//	  戻り値: [response: 検証済みresponse, error: JSON・version・request対応が不正な場合のprotocol error]
//	}
func decodeHelperResponse(responseBytes []byte, expectedRequestID string, backendID string) (helperResponse, error) {
	var response helperResponse
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return helperResponse{}, newBackendError("protocol_error", "Parser helper returned invalid JSON.", backendID, false)
	}
	if response.ContractVersion != ParserBackendContractVersion || response.RequestID != expectedRequestID {
		return helperResponse{}, newBackendError("protocol_error", "Parser helper response contract or request id does not match.", backendID, false)
	}
	if !response.OK && response.Error == nil {
		return helperResponse{}, newBackendError("protocol_error", "Parser helper response is missing error details.", backendID, false)
	}
	return response, nil
}

//	{
//	  責務: [normalizeHelperError: helper error kindを許可済み分類へ正規化してProcess callerへ返す]
//	  処理: [1: messageとkindの必須値を確認する, 2: 未知kindをprotocol errorにする, 3: backend IDをrequest元の値で固定して返す]
//	  引数: [helperFailure: helperが返したerror object, backendID: registry上のhelper ID]
//	  戻り値: [failure: 正規化されたBackendError]
//	}
func normalizeHelperError(helperFailure *helperError, backendID string) *BackendError {
	if helperFailure == nil || strings.TrimSpace(helperFailure.Message) == "" {
		return newBackendError("protocol_error", "Parser helper error response is incomplete.", backendID, false)
	}
	switch helperFailure.Kind {
	case "failure", "timeout", "unsupported_syntax", "unsupported_language", "protocol_error":
		return newBackendError(helperFailure.Kind, helperFailure.Message, backendID, helperFailure.Retryable)
	default:
		return newBackendError("protocol_error", "Parser helper returned an unsupported failure kind.", backendID, false)
	}
}

//	{
//	  責務: [decodeHelperIR: helper IR JSONをCommon IR schemaで検証し、要求languageのpayloadだけを返す]
//	  処理: [1: IR payloadの存在を確認する, 2: schema versionと各fieldをdecode・検証する, 3: language一致を確認する]
//	  引数: [irBytes: responseのIR object, expectedLanguage: requestで指定したlanguage, backendID: registry上のhelper ID]
//	  戻り値: [payload: 検証済みCommon IR, error: schemaまたはlanguage不正のprotocol error]
//	}
func decodeHelperIR(irBytes json.RawMessage, expectedLanguage string, backendID string) (commonir.Payload, error) {
	if len(irBytes) == 0 || string(irBytes) == "null" {
		return commonir.Payload{}, newBackendError("protocol_error", "Parser helper response is missing Common IR payload.", backendID, false)
	}
	payload, err := commonir.Decode(irBytes)
	if err != nil {
		return commonir.Payload{}, newBackendError("protocol_error", fmt.Sprintf("Parser helper Common IR payload is invalid: %v", err), backendID, false)
	}
	if payload.Language != expectedLanguage {
		return commonir.Payload{}, newBackendError("protocol_error", "Parser helper returned Common IR for the wrong language.", backendID, false)
	}
	return payload, nil
}

//	{
//	  責務: [newBackendError: protocol boundaryのfailure fieldsを一つのerror valueへまとめる]
//	  処理: [1: 安定したfailure情報を保持するBackendErrorを作る]
//	  引数: [kind: failure分類, message: 利用者向け説明, backendID: 登録済みhelper ID, retryable: 再試行可能性]
//	  戻り値: [error: 正規化されたBackendError]
//	}
func newBackendError(kind string, message string, backendID string, retryable bool) *BackendError {
	return &BackendError{Failure: BackendFailure{
		Kind: kind, Message: message, BackendID: backendID, Retryable: retryable,
	}}
}
