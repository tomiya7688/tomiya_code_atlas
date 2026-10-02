package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
)

type request struct {
	ContractVersion string `json:"contract_version"`
	RequestID       string `json:"request_id"`
	Operation       string `json:"operation"`
	Language        string `json:"language"`
	Source          string `json:"source"`
	Path            string `json:"path,omitempty"`
}
type entity struct {
	Kind    string   `json:"kind"`
	Name    string   `json:"name"`
	Line    int      `json:"line"`
	EndLine int      `json:"end_line"`
	Parent  string   `json:"parent,omitempty"`
	Bases   []string `json:"bases,omitempty"`
}
type ir struct {
	SchemaVersion string           `json:"schema_version"`
	Language      string           `json:"language"`
	Entities      []entity         `json:"entities"`
	Imports       []string         `json:"imports"`
	Diagnostics   []map[string]any `json:"diagnostics"`
}
type response struct {
	ContractVersion string         `json:"contract_version"`
	RequestID       string         `json:"request_id"`
	OK              bool           `json:"ok"`
	IR              *ir            `json:"ir,omitempty"`
	Error           map[string]any `json:"error,omitempty"`
}

func line(fset *token.FileSet, pos token.Pos) int { return fset.Position(pos).Line }
func typeName(expr ast.Expr) string {
	if id, ok := expr.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}
func parse(req request) response {
	if req.ContractVersion != "1" || req.Operation != "parse" || req.Language != "go" {
		return response{"1", req.RequestID, false, nil, map[string]any{"kind": "protocol_error", "message": "unsupported request", "backend_id": "go-stdlib-types-helper", "retryable": false}}
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, req.Path, req.Source, parser.ParseComments)
	if err != nil {
		return response{"1", req.RequestID, false, nil, map[string]any{"kind": "failure", "message": err.Error(), "backend_id": "go-stdlib-types-helper", "retryable": false}}
	}
	result := &ir{SchemaVersion: "1", Language: "go", Entities: []entity{}, Imports: []string{}, Diagnostics: []map[string]any{}}
	if file.Name != nil {
		result.Entities = append(result.Entities, entity{Kind: "module", Name: file.Name.Name, Line: line(fset, file.Name.Pos()), EndLine: line(fset, file.End())})
	}
	for _, spec := range file.Imports {
		result.Imports = append(result.Imports, spec.Path.Value)
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					kind := "class"
					if _, ok := ts.Type.(*ast.InterfaceType); ok {
						kind = "interface"
					}
					result.Entities = append(result.Entities, entity{Kind: kind, Name: ts.Name.Name, Line: line(fset, ts.Pos()), EndLine: line(fset, ts.End())})
				}
			}
		case *ast.FuncDecl:
			kind, parent := "function", ""
			if d.Recv != nil && len(d.Recv.List) > 0 {
				kind = "method"
				parent = typeName(d.Recv.List[0].Type)
			}
			result.Entities = append(result.Entities, entity{Kind: kind, Name: d.Name.Name, Line: line(fset, d.Pos()), EndLine: line(fset, d.End()), Parent: parent})
		}
	}
	conf := types.Config{Importer: importer.Default(), Error: func(err error) {
		result.Diagnostics = append(result.Diagnostics, map[string]any{"kind": "type_error", "message": err.Error()})
	}}
	_, _ = conf.Check(req.Path, fset, []*ast.File{file}, nil)
	return response{"1", req.RequestID, true, result, nil}
}

func main() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var req request
	if err := json.Unmarshal(data, &req); err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(response{"1", "", false, nil, map[string]any{"kind": "protocol_error", "message": err.Error(), "backend_id": "go-stdlib-types-helper", "retryable": false}})
		return
	}
	_ = json.NewEncoder(os.Stdout).Encode(parse(req))
}
