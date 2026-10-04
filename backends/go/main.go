package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

const backendID = "go-stdlib-types-helper"

type request struct {
	ContractVersion string `json:"contract_version"`
	RequestID       string `json:"request_id"`
	Operation       string `json:"operation"`
	Language        string `json:"language"`
	Source          string `json:"source"`
	Path            string `json:"path,omitempty"`
}

type response struct {
	ContractVersion string            `json:"contract_version"`
	RequestID       string            `json:"request_id"`
	OK              bool              `json:"ok"`
	IR              *commonir.Payload `json:"ir,omitempty"`
	Error           map[string]any    `json:"error,omitempty"`
}

func line(fset *token.FileSet, pos token.Pos) int { return fset.Position(pos).Line }

func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func expression(fset *token.FileSet, expr ast.Node) string {
	if expr == nil {
		return ""
	}
	var output bytes.Buffer
	if err := formatNode(&output, fset, expr); err != nil {
		return ""
	}
	return output.String()
}

func namedType(expr ast.Expr) string {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		return namedType(value.X)
	case *ast.IndexExpr:
		return namedType(value.X)
	case *ast.IndexListExpr:
		return namedType(value.X)
	case *ast.ParenExpr:
		return namedType(value.X)
	case *ast.SelectorExpr:
		return namedType(value.Sel)
	default:
		return ""
	}
}

func commentText(comment *ast.CommentGroup) *string {
	if comment == nil {
		return nil
	}
	text := strings.TrimSpace(comment.Text())
	return stringPointer(text)
}

func entity(kind, name string, fset *token.FileSet, node ast.Node) commonir.Entity {
	return commonir.Entity{
		Kind:           kind,
		Name:           name,
		SourceLocation: commonir.SourceLocation{Line: line(fset, node.Pos()), EndLine: line(fset, node.End())},
		Visibility:     visibility(name),
	}
}

func visibility(name string) string {
	if name != "" && ast.IsExported(name) {
		return "public"
	}
	return "internal"
}

func fail(req request, kind, message string) response {
	return response{"1", req.RequestID, false, nil, map[string]any{
		"kind": kind, "message": message, "backend_id": backendID, "retryable": false,
	}}
}

func parse(req request) response {
	if req.ContractVersion != "1" || req.RequestID == "" || req.Operation != "parse" || req.Language != "go" {
		return fail(req, "protocol_error", "unsupported Go parser backend request")
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, req.Path, req.Source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return fail(req, "unsupported_syntax", err.Error())
	}

	result := &commonir.Payload{
		SchemaVersion: commonir.SchemaVersion,
		Module: commonir.Module{
			Language:    "go",
			Entities:    []commonir.Entity{},
			Imports:     []string{},
			Diagnostics: []commonir.Diagnostic{},
		},
	}
	result.Module.ModuleDocstring = commentText(file.Doc)
	if file.Name != nil {
		pkg := entity("module", file.Name.Name, fset, file.Name)
		pkg.DeclarationKind = stringPointer("package")
		pkg.Docstring = commentText(file.Doc)
		result.Entities = append(result.Entities, pkg)
	}
	for _, spec := range file.Imports {
		path, unquoteErr := strconv.Unquote(spec.Path.Value)
		if unquoteErr != nil {
			return fail(req, "unsupported_syntax", "invalid import path: "+unquoteErr.Error())
		}
		result.Imports = append(result.Imports, path)
	}

	info := &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
		Scopes:     map[ast.Node]*types.Scope{},
	}
	config := types.Config{Error: func(typeErr error) {
		diagnostic := commonir.Diagnostic{Kind: "type_error", Message: typeErr.Error()}
		if positioned, ok := typeErr.(types.Error); ok {
			lineNumber := line(fset, positioned.Pos)
			diagnostic.Line = &lineNumber
		}
		result.Diagnostics = append(result.Diagnostics, diagnostic)
	}}
	_, _ = config.Check(packagePath(req.Path, file.Name.Name), fset, []*ast.File{file}, info)

	for _, declaration := range file.Decls {
		switch d := declaration.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch item := spec.(type) {
				case *ast.TypeSpec:
					appendType(result, fset, item, d.Doc, info)
				case *ast.ValueSpec:
					appendValues(result, fset, item, d.Tok, d.Doc, info)
				}
			}
		case *ast.FuncDecl:
			result.Entities = append(result.Entities, functionEntity(fset, d, info))
		}
	}
	return response{"1", req.RequestID, true, result, nil}
}

