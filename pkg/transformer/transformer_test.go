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
			expected: &transformer.NodeSpec{
				Annotations: map[string]string{},
				Labels:      map[string]string{},
				Taints:      map[string]string{},
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
				Taints: map[string]string{},
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
				Taints: map[string]string{},
			},
		},
		{
			name: "Transform taints",
			terms: []transformer.NodeTerm{
				{
					Name: "my-transformer",
					Taints: map[string]string{
						"my-taint-name": "NoSchedule",
					},
				},
			},
			metadata: runtime.PlatformMetadataSpec{
				Platform: "test-platform",
				Hostname: "test-hostname",
			},
			expected: &transformer.NodeSpec{
				Annotations: map[string]string{},
				Labels:      map[string]string{},
				Taints: map[string]string{
					"my-taint-name": "NoSchedule",
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
			expectedError: fmt.Errorf("failed to transformer label \"label-template\": failed to parse template \"my-value-{{ .Spot\": template: transformer:1: unclosed action"),
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
				Taints: map[string]string{},
			},
		},
		{
			name: "Transform labels with template and missing metadata fields",
			terms: []transformer.NodeTerm{
				{
					Name: "my-transformer",
					Labels: map[string]string{
						"label-template": "my-value-{{ .Spot }}-{{ .Zone }}a",
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
					"label-template": "my-value-false-a",
				},
				Taints: map[string]string{},
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
				Taints: map[string]string{},
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
				Taints:      map[string]string{},
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
						"squat.ai/enabled":           `{{ if semverCompare "=> 1.8" .TalosVersion }}true{{ end }}`,
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
					"squat.ai/enabled":           "true",
				},
				Taints: map[string]string{},
			},
			expectedMeta: &runtime.PlatformMetadataSpec{
				Platform: "test-platform",
				Hostname: "test-hostname",
				Zone:     "us-west1",
			},
		},
		{
			name: "Transform labels with bad label name",
			terms: []transformer.NodeTerm{
				{
					Name: "my-transformer",
					Labels: map[string]string{
						"-template": "my-value",
					},
				},
			},
			metadata: runtime.PlatformMetadataSpec{
				Platform: "test-platform",
				Hostname: "test-hostname",
			},
			expectedError: fmt.Errorf("invalid label name \"-template\": [name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')]"), //nolint:lll
		},
		{
			name: "Transform taint with bad name",
			terms: []transformer.NodeTerm{
				{
					Name: "my-transformer",
					Taints: map[string]string{
						"node.kubernetes.io/pid-pressure": "my-value",
					},
				},
			},
			metadata: runtime.PlatformMetadataSpec{
				Platform: "test-platform",
				Hostname: "test-hostname",
			},
			expectedError: fmt.Errorf("invalid taint name \"node.kubernetes.io/pid-pressure\": [taint in kubernetes namespace]"), //nolint:lll
		},
		{
			name: "Transform taint with bad value",
			terms: []transformer.NodeTerm{
				{
					Name: "my-transformer",
					Taints: map[string]string{
						"node.cloudprovider.kubernetes.io/storage-type": "my-value:PleaseSchedule",
					},
				},
			},
			metadata: runtime.PlatformMetadataSpec{
				Platform: "test-platform",
				Hostname: "test-hostname",
			},
			expectedError: fmt.Errorf("invalid taint value \"my-value:PleaseSchedule\": [taint effect \"PleaseSchedule\" is not valid]"), //nolint:lll
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			node, err := transformer.TransformNode(tt.terms, &tt.metadata, nil, "1.8.0")

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
