package generators

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

type StateDiagramTransition struct {
	Source    string  `json:"source"`
	Target    string  `json:"target"`
	Event     *string `json:"event,omitempty"`
	Condition *string `json:"condition,omitempty"`
}

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

type StateDiagramStatistics struct {
	MachineCount    int `json:"machine_count"`
	StateCount      int `json:"state_count"`
	TransitionCount int `json:"transition_count"`
}

type StateDiagramBundle struct {
	Diagrams   []StateDiagram         `json:"diagrams"`
	Statistics StateDiagramStatistics `json:"statistics"`
}

type TimingChartEvent struct {
	Order  int     `json:"order"`
	Kind   string  `json:"kind"`
	Line   int     `json:"line"`
	Target *string `json:"target,omitempty"`
	Detail *string `json:"detail,omitempty"`
}

type TimingChart struct {
	Name    string             `json:"name"`
	Owner   string             `json:"owner"`
	IsAsync bool               `json:"is_async"`
	Events  []TimingChartEvent `json:"events"`
}

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

type TimingChartBundle struct {
	Charts     []TimingChart         `json:"charts"`
	Statistics TimingChartStatistics `json:"statistics"`
}

var safeOutputChars = regexp.MustCompile(`[^A-Za-z0-9_-]+`)
var safeTimingChars = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

// BuildStateDiagramBundle maps passive Common IR state facts to logical diagrams.
func BuildStateDiagramBundle(module commonir.Module) StateDiagramBundle {
	result := StateDiagramBundle{Diagrams: []StateDiagram{}}
	for _, machine := range module.StateMachines {
		logicalName := machine.Owner + "." + machine.StateVariable
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
		result.Statistics.MachineCount++
		result.Statistics.StateCount += len(machine.States)
		result.Statistics.TransitionCount += len(machine.Transitions)
	}
	return result
}

// BuildTimingChartBundle maps ordered Common IR timing facts to logical charts.
func BuildTimingChartBundle(module commonir.Module) TimingChartBundle {
	result := TimingChartBundle{Charts: []TimingChart{}}
	for _, flow := range module.TimingFlows {
		if len(flow.Events) == 0 {
			continue
		}
		events := make([]TimingChartEvent, 0, len(flow.Events))
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

func stableOutputName(prefix, logicalName, fallback string) string {
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
	digest := sha256.Sum256([]byte(logicalName))
	return fmt.Sprintf("%s_%x_%s", prefix, digest[:5], readable)
}

func safeTimingName(value string) string {
	name := strings.Trim(safeTimingChars.ReplaceAllString(value, "_"), "_.-")
	if name == "" {
		return "timing"
	}
	return name
}

