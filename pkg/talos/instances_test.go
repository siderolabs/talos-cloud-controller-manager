package talos

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
)

type ccmTestSuite struct {
	suite.Suite

	i *instances
}

func (ts *ccmTestSuite) SetupTest() {
	ts.i = newInstances(nil)
}

func TestSuiteCCM(t *testing.T) {
	suite.Run(t, new(ccmTestSuite))
}

func (ts *ccmTestSuite) TestInstanceExists() {
	exists, err := ts.i.InstanceExists(context.Background(), &v1.Node{})
	ts.Require().NoError(err)
	ts.Require().True(exists)
}

func (ts *ccmTestSuite) TestInstanceShutdown() {
	exists, err := ts.i.InstanceShutdown(context.Background(), &v1.Node{})
	ts.Require().NoError(err)
	ts.Require().False(exists)
}

func TestInstanceMetadata(t *testing.T) {
	t.Setenv("TALOSCONFIG", "../../hack/talosconfig")

	cfg := cloudConfig{}

	ctx := t.Context()
	client, err := newClient(ctx, &cfg)
	assert.NoError(t, err)

	i := newInstances(client)

	for _, tt := range []struct {
		name     string
		node     *v1.Node
		expected *cloudprovider.InstanceMetadata
	}{
		{
			name: "node does not have --cloud-provider=external",
			node: &v1.Node{
				Spec: v1.NodeSpec{},
			},
			expected: &cloudprovider.InstanceMetadata{},
		},
		// {
		// 	name: "node has providerID",
		// 	node: &v1.Node{
		// 		Spec: v1.NodeSpec{ProviderID: "provider:///id"},
		// 		ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
		// 			cloudproviderapi.AnnotationAlphaProvidedIPAddr: "127.0.0.1",
		// 		}},
		// 	},
		// 	expected: &cloudprovider.InstanceMetadata{},
		// },
	} {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := i.InstanceMetadata(ctx, tt.node)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, metadata)
		})
	}
}
