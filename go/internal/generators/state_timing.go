package generators

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

//	{
//	  責務: [
//	    StateDiagramTransition: 状態間の遷移条件とeventを保持する
//	  ]
//	  フィールド: [
//	    Source: 遷移元state
//	    Target: 遷移先state
//	    Event: 遷移を起こすevent
//	    Condition: 遷移条件
//	  ]
//	}
type StateDiagramTransition struct {
	Source    string  `json:"source"`
	Target    string  `json:"target"`
	Event     *string `json:"event,omitempty"`
	Condition *string `json:"condition,omitempty"`
}

//	{
//	  責務: [
//	    StateDiagram: Common IRから生成した状態図logical modelを保持する
//	  ]
//	  フィールド: [
//	    Name: 安定したdiagram名
//	    Owner: 状態機械の所有元
//	    StateType: 状態の型
//	    StateVariable: 状態変数名
//	    States: 状態一覧
//	    Transitions: 状態遷移一覧
//	    InitialState: 初期state
//	    TerminalStates: 終端state一覧
//	  ]
//	}
type StateDiagram struct {
	Name           string                   `json:"name"`
	Owner          string                   `json:"owner"`
	StateType      string                   `json:"state_type"`
	StateVariable  string                   `json:"state_variable"`
	States         []string                 `json:"states"`
	Transitions    []StateDiagramTransition `json:"transitions"`
	InitialState   *string                  `json:"initial_state,omitempty"`
	TerminalStates []string                 `json:"terminal_states"`
}

//	{
//	  責務: [
//	    StateDiagramStatistics: 状態図bundleの件数を集計する
//	  ]
//	  フィールド: [
//	    MachineCount: 状態機械の数
//	    StateCount: stateの合計数
//	    TransitionCount: transitionの合計数
//	  ]
//	}
type StateDiagramStatistics struct {
	MachineCount    int `json:"machine_count"`
	StateCount      int `json:"state_count"`
	TransitionCount int `json:"transition_count"`
}

//	{
//	  責務: [
//	    StateDiagramBundle: 状態図と集計値をまとめる
//	  ]
//	  フィールド: [
//	    Diagrams: 生成した状態図一覧
//	    Statistics: 状態図の集計値
//	  ]
//	}
type StateDiagramBundle struct {
	Diagrams   []StateDiagram         `json:"diagrams"`
	Statistics StateDiagramStatistics `json:"statistics"`
}

//	{
//	  責務: [
//	    TimingChartEvent: timing flow内の一つの処理eventを保持する
//	  ]
//	  フィールド: [
//	    Order: flow内の順序
//	    Kind: eventの種類
//	    Line: source上の行番号
//	    Target: 対象名
//	    Detail: eventの補足情報
//	  ]
//	}
type TimingChartEvent struct {
	Order  int     `json:"order"`
	Kind   string  `json:"kind"`
	Line   int     `json:"line"`
	Target *string `json:"target,omitempty"`
	Detail *string `json:"detail,omitempty"`
}

//	{
//	  責務: [
//	    TimingChart: ordered Common IR factsから作るtiming chartを保持する
//	  ]
//	  フィールド: [
//	    Name: chart名
//	    Owner: 処理flowの所有元
//	    IsAsync: 非同期flowかどうか
//	    Events: 順序付きevent一覧
//	  ]
//	}
type TimingChart struct {
	Name    string             `json:"name"`
	Owner   string             `json:"owner"`
	IsAsync bool               `json:"is_async"`
	Events  []TimingChartEvent `json:"events"`
}

//	{
//	  責務: [
//	    TimingChartStatistics: timing chartの件数とevent種別を集計する
//	  ]
//	  フィールド: [
//	    ChartCount: chart数
//	    AsyncFlowCount: 非同期flow数
//	    AwaitCount: await数
//	    ParallelStartCount: parallel start数
//	    ParallelJoinCount: parallel join数
//	    WaitCount: wait数
//	    SyncCount: sync数
//	    TimerCount: timer数
//	    CallbackCount: callback数
//	    PeriodicCount: periodic数
//	  ]
//	}
type TimingChartStatistics struct {
	ChartCount         int `json:"chart_count"`
	AsyncFlowCount     int `json:"async_flow_count"`
	AwaitCount         int `json:"await_count"`
	ParallelStartCount int `json:"parallel_start_count"`
	ParallelJoinCount  int `json:"parallel_join_count"`
	WaitCount          int `json:"wait_count"`
	SyncCount          int `json:"sync_count"`
	TimerCount         int `json:"timer_count"`
	CallbackCount      int `json:"callback_count"`
	PeriodicCount      int `json:"periodic_count"`
}

//	{
//	  責務: [
//	    TimingChartBundle: timing chartと集計値をまとめる
//	  ]
//	  フィールド: [
//	    Charts: 生成したchart一覧
//	    Statistics: chartとeventの集計値
//	  ]
//	}
type TimingChartBundle struct {
	Charts     []TimingChart         `json:"charts"`
	Statistics TimingChartStatistics `json:"statistics"`
}

var safeOutputChars = regexp.MustCompile(`[^A-Za-z0-9_-]+`)
var safeTimingChars = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

