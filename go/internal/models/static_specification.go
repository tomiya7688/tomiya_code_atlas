package models

//	{
//	  責務: [StaticDeclaration: 静的仕様書に載せる宣言情報と関係を保持する]
//	  フィールド: [Kind: 宣言種別; Name: qualified name; Line/EndLine: 定義範囲; Visibility: 公開範囲; Bases: 継承先; Signature: 型注釈付き宣言; Docstring: 既存説明; Calls: 観測した呼び出し; ResolvedCalls: 一意に解決した呼び出し先; CalledBy: 一意に解決した呼び出し元]
//	}
type StaticDeclaration struct {
	Kind            string     `json:"kind"`
	DeclarationKind string     `json:"declaration_kind,omitempty"`
	Name            string     `json:"name"`
	QualifiedName   string     `json:"qualified_name"`
	Line            int        `json:"line"`
	EndLine         int        `json:"end_line"`
	Visibility      string     `json:"visibility"`
	Bases           []string   `json:"bases,omitempty"`
	Parameters      []string   `json:"parameters,omitempty"`
	ParameterTypes  [][]string `json:"parameter_types,omitempty"`
	ReturnType      string     `json:"return_type,omitempty"`
	TypeParameters  []string   `json:"type_parameters,omitempty"`
	Docstring       string     `json:"docstring,omitempty"`
	Calls           []string   `json:"calls,omitempty"`
	ResolvedCalls   []string   `json:"resolved_calls,omitempty"`
	CalledBy        []string   `json:"called_by,omitempty"`
}

//	{
//	  責務: [StaticSpecification: source moduleの宣言一覧をrenderer非依存で保持する]
//	  フィールド: [SourceName: 表示するsource名; Language: Common IR language; ModuleDocstring: module説明; Declarations: source順の宣言model]
//	}
type StaticSpecification struct {
	SourceName      string              `json:"source_name"`
	Language        string              `json:"language"`
	ModuleDocstring string              `json:"module_docstring,omitempty"`
	Declarations    []StaticDeclaration `json:"declarations"`
}
