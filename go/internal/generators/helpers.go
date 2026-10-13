package generators

import (
	"sort"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

func qualifiedEntityName(entity commonir.Entity) string {
	if entity.Parent != nil && *entity.Parent != "" {
		return *entity.Parent + "." + entity.Name
	}
	return entity.Name
}

func objectKey(instance commonir.ObjectInstance) string {
	scope := "<module>"
	if instance.Scope != nil && *instance.Scope != "" {
		scope = *instance.Scope
	}
	return scope + ":" + instance.Name
}

func stringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func sortedSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
