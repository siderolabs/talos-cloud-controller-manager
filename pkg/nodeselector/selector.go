// Package nodeselector provides a mechanism to match node based on node selector rules.
package nodeselector

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Match returns true if the node metadata matches the node selector rules.
func Match(rules []NodeSelectorTerm, fields map[string]string) (bool, error) {
	if len(rules) == 0 {
		return true, nil
	}

	for _, rule := range rules {
		match, err := MatchExpressions(rule.MatchExpressions, fields)
		if err != nil {
			return false, err
		}

		if match {
			return true, nil
		}
	}

	return false, nil
}

// MatchExpressions returns true if the node metadata matches the node selector expressions.
//
//nolint:cyclop,gocyclo
func MatchExpressions(rules []NodeSelectorRequirement, fields map[string]string) (bool, error) {
	if len(rules) == 0 {
		return false, nil
	}

	matchs := make([]bool, len(rules))

	for idx, rule := range rules {
		switch rule.Operator {
		case NodeSelectorOpIn:
			if len(rule.Values) == 0 {
				return false, fmt.Errorf("values must be non-empty for operator '%s'", rule.Operator)
			}

			if value, ok := fields[strings.ToLower(rule.Key)]; ok {
				matchs[idx] = slices.Contains(rule.Values, value)
			}

		case NodeSelectorOpNotIn:
			if len(rule.Values) == 0 {
				return false, fmt.Errorf("values must be non-empty for operator '%s'", rule.Operator)
			}

			if value, ok := fields[strings.ToLower(rule.Key)]; ok {
				matchs[idx] = !slices.Contains(rule.Values, value)
			}

		case NodeSelectorOpExists:
			if len(rule.Values) > 0 {
				return false, fmt.Errorf("values must be empty for operator %s", rule.Operator)
			}

			if _, ok := fields[strings.ToLower(rule.Key)]; ok {
				matchs[idx] = true
			}

		case NodeSelectorOpDoesNotExist:
			if len(rule.Values) > 0 {
				return false, fmt.Errorf("values must be empty for operator %s", rule.Operator)
			}

			if _, ok := fields[strings.ToLower(rule.Key)]; !ok {
				matchs[idx] = true
			}

		case NodeSelectorOpGt, NodeSelectorOpLt:
			if len(rule.Values) != 1 {
				return false, fmt.Errorf("values must have a single element for operator %s", rule.Operator)
			}

			if value, ok := fields[strings.ToLower(rule.Key)]; ok {
				lsValue, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return false, fmt.Errorf("failed to parse value %s as int", value)
				}

				rValue, err := strconv.ParseInt(rule.Values[0], 10, 64)
				if err != nil {
					return false, fmt.Errorf("failed to parse value %s as int", rule.Values[0])
				}

				matchs[idx] = (rule.Operator == NodeSelectorOpGt && lsValue > rValue) || (rule.Operator == NodeSelectorOpLt && lsValue < rValue)
			}

		case NodeSelectorOpRegexp:
			if len(rule.Values) != 1 {
				return false, fmt.Errorf("values must have a single element for operator %s", rule.Operator)
			}

			if value, ok := fields[strings.ToLower(rule.Key)]; ok {
				r := regexp.MustCompile(rule.Values[0])

				matchs[idx] = r.MatchString(value)
			}

		default:
			return false, fmt.Errorf("%s not a valid selector operator", rule.Operator)
		}
	}

	// And operation between all matchs
	for _, match := range matchs {
		if !match {
			return false, nil
		}
	}

	return true, nil
}
