package generators

import (
	"sort"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
	"github.com/tomiya7688/tomiya_code_atlas/internal/models"
)

//	{
//	  責務: [BuildStaticSpecification: Common IRから静的仕様書の論理modelを組み立てる]
//	  引数: [module: 言語非依存の解析結果; sourceName: 表示用source名]
//	  戻り値: [models.StaticSpecification: 宣言と同一module内で解決した呼び出し関係]
//	}
func BuildStaticSpecification(module commonir.Module, sourceName string) models.StaticSpecification {
	qualifiedNames := make([]string, len(module.Entities))
	callables := make(map[string][]string)
	for index, entity := range module.Entities {
		qualifiedNames[index] = qualifiedEntityName(entity)
		if isCallable(entity.Kind) {
			callables[entity.Name] = append(callables[entity.Name], qualifiedNames[index])
			callables[qualifiedNames[index]] = append(callables[qualifiedNames[index]], qualifiedNames[index])
		}
	}

	resolvedCalls := make([][]string, len(module.Entities))
	calledBy := make(map[string]map[string]struct{})
	for index, entity := range module.Entities {
		resolvedCalls[index] = resolveEntityCalls(entity, callables)
		for _, target := range resolvedCalls[index] {
			if calledBy[target] == nil {
				calledBy[target] = make(map[string]struct{})
			}
			calledBy[target][qualifiedNames[index]] = struct{}{}
		}
	}

	declarations := make([]models.StaticDeclaration, 0, len(module.Entities))
	for index, entity := range module.Entities {
		calls := entity.CallSequence
		if len(calls) == 0 {
			calls = entity.Calls
		}
		declaration := models.StaticDeclaration{
			Kind: entity.Kind, Name: entity.Name, QualifiedName: qualifiedNames[index],
			Line: entity.Line, EndLine: entity.EndLine, Visibility: entity.Visibility,
			Bases: copyStrings(entity.Bases), Parameters: copyStrings(entity.Parameters),
			ParameterTypes: copyPairs(entity.ParameterTypes), TypeParameters: copyStrings(entity.TypeParameters),
			Calls: uniqueStrings(calls), ResolvedCalls: resolvedCalls[index],
		}
		if entity.DeclarationKind != nil {
			declaration.DeclarationKind = *entity.DeclarationKind
		}
		if entity.Docstring != nil {
			declaration.Docstring = *entity.Docstring
		}
		if entity.ReturnType != nil {
			declaration.ReturnType = *entity.ReturnType
		}
		for caller := range calledBy[qualifiedNames[index]] {
			declaration.CalledBy = append(declaration.CalledBy, caller)
		}
		sort.Strings(declaration.CalledBy)
		declarations = append(declarations, declaration)
	}

	result := models.StaticSpecification{
		SourceName: sourceName, Language: module.Language, Declarations: declarations,
	}
	if module.ModuleDocstring != nil {
		result.ModuleDocstring = *module.ModuleDocstring
	}
	return result
}

//	{
//	  責務: [resolveEntityCalls: 明示解決結果を優先し、曖昧でない同一module呼び出しを解決する]
//	  引数: [entity: 呼び出し元宣言; caller: 修飾名; callables: 同名宣言の索引]
//	  戻り値: [[]string: 重複のない解決先一覧]
//	}
func resolveEntityCalls(entity commonir.Entity, callables map[string][]string) []string {
	result := make([]string, 0, len(entity.ResolvedCalls))
	for _, target := range entity.ResolvedCalls {
		if isKnownCallable(target, callables) {
			result = append(result, target)
		}
	}
	if len(result) == 0 {
		calls := entity.CallSequence
		if len(calls) == 0 {
			calls = entity.Calls
		}
		for _, call := range calls {
			if target, ok := resolveCallTarget(entity, call, callables); ok {
				result = append(result, target)
			}
		}
	}
	return uniqueStrings(result)
}

//	{
//	  責務: [resolveCallTarget: 呼び出し式を同一moduleの一意なcallableへ対応付ける]
//	  引数: [entity: 呼び出し元; caller: 修飾名; call: 観測した式; callables: callable索引]
//	  戻り値: [string, bool: 解決先と解決可否]
//	}
func resolveCallTarget(entity commonir.Entity, call string, callables map[string][]string) (string, bool) {
	parts := strings.Split(call, ".")
	name := strings.TrimSpace(parts[len(parts)-1])
	if entity.Parent != nil && len(parts) > 1 && (parts[0] == "self" || parts[0] == "cls" || parts[0] == *entity.Parent) {
		candidate := *entity.Parent + "." + name
		if isKnownCallable(candidate, callables) {
			return candidate, true
		}
	}
	candidates := uniqueStrings(callables[name])
	if len(candidates) == 1 {
		return candidates[0], true
	}
	if entity.Parent != nil {
		var sameParent string
		for _, candidate := range candidates {
			if strings.HasPrefix(candidate, *entity.Parent+".") {
				if sameParent != "" && sameParent != candidate {
					return "", false
				}
				sameParent = candidate
			}
		}
		if sameParent != "" {
			return sameParent, true
		}
	}
	return "", false
}

//	{
//	  責務: [isKnownCallable: 名前が索引内のcallableとして一意に存在するか判定する]
//	  引数: [name: 修飾名; callables: callable索引]
//	  戻り値: [bool: callable存在判定]
//	}
func isKnownCallable(name string, callables map[string][]string) bool {
	return len(callables[name]) > 0
}

//	{
//	  責務: [isCallable: declaration kindが関数またはメソッドか判定する]
//	  引数: [kind: Common IR declaration kind]
//	  戻り値: [bool: callable判定]
//	}
func isCallable(kind string) bool {
	return kind == "function" || kind == "method"
}

//	{
//	  責務: [uniqueStrings: 初出順を保ったまま文字列の重複を取り除く]
//	  引数: [values: 重複を含む文字列一覧]
//	  戻り値: [[]string: 初出順の一意な値]
//	}
func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

//	{
//	  責務: [copyStrings: model用に文字列sliceを複製する]
//	  引数: [values: 元slice]
//	  戻り値: [[]string: 独立したslice]
//	}
func copyStrings(values []string) []string {
	return append([]string(nil), values...)
}

//	{
//	  責務: [copyPairs: model用にparameter typeのpairを複製する]
//	  引数: [values: 元pair slice]
//	  戻り値: [[][]string: 独立したpair slice]
//	}
func copyPairs(values [][]string) [][]string {
	result := make([][]string, len(values))
	for index, value := range values {
		result[index] = copyStrings(value)
	}
	return result
}
