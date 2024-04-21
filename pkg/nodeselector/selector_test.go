package nodeselector_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/siderolabs/talos-cloud-controller-manager/pkg/nodeselector"
)

func TestMatch(t *testing.T) {
	fields := map[string]string{
		"platform": "test-platform",
		"hostname": "test-hostname",
		"region":   "region-a",
		"int":      "10",
	}

	for _, tt := range []struct {
		name          string
		rules         []nodeselector.NodeSelectorTerm
		fields        map[string]string
		expected      bool
		expectedError error
	}{
		{
			name:     "NotMatch with empty rules",
			rules:    []nodeselector.NodeSelectorTerm{},
			fields:   fields,
			expected: true,
		},
		{
			name: "NotMatch with empty expression rules",
			rules: []nodeselector.NodeSelectorTerm{
				{
					MatchExpressions: []nodeselector.NodeSelectorRequirement{},
				},
			},
			fields:   fields,
			expected: false,
		},
		{
			name: "NotMatch with nonexistent platform",
			rules: []nodeselector.NodeSelectorTerm{
				{
					MatchExpressions: []nodeselector.NodeSelectorRequirement{
						{
							Key:      "platform",
							Operator: nodeselector.NodeSelectorOpIn,
							Values:   []string{"bad-platform"},
						},
					},
				},
			},
			fields:   fields,
			expected: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			match, err := nodeselector.Match(tt.rules, tt.fields)

			if tt.expectedError != nil {
				assert.NotNil(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, match)
			}
		})
	}
}

func TestMatchExpressions(t *testing.T) {
	fields := map[string]string{
		"platform": "test-platform",
		"hostname": "test-hostname",
		"region":   "region-a",
		"int":      "10",
	}

	for _, tt := range []struct {
		name          string
		rules         []nodeselector.NodeSelectorRequirement
		fields        map[string]string
		expected      bool
		expectedError error
	}{
		{
			name: "MatchExpressions with In operator error",
			rules: []nodeselector.NodeSelectorRequirement{
				{
					Key:      "platform",
					Operator: nodeselector.NodeSelectorOpIn,
					Values:   []string{},
				},
			},
			fields:        fields,
			expected:      false,
			expectedError: fmt.Errorf("values must be non-empty for operator 'In'"),
		},
		{
			name: "MatchExpressions with In and Exists operator",
			rules: []nodeselector.NodeSelectorRequirement{
				{
					Key:      "platform",
					Operator: nodeselector.NodeSelectorOpIn,
					Values:   []string{"test-platform"},
				},
				{
					Key:      "region",
					Operator: nodeselector.NodeSelectorOpExists,
				},
				{
					Key:      "fake-key",
					Operator: nodeselector.NodeSelectorOpDoesNotExist,
				},
			},
			fields:   fields,
			expected: true,
		},
		{
			name: "MatchExpressions with GtLt operator",
			rules: []nodeselector.NodeSelectorRequirement{
				{
					Key:      "platform",
					Operator: nodeselector.NodeSelectorOpIn,
					Values:   []string{"test-platform"},
				},
				{
					Key:      "int",
					Operator: nodeselector.NodeSelectorOpGt,
					Values:   []string{"5"},
				},
			},
			fields:   fields,
			expected: true,
		},
		{
			name: "MatchExpressions with regexp operator",
			rules: []nodeselector.NodeSelectorRequirement{
				{
					Key:      "platform",
					Operator: nodeselector.NodeSelectorOpIn,
					Values:   []string{"test-platform"},
				},
				{
					Key:      "hostname",
					Operator: nodeselector.NodeSelectorOpRegexp,
					Values:   []string{"^test.+$"},
				},
			},
			fields:   fields,
			expected: true,
		},
		{
			name: "MatchExpressions with regexp operator did not match",
			rules: []nodeselector.NodeSelectorRequirement{
				{
					Key:      "platform",
					Operator: nodeselector.NodeSelectorOpIn,
					Values:   []string{"test-platform"},
				},
				{
					Key:      "hostname",
					Operator: nodeselector.NodeSelectorOpRegexp,
					Values:   []string{"^somename.+$"},
				},
			},
			fields:   fields,
			expected: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			match, err := nodeselector.MatchExpressions(tt.rules, tt.fields)

			if tt.expectedError != nil {
				assert.NotNil(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, match)
			}
		})
	}
}