func packagePath(path, packageName string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	if filepath.Ext(path) == ".go" {
		path = filepath.ToSlash(filepath.Dir(path))
	}
	if path == "" || path == "." || strings.HasSuffix(path, ":") {
		return packageName
	}
	return path
}

func appendType(result *commonir.Payload, fset *token.FileSet, spec *ast.TypeSpec, groupDoc *ast.CommentGroup, info *types.Info) {
	kind, declarationKind := "class", "defined_type"
	var bases []string
	var members []commonir.Entity
	switch typeNode := spec.Type.(type) {
	case *ast.StructType:
		declarationKind = "struct"
		if typeNode.Fields != nil {
			for _, field := range typeNode.Fields.List {
				if len(field.Names) == 0 {
					if base := namedType(field.Type); base != "" {
						bases = append(bases, base)
					}
					continue
				}
				for _, name := range field.Names {
					member := entity("field", name.Name, fset, field)
					member.Parent = stringPointer(spec.Name.Name)
					member.TypeName = stringPointer(expression(fset, field.Type))
					member.Docstring = commentText(firstComment(field.Doc, field.Comment))
					members = append(members, member)
				}
			}
		}
	case *ast.InterfaceType:
		kind, declarationKind = "class", "interface"
		if typeNode.Methods != nil {
			for _, field := range typeNode.Methods.List {
				if len(field.Names) == 0 {
					if base := namedType(field.Type); base != "" {
						bases = append(bases, base)
					}
					continue
				}
				for _, name := range field.Names {
					member := entity("method", name.Name, fset, field)
					member.Parent = stringPointer(spec.Name.Name)
					member.Docstring = commentText(firstComment(field.Doc, field.Comment))
					if signature, ok := field.Type.(*ast.FuncType); ok {
						setSignature(&member, fset, signature)
					}
					members = append(members, member)
				}
			}
		}
	default:
		if spec.Assign.IsValid() {
			declarationKind = "type_alias"
		} else {
			declarationKind = "defined_type"
		}
	}
	if spec.Assign.IsValid() {
		declarationKind = "type_alias"
	}
	typeEntity := entity(kind, spec.Name.Name, fset, spec)
	typeEntity.DeclarationKind = stringPointer(declarationKind)
	typeEntity.Bases = bases
	typeEntity.Docstring = commentText(firstComment(spec.Doc, groupDoc))
	typeEntity.TypeName = stringPointer(expression(fset, spec.Type))
	if spec.TypeParams != nil {
		for _, parameter := range spec.TypeParams.List {
			constraint := expression(fset, parameter.Type)
			for _, name := range parameter.Names {
				typeEntity.TypeParameters = append(typeEntity.TypeParameters, name.Name)
				typeEntity.TypeConstraints = append(typeEntity.TypeConstraints, name.Name+" "+constraint)
			}
		}
	}
	if object := info.Defs[spec.Name]; object != nil {
		typeEntity.SymbolID = stringPointer(types.ObjectString(object, packageQualifier))
	}
	result.Entities = append(result.Entities, typeEntity)
	result.Entities = append(result.Entities, members...)
}

func appendValues(result *commonir.Payload, fset *token.FileSet, spec *ast.ValueSpec, tokenKind token.Token, groupDoc *ast.CommentGroup, info *types.Info) {
	for _, name := range spec.Names {
		value := entity("field", name.Name, fset, spec)
		value.DeclarationKind = stringPointer(strings.ToLower(tokenKind.String()))
		typeName := expression(fset, spec.Type)
		if typeName == "" {
			if object := info.Defs[name]; object != nil && object.Type() != nil {
				typeName = types.TypeString(object.Type(), packageQualifier)
			}
		}
		value.TypeName = stringPointer(typeName)
		value.Docstring = commentText(firstComment(spec.Doc, groupDoc))
		result.Entities = append(result.Entities, value)
	}
}

func functionEntity(fset *token.FileSet, declaration *ast.FuncDecl, info *types.Info) commonir.Entity {
	kind, parent := "function", ""
	if declaration.Recv != nil && len(declaration.Recv.List) > 0 {
		kind = "method"
		parent = namedType(declaration.Recv.List[0].Type)
	}
	method := entity(kind, declaration.Name.Name, fset, declaration)
	method.Parent = stringPointer(parent)
	method.DeclarationKind = stringPointer(kind)
	method.Docstring = commentText(declaration.Doc)
	if declaration.Type != nil {
		setSignature(&method, fset, declaration.Type)
	}
	if object := info.Defs[declaration.Name]; object != nil {
		method.SymbolID = stringPointer(types.ObjectString(object, packageQualifier))
	}
	method.Calls, method.CallSequence, method.ResolvedCalls = callsInBody(declaration.Body, info)
	return method
}

