package transformer_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/siderolabs/talos-cloud-controller-manager/pkg/transformer"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
)

func TestMatch(t *testing.T) {
	for _, tt := range []struct {
		name          string
		terms         []transformer.NodeTerm
		metadata      runtime.PlatformMetadataSpec
		expected      *transformer.NodeSpec
		expectedMeta  *runtime.PlatformMetadataSpec
		expectedError error
	}{
		{
			name:  "Empty rules",
			terms: []transformer.NodeTerm{},
			metadata: runtime.PlatformMetadataSpec{
				Platform: "test-platform",
				Hostname: "test-hostname",
			},
		},
		{
			name: "Transform labels",
			terms: []transformer.NodeTerm{
				{
					Name: "my-transformer",
					Labels: map[string]string{
						"my-label-name": "my-value",
					},
				},
			},
			metadata: runtime.PlatformMetadataSpec{
				Platform: "test-platform",
				Hostname: "test-hostname",
			},
			expected: &transformer.NodeSpec{
				Annotations: map[string]string{},
				Labels: map[string]string{
					"my-label-name": "my-value",
				},
			},
		},
		{
			name: "Transform annotations and labels",
			terms: []transformer.NodeTerm{
				{
					Name: "my-transformer",
					Labels: map[string]string{
						"my-label-name": "my-value",
					},
					Annotations: map[string]string{
						"my-annotation-name": "my-annotation-value",
					},
				},
			},
			metadata: runtime.PlatformMetadataSpec{
				Platform: "test-platform",
				Hostname: "test-hostname",
			},
			expected: &transformer.NodeSpec{
				Annotations: map[string]string{
					"my-annotation-name": "my-annotation-value",
				},
				Labels: map[string]string{
					"my-label-name": "my-value",
				},
			},
		},
		{
			name: "Transform bad template",
			terms: []transformer.NodeTerm{
				{
					Name: "my-transformer",
					Labels: map[string]string{
						"label-template": "my-value-{{ .Spot",
					},
				},
			},
			metadata: runtime.PlatformMetadataSpec{
				Platform: "test-platform",
				Hostname: "test-hostname",
			},
			expectedError: fmt.Errorf("failed to transformer label 'label-template': failed to parse template \"my-value-{{ .Spot\": template: transformer:1: unclosed action"),
		},
		{
			name: "Transform annotations with template",
			terms: []transformer.NodeTerm{
				{
					Name: "my-transformer",
					Annotations: map[string]string{
						"annotation-template": "my-value-{{ .Platform }}",
					},
				},
			},
			metadata: runtime.PlatformMetadataSpec{
				Platform: "test-platform",
				Hostname: "test-hostname",
			},
			expected: &transformer.NodeSpec{
				Annotations: map[string]string{
					"annotation-template": "my-value-test-platform",
				},
				Labels: map[string]string{},
			},
		},
		{
			name: "Transform labels with template and missing metadata fields",
			terms: []transformer.NodeTerm{
				{
					Name: "my-transformer",
					Labels: map[string]string{
						"label-template": "my-value-{{ .Spot }}-{{ .Zone }}",
					},
				},
			},
			metadata: runtime.PlatformMetadataSpec{
				Platform: "test-platform",
				Hostname: "test-hostname",
			},
			expected: &transformer.NodeSpec{
				Annotations: map[string]string{},
				Labels: map[string]string{
					"label-template": "my-value-false-",
				},
			},
		},
		{
			name: "Transform metadata fields",
			terms: []transformer.NodeTerm{
				{
					Name: "my-transformer",
					Labels: map[string]string{
						"karpenter.sh/capacity-type": "{{ if .Spot }}spot{{ else }}on-demand{{ end }}",
					},
					PlatformMetadata: map[string]string{
						"Zone": "us-west1",
					},
				},
			},
			metadata: runtime.PlatformMetadataSpec{
				Platform: "test-platform",
				Hostname: "test-hostname",
				Spot:     true,
			},
			expected: &transformer.NodeSpec{
				Annotations: map[string]string{},
				Labels: map[string]string{
					"karpenter.sh/capacity-type": "spot",
				},
			},
			expectedMeta: &runtime.PlatformMetadataSpec{
				Platform: "test-platform",
				Hostname: "test-hostname",
				Spot:     true,
				Zone:     "us-west1",
			},
		},
		{
			name: "Transform metadata with wrong fields",
			terms: []transformer.NodeTerm{
				{
					Name: "my-transformer",
					PlatformMetadata: map[string]string{
						"Hostname":     "fake-hostname",
						"spot":         "true",
						"zoNe":         "us-west1",
						"wrong":        "value",
						"InstanceType": `{{ regexFindString "^type-([a-z0-9]+)-(.*)$" .Hostname 1 }}`,
					},
				},
			},
			metadata: runtime.PlatformMetadataSpec{
				Platform: "test-platform",
				Hostname: "type-c1m5-hostname",
			},
			expected: &transformer.NodeSpec{
				Annotations: map[string]string{},
				Labels:      map[string]string{},
			},
			expectedMeta: &runtime.PlatformMetadataSpec{
				Platform:     "test-platform",
				Hostname:     "type-c1m5-hostname",
				Spot:         true,
				Zone:         "us-west1",
				InstanceType: "c1m5",
			},
		},
		{
			name: "Multiple transformers",
			terms: []transformer.NodeTerm{
				{
					Name: "first-rule",
					Annotations: map[string]string{
						"first-annotation": "first-value",
					},
					Labels: map[string]string{
						"karpenter.sh/capacity-type": "on-demand",
					},
				},
				{
					Name: "second-rule",
					Labels: map[string]string{
						"karpenter.sh/capacity-type": "spot",
					},
					PlatformMetadata: map[string]string{
						"Zone": "us-west1",
					},
				},
			},
			metadata: runtime.PlatformMetadataSpec{
				Platform: "test-platform",
				Hostname: "test-hostname",
			},
			expected: &transformer.NodeSpec{
				Annotations: map[string]string{
					"first-annotation": "first-value",
				},
				Labels: map[string]string{
					"karpenter.sh/capacity-type": "spot",
				},
			},
			expectedMeta: &runtime.PlatformMetadataSpec{
				Platform: "test-platform",
				Hostname: "test-hostname",
				Zone:     "us-west1",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			node, err := transformer.TransformNode(tt.terms, &tt.metadata)

			if tt.expectedError != nil {
				assert.NotNil(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)

				if tt.expected != nil {
					assert.NotNil(t, node)
					assert.EqualValues(t, tt.expected, node)

					if tt.expectedMeta != nil {
						assert.EqualValues(t, tt.expectedMeta, &tt.metadata)
					}
				}
			}
		})
	}
}