//	{
//	  責務: [
//	    BuildStateDiagramBundle: Common IRのstate factsを状態図logical modelへ変換する
//	  ]
//	  処理: [
//	    1: state machineごとに安定したdiagram名を作る
//	    2: state、transition、初期・終端stateをコピーする
//	    3: diagramと統計をbundleで返す
//	  ]
//	  引数: [
//	    module: 解析済みCommon IR
//	  ]
//	  戻り値: [
//	    StateDiagramBundle: 生成した状態図と件数
//	  ]
//	}
func BuildStateDiagramBundle(module commonir.Module) StateDiagramBundle {
	result := StateDiagramBundle{Diagrams: []StateDiagram{}}
	// Common IRの状態機械ごとに、出力形式から独立したdiagram modelを作ります。
	for _, machine := range module.StateMachines {
		logicalName := machine.Owner + "." + machine.StateVariable
		// transitionとstate一覧をコピーし、元のCommon IR sliceを共有しません。
		transitions := make([]StateDiagramTransition, 0, len(machine.Transitions))
		for _, transition := range machine.Transitions {
			transitions = append(transitions, StateDiagramTransition{
				Source: transition.Source, Target: transition.Target,
				Event: transition.Event, Condition: transition.Condition,
			})
		}
		states := append([]string{}, machine.States...)
		terminalStates := append([]string{}, machine.TerminalStates...)
		result.Diagrams = append(result.Diagrams, StateDiagram{
			Name:  stableOutputName("state", logicalName, "state_machine"),
			Owner: machine.Owner, StateType: machine.StateType,
			StateVariable: machine.StateVariable, States: states,
			Transitions: transitions, InitialState: machine.InitialState,
			TerminalStates: terminalStates,
		})
		// Bundleの件数は出力したdiagramの実体から集計します。
		result.Statistics.MachineCount++
		result.Statistics.StateCount += len(machine.States)
		result.Statistics.TransitionCount += len(machine.Transitions)
	}
	return result
}

//	{
//	  責務: [
//	    BuildTimingChartBundle: Common IRのtiming factsをchartと集計値へ変換する
//	  ]
//	  処理: [
//	    1: eventのないflowを除外する
//	    2: event順序を保ってchartへコピーし種類別件数を集計する
//	    3: chartと統計をbundleで返す
//	  ]
//	  引数: [
//	    module: 解析済みCommon IR
//	  ]
//	  戻り値: [
//	    TimingChartBundle: 生成したchartと集計値
//	  ]
//	}
func BuildTimingChartBundle(module commonir.Module) TimingChartBundle {
	result := TimingChartBundle{Charts: []TimingChart{}}
	// TimingFlowごとに、順序付きeventと種類別統計を一度に組み立てます。
	for _, flow := range module.TimingFlows {
		if len(flow.Events) == 0 {
			continue
		}
		events := make([]TimingChartEvent, 0, len(flow.Events))
		// eventの並びを保ったままchartへ移し、同じ走査中に件数を数えます。
		for _, event := range flow.Events {
			events = append(events, TimingChartEvent{Order: event.Order, Kind: event.Kind, Line: event.Line, Target: event.Target, Detail: event.Detail})
			switch event.Kind {
			case "await":
				result.Statistics.AwaitCount++
			case "parallel_start":
				result.Statistics.ParallelStartCount++
			case "parallel_join":
				result.Statistics.ParallelJoinCount++
			case "wait":
				result.Statistics.WaitCount++
			case "sync":
				result.Statistics.SyncCount++
			case "timer":
				result.Statistics.TimerCount++
			case "callback":
				result.Statistics.CallbackCount++
			case "periodic":
				result.Statistics.PeriodicCount++
			}
		}
		result.Charts = append(result.Charts, TimingChart{Name: safeTimingName(flow.Owner), Owner: flow.Owner, IsAsync: flow.IsAsync, Events: events})
		result.Statistics.ChartCount++
		if flow.IsAsync {
			result.Statistics.AsyncFlowCount++
		}
	}
	return result
}

//	{
//	  責務: [
//	    stableOutputName: logical nameから衝突しにくい安定output名を作る
//	  ]
//	  処理: [
//	    1: 表示可能な文字へ置換して長さを制限する
//	    2: 空名にfallbackを使う
//	    3: logical nameのhashを含む名前を返す
//	  ]
//	  引数: [
//	    prefix: output種別の接頭辞
//	    logicalName: 元となる完全名
//	    fallback: 表示名が空の場合の名前
//	  ]
//	  戻り値: [
//	    string: 安定したoutput名
//	  ]
//	}
func stableOutputName(prefix, logicalName, fallback string) string {
	// 読みやすい名前部分を作り、空名や長すぎる名前を制限します。
	readable := strings.Trim(safeOutputChars.ReplaceAllString(logicalName, "_"), "_-")
	if readable == "" {
		readable = fallback
	}
	if len(readable) > 80 {
		readable = strings.TrimRight(readable[:80], "_-")
	}
	if readable == "" {
		readable = fallback
	}
	// 切り詰めで名前が重なるのを避けるため、元名のhashを含めます。
	digest := sha256.Sum256([]byte(logicalName))
	return fmt.Sprintf("%s_%x_%s", prefix, digest[:5], readable)
}

//	{
//	  責務: [
//	    safeTimingName: timing chart名を利用可能な文字へ正規化する
//	  ]
//	  処理: [
//	    1: 許可文字以外をunderscoreへ置換する
//	    2: 空名には既定名を返す
//	  ]
//	  引数: [
//	    value: 正規化する元の名前
//	  ]
//	  戻り値: [
//	    string: 利用可能なchart名
//	  ]
//	}
func safeTimingName(value string) string {
	// chart名に使えない文字を置き換え、空文字には固定名を使います。
	name := strings.Trim(safeTimingChars.ReplaceAllString(value, "_"), "_.-")
	if name == "" {
		return "timing"
	}
	return name
}

