/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nodeselector

// Source(04/2024): https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/types.go with modifications (NodeSelectorOpRegexp)

// NodeSelectorTerm represents expressions and fields required to select nodes.
// A null or empty node selector term matches no objects. The requirements of
// them are ANDed.
// The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.
//
//nolint:revive
type NodeSelectorTerm struct {
	// A list of node selector requirements by node's labels.
	MatchExpressions []NodeSelectorRequirement `yaml:"matchExpressions,omitempty"`
}

// NodeSelectorRequirement is a selector that contains values, a key, and an operator
// that relates the key and values.
//
//nolint:revive
type NodeSelectorRequirement struct {
	// The label key that the selector applies to.
	Key string `yaml:"key,omitempty"`
	// Represents a key's relationship to a set of values.
	// Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
	Operator NodeSelectorOperator `yaml:"operator,omitempty"`
	// An array of string values. If the operator is In or NotIn,
	// the values array must be non-empty. If the operator is Exists or DoesNotExist,
	// the values array must be empty. If the operator is Gt or Lt, the values
	// array must have a single element, which will be interpreted as an integer.
	// This array is replaced during a strategic merge patch.
	// +optional
	Values []string `yaml:"values,omitempty"`
}

// NodeSelectorOperator is the set of operators that can be used in
// a node selector requirement.
//
//nolint:revive
type NodeSelectorOperator string

// These are valid values of NodeSelectorOperator.
const (
	NodeSelectorOpIn           NodeSelectorOperator = "In"
	NodeSelectorOpNotIn        NodeSelectorOperator = "NotIn"
	NodeSelectorOpExists       NodeSelectorOperator = "Exists"
	NodeSelectorOpDoesNotExist NodeSelectorOperator = "DoesNotExist"
	NodeSelectorOpGt           NodeSelectorOperator = "Gt"
	NodeSelectorOpLt           NodeSelectorOperator = "Lt"
	NodeSelectorOpRegexp       NodeSelectorOperator = "Regexp"
)