func setSignature(target *commonir.Entity, fset *token.FileSet, signature *ast.FuncType) {
	appendParameters := func(fields *ast.FieldList) {
		if fields == nil {
			return
		}
		for _, field := range fields.List {
			typeName := expression(fset, field.Type)
			if len(field.Names) == 0 {
				index := len(target.Parameters)
				name := fmt.Sprintf("arg%d", index)
				target.Parameters = append(target.Parameters, name)
				target.ParameterTypes = append(target.ParameterTypes, []string{name, typeName})
				continue
			}
			for _, name := range field.Names {
				target.Parameters = append(target.Parameters, name.Name)
				target.ParameterTypes = append(target.ParameterTypes, []string{name.Name, typeName})
			}
		}
	}
	appendParameters(signature.Params)
	if signature.Results != nil {
		var resultTypes []string
		for _, field := range signature.Results.List {
			resultType := expression(fset, field.Type)
			for range field.Names {
				resultTypes = append(resultTypes, resultType)
			}
			if len(field.Names) == 0 {
				resultTypes = append(resultTypes, resultType)
			}
		}
		switch len(resultTypes) {
		case 1:
			target.ReturnType = stringPointer(resultTypes[0])
		case 0:
		default:
			target.ReturnType = stringPointer("(" + strings.Join(resultTypes, ", ") + ")")
		}
	}
	if signature.TypeParams != nil {
		for _, parameter := range signature.TypeParams.List {
			constraint := expression(fset, parameter.Type)
			for _, name := range parameter.Names {
				target.TypeParameters = append(target.TypeParameters, name.Name)
				target.TypeConstraints = append(target.TypeConstraints, name.Name+" "+constraint)
			}
		}
	}
}

func callsInBody(body *ast.BlockStmt, info *types.Info) ([]string, []string, []string) {
	var calls, sequence, resolved []string
	seenCalls := map[string]struct{}{}
	if body == nil {
		return calls, sequence, resolved
	}
	ast.Inspect(body, func(node ast.Node) bool {
		if _, ok := node.(*ast.FuncLit); ok {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := callName(call.Fun)
		if name != "" {
			if _, exists := seenCalls[name]; !exists {
				calls = append(calls, name)
				seenCalls[name] = struct{}{}
			}
			sequence = append(sequence, name)
		}
		object := calledObject(call.Fun, info)
		if object != nil {
			resolved = append(resolved, types.ObjectString(object, packageQualifier))
		}
		return true
	})
	return calls, sequence, resolved
}

func callName(expression ast.Expr) string {
	switch function := expression.(type) {
	case *ast.Ident:
		return function.Name
	case *ast.SelectorExpr:
		return function.Sel.Name
	case *ast.IndexExpr:
		return callName(function.X)
	case *ast.IndexListExpr:
		return callName(function.X)
	case *ast.ParenExpr:
		return callName(function.X)
	default:
		return ""
	}
}

func calledObject(expression ast.Expr, info *types.Info) types.Object {
	switch function := expression.(type) {
	case *ast.Ident:
		return info.Uses[function]
	case *ast.SelectorExpr:
		if selection := info.Selections[function]; selection != nil {
			return selection.Obj()
		}
		return info.Uses[function.Sel]
	case *ast.IndexExpr:
		return calledObject(function.X, info)
	case *ast.IndexListExpr:
		return calledObject(function.X, info)
	case *ast.ParenExpr:
		return calledObject(function.X, info)
	default:
		return nil
	}
}

func packageQualifier(pkg *types.Package) string {
	return pkg.Path()
}

func firstComment(groups ...*ast.CommentGroup) *ast.CommentGroup {
	for _, group := range groups {
		if group != nil {
			return group
		}
	}
	return nil
}

func formatNode(output *bytes.Buffer, fset *token.FileSet, node ast.Node) error {
	return format.Node(output, fset, node)
}

func main() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var req request
	if err := json.Unmarshal(data, &req); err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(fail(request{}, "protocol_error", err.Error()))
		return
	}
	_ = json.NewEncoder(os.Stdout).Encode(parse(req))
}

