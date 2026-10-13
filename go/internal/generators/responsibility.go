package generators

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/analyzers"
	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

var camelWordBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

type ResponsibilityRow struct {
	ClassName      string `json:"class_name"`
	Responsibility string `json:"responsibility"`
}

type ResponsibilityTable struct {
	Name string              `json:"name"`
	Rows []ResponsibilityRow `json:"rows"`
}

type ResponsibilityTableBundle struct {
	Tables     []ResponsibilityTable `json:"tables"`
	Statistics map[string]any        `json:"statistics"`
	Placements []OutputPlacement     `json:"placements,omitempty"`
}

//	{
//	  責務: [ResponsibilityRows: classのdocstringまたはmethod名から決定的な責務文を生成する]
//	  引数: [module: classとmethodを含むCommon IR]
//	  戻り値: [[]ResponsibilityRow: source orderを保つclass責務]
//	}
func ResponsibilityRows(module commonir.Module) []ResponsibilityRow {
	result := []ResponsibilityRow{}
	for _, classEntity := range module.Entities {
		if classEntity.Kind != "class" {
			continue
		}
		className := qualifiedEntityName(classEntity)
		methods := []commonir.Entity{}
		for _, entity := range module.Entities {
			if entity.Kind == "method" && entity.Parent != nil && *entity.Parent == className {
				methods = append(methods, entity)
			}
		}
		result = append(result, ResponsibilityRow{ClassName: className, Responsibility: describeResponsibility(classEntity, methods)})
	}
	return result
}

func describeResponsibility(entity commonir.Entity, methods []commonir.Entity) string {
	if entity.Docstring != nil && strings.TrimSpace(*entity.Docstring) != "" {
		line := strings.Split(strings.TrimSpace(*entity.Docstring), "\n")[0]
		return strings.TrimSpace(line)
	}
	normalized := camelWordBoundary.ReplaceAllString(entity.Name, `${1}_${2}`)
	parts := []string{}
	for _, item := range strings.Split(strings.Trim(normalized, "_"), "_") {
		if item != "" {
			parts = append(parts, strings.ToLower(item))
		}
	}
	words := strings.Join(parts, " ")
	names := map[string]bool{}
	for _, method := range methods {
		names[strings.ToLower(method.Name)] = true
	}
	if anyName(names, "load", "read", "save", "write") {
		return fmt.Sprintf("Handles %s data input and output.", words)
	}
	if anyName(names, "update", "process", "run", "execute") {
		return fmt.Sprintf("Coordinates %s processing.", words)
	}
	if anyName(names, "validate", "check", "evaluate") {
		return fmt.Sprintf("Validates or evaluates %s state.", words)
	}
	return fmt.Sprintf("Manages %s state and behavior.", words)
}

func anyName(names map[string]bool, candidates ...string) bool {
	for _, candidate := range candidates {
		if names[candidate] {
			return true
		}
	}
	return false
}

//	{
//	  責務: [BuildResponsibilityTableBundle: class relation partitionに沿って責務表を分割する]
//	}
func BuildResponsibilityTableBundle(module commonir.Module, fanInThreshold int) (ResponsibilityTableBundle, error) {
	rows := ResponsibilityRows(module)
	byName := map[string]ResponsibilityRow{}
	for _, row := range rows {
		byName[row.ClassName] = row
	}
	classGraph := analyzers.BuildClassRelationGraph(module)
	pairs := [][2]string{}
	for _, edge := range classGraph.Edges() {
		if _, sourceOK := byName[edge.Caller]; !sourceOK {
			continue
		}
		if _, targetOK := byName[edge.Callee]; !targetOK {
			continue
		}
		pairs = append(pairs, [2]string{edge.Caller, edge.Callee})
	}
	nodes := make([]string, 0, len(byName))
	for name := range byName {
		nodes = append(nodes, name)
	}
	sort.Strings(nodes)
	graph, err := relationGraph(nodes, pairs)
	if err != nil {
		return ResponsibilityTableBundle{}, err
	}
	partition, err := analyzers.PartitionGraph(graph, fanInThreshold, 3)
	if err != nil {
		return ResponsibilityTableBundle{}, err
	}
	result := ResponsibilityTableBundle{Statistics: partitionStatistics(partition)}
	for i, series := range partition.Series {
		selected := []ResponsibilityRow{}
		for _, name := range series {
			if row, ok := byName[name]; ok {
				selected = append(selected, row)
			}
		}
		if len(selected) == 0 {
			continue
		}
		name := stableOutputName(fmt.Sprintf("series_%d", i+1), partition.SeriesRoots[i], "responsibility")
		result.Tables = append(result.Tables, ResponsibilityTable{Name: name, Rows: selected})
		result.Placements = append(result.Placements, regularPlacement(name, partition, i))
	}
	for _, shared := range partition.Shared {
		row, ok := byName[shared]
		if !ok {
			continue
		}
		name := stableOutputName("shared", shared, "responsibility")
		result.Tables = append(result.Tables, ResponsibilityTable{Name: name, Rows: []ResponsibilityRow{row}})
		result.Placements = append(result.Placements, sharedPlacement(name, shared))
	}
	return result, nil
}
