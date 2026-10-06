package generators

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

const stateTimingFixturePath = "../../../tests/fixtures/analyzers/state_timing_v1.json"

func TestStateAndTimingBundlesMatchSharedGolden(t *testing.T) {
	var fixture struct {
		CommonIR commonir.Module `json:"common_ir"`
		Expected struct {
			State  StateDiagramBundle `json:"state"`
			Timing TimingChartBundle  `json:"timing"`
		} `json:"expected"`
	}
	data, err := os.ReadFile(stateTimingFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}

	if got := BuildStateDiagramBundle(fixture.CommonIR); !reflect.DeepEqual(got, fixture.Expected.State) {
		t.Fatalf("state bundle = %#v", got)
	}
	if got := BuildTimingChartBundle(fixture.CommonIR); !reflect.DeepEqual(got, fixture.Expected.Timing) {
		t.Fatalf("timing bundle = %#v", got)
	}
}

func TestStateAndTimingBundlesDoNotAliasCommonIRSlices(t *testing.T) {
	module := commonir.Module{StateMachines: []commonir.StateMachine{{Owner: "Worker", States: []string{"IDLE"}}}, TimingFlows: []commonir.TimingFlow{{Owner: "load", Events: []commonir.TimingEvent{{Order: 1, Kind: "await", Line: 2}}}}}
	state := BuildStateDiagramBundle(module)
	timing := BuildTimingChartBundle(module)
	state.Diagrams[0].States[0] = "MUTATED"
	timing.Charts[0].Events[0].Kind = "MUTATED"
	if module.StateMachines[0].States[0] != "IDLE" || module.TimingFlows[0].Events[0].Kind != "await" {
		t.Fatal("logical output aliases Common IR slices")
	}
}

