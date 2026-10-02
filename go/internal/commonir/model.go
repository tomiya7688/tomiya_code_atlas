// Package commonir defines the language-neutral, versioned JSON model shared
// by parser backends and the Go application.
package commonir

// SchemaVersion is the current Common IR wire schema version.
const SchemaVersion = "1"

// SourceLocation is a one-based, inclusive source range.
// It is embedded in Entity so JSON keeps the v1 line/end_line wire shape.
type SourceLocation struct {
	Line    int `json:"line"`
	EndLine int `json:"end_line"`
}

// Entity is a passive language-neutral declaration fact.
type Entity struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	SourceLocation
	Indent          int        `json:"indent,omitempty"`
	Parent          *string    `json:"parent,omitempty"`
	Docstring       *string    `json:"docstring,omitempty"`
	Parameters      []string   `json:"parameters,omitempty"`
	Decorators      []string   `json:"decorators,omitempty"`
	Calls           []string   `json:"calls,omitempty"`
	CallSequence    []string   `json:"call_sequence,omitempty"`
	Visibility      string     `json:"visibility,omitempty"`
	Bases           []string   `json:"bases,omitempty"`
	DeclarationKind *string    `json:"declaration_kind,omitempty"`
	TypeParameters  []string   `json:"type_parameters,omitempty"`
	TypeConstraints []string   `json:"type_constraints,omitempty"`
	ResolvedCalls   []string   `json:"resolved_calls,omitempty"`
	SymbolID        *string    `json:"symbol_id,omitempty"`
	IsAsync         bool       `json:"is_async,omitempty"`
	ParameterTypes  [][]string `json:"parameter_types,omitempty"`
	ReturnType      *string    `json:"return_type,omitempty"`
}

// ObjectInstance is a passive object construction/reference fact.
type ObjectInstance struct {
	Name       string     `json:"name"`
	TypeName   string     `json:"type_name"`
	Line       int        `json:"line"`
	Scope      *string    `json:"scope,omitempty"`
	Values     [][]string `json:"values,omitempty"`
	References [][]string `json:"references,omitempty"`
}

// StateTransition is a passive edge between named states.
type StateTransition struct {
	Source    string  `json:"source"`
	Target    string  `json:"target"`
	Line      int     `json:"line"`
	Event     *string `json:"event,omitempty"`
	Condition *string `json:"condition,omitempty"`
}

// StateMachine contains normalized declarations without execution behavior.
type StateMachine struct {
	Owner          string            `json:"owner"`
	StateType      string            `json:"state_type"`
	StateVariable  string            `json:"state_variable"`
	States         []string          `json:"states"`
	Transitions    []StateTransition `json:"transitions,omitempty"`
	InitialState   *string           `json:"initial_state,omitempty"`
	TerminalStates []string          `json:"terminal_states,omitempty"`
}

// TimingEvent is an ordered, normalized timing fact.
type TimingEvent struct {
	Order  int     `json:"order"`
	Kind   string  `json:"kind"`
	Line   int     `json:"line"`
	Target *string `json:"target,omitempty"`
	Detail *string `json:"detail,omitempty"`
}

// TimingFlow groups timing facts for one owner.
type TimingFlow struct {
	Owner   string        `json:"owner"`
	IsAsync bool          `json:"is_async"`
	Events  []TimingEvent `json:"events"`
}

// InputEvent records a normalized input-to-handler registration.
type InputEvent struct {
	Actor     string  `json:"actor"`
	Component string  `json:"component"`
	Event     string  `json:"event"`
	Handler   string  `json:"handler"`
	Line      int     `json:"line"`
	Label     *string `json:"label,omitempty"`
	Scope     *string `json:"scope,omitempty"`
}

// Signal is a normalized event declaration.
type Signal struct {
	Name       string   `json:"name"`
	Line       int      `json:"line"`
	Owner      *string  `json:"owner,omitempty"`
	Parameters []string `json:"parameters,omitempty"`
}

// Diagnostic is a normalized parser or semantic diagnostic.
type Diagnostic struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
	Line    *int   `json:"line,omitempty"`
}

// Module is the passive Common IR payload. Optional collections may be absent
// in helper responses; the required v1 collections are validated by Decode.
type Module struct {
	Language        string           `json:"language"`
	Entities        []Entity         `json:"entities"`
	Objects         []ObjectInstance `json:"objects,omitempty"`
	StateMachines   []StateMachine   `json:"state_machines,omitempty"`
	TimingFlows     []TimingFlow     `json:"timing_flows,omitempty"`
	InputEvents     []InputEvent     `json:"input_events,omitempty"`
	Signals         []Signal         `json:"signals,omitempty"`
	Diagnostics     []Diagnostic     `json:"diagnostics"`
	Imports         []string         `json:"imports"`
	ModuleDocstring *string          `json:"module_docstring,omitempty"`
}

// Payload is a helper response IR object with schema_version at the root.
type Payload struct {
	SchemaVersion string `json:"schema_version"`
	Module
}

// Relation is a renderer-neutral logical call-graph edge.
type Relation struct {
	Caller   string `json:"caller"`
	Callee   string `json:"callee"`
	CallType string `json:"call_type"`
}

// CallGraph contains logical node names and deduplicated relation facts.
type CallGraph struct {
	Nodes []string   `json:"nodes"`
	Edges []Relation `json:"edges"`
}

// LogicalOutput groups language-neutral generator input models.
type LogicalOutput struct {
	CallGraph CallGraph `json:"call_graph"`
}

// Fixture is the Python-produced serialized Common IR v1 golden wrapper.
type Fixture struct {
	SchemaVersion     string        `json:"schema_version"`
	Source            string        `json:"source"`
	CommonIR          Module        `json:"common_ir"`
	DiagnosticExample Diagnostic    `json:"diagnostic_example"`
	LogicalOutput     LogicalOutput `json:"logical_output"`
}
