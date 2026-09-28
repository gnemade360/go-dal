package file

import (
	"fmt"
	"strings"

	"github.com/gnemade360/go-dal/dal/interfaces"
)

func matchesFilter(record map[string]interface{}, filter interfaces.Filter) bool {
	if isEmptyFilter(filter) {
		return true
	}

	if filter.Field != "" {
		if !matchField(record, filter) {
			return false
		}
	}

	for _, f := range filter.And {
		if !matchesFilter(record, f) {
			return false
		}
	}

	if len(filter.Or) > 0 {
		anyMatch := false
		for _, f := range filter.Or {
			if matchesFilter(record, f) {
				anyMatch = true
				break
			}
		}
		if !anyMatch {
			return false
		}
	}

	return true
}

func matchField(record map[string]interface{}, filter interfaces.Filter) bool {
	val, exists := record[filter.Field]

	switch strings.ToUpper(filter.Op) {
	case "=", "==", "EQ":
		return fmt.Sprintf("%v", val) == fmt.Sprintf("%v", filter.Value)
	case "!=", "<>", "NEQ":
		return fmt.Sprintf("%v", val) != fmt.Sprintf("%v", filter.Value)
	case ">", "GT":
		return compareValues(val, filter.Value) > 0
	case ">=", "GTE":
		return compareValues(val, filter.Value) >= 0
	case "<", "LT":
		return compareValues(val, filter.Value) < 0
	case "<=", "LTE":
		return compareValues(val, filter.Value) <= 0
	case "LIKE":
		return matchLike(fmt.Sprintf("%v", val), fmt.Sprintf("%v", filter.Value))
	case "IN":
		return matchIn(val, filter.Value)
	case "IS NULL":
		return !exists || val == nil
	case "IS NOT NULL":
		return exists && val != nil
	default:
		return fmt.Sprintf("%v", val) == fmt.Sprintf("%v", filter.Value)
	}
}

func matchLike(value, pattern string) bool {
	pattern = strings.ToLower(pattern)
	value = strings.ToLower(value)

	if strings.HasPrefix(pattern, "%") && strings.HasSuffix(pattern, "%") {
		return strings.Contains(value, pattern[1:len(pattern)-1])
	}
	if strings.HasPrefix(pattern, "%") {
		return strings.HasSuffix(value, pattern[1:])
	}
	if strings.HasSuffix(pattern, "%") {
		return strings.HasPrefix(value, pattern[:len(pattern)-1])
	}
	return value == pattern
}

func matchIn(value, candidates interface{}) bool {
	vs := fmt.Sprintf("%v", value)

	switch c := candidates.(type) {
	case []interface{}:
		for _, item := range c {
			if fmt.Sprintf("%v", item) == vs {
				return true
			}
		}
	case []string:
		for _, item := range c {
			if item == vs {
				return true
			}
		}
	}
	return false
}

func isEmptyFilter(f interfaces.Filter) bool {
	return f.Field == "" && len(f.And) == 0 && len(f.Or) == 0
}
