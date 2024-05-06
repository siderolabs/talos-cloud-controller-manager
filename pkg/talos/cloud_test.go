package talos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func config() cloudConfig {
	cfg := cloudConfig{}

	return cfg
}

func TestNewCloudError(t *testing.T) {
	ccm, err := newCloud(nil)
	assert.NotNil(t, err)
	assert.Nil(t, ccm)
	assert.EqualError(t, err, "talos cloudConfig is nil")
}

func TestNewCloud(t *testing.T) {
	t.Setenv("TALOSCONFIG", "../../hack/talosconfig")

	cfg := config()

	ccm, err := newCloud(&cfg)
	if err != nil {
		t.Fatalf("Failed to create Talos CCM: %s", err)
	}

	assert.Nil(t, err)
	assert.NotNil(t, ccm)

	lb, res := ccm.LoadBalancer()
	assert.Nil(t, lb)
	assert.Equal(t, res, false)

	ins, res := ccm.Instances()
	assert.Nil(t, ins)
	assert.Equal(t, res, false)

	ins2, res := ccm.InstancesV2()
	assert.NotNil(t, ins2)
	assert.Equal(t, res, true)

	zone, res := ccm.Zones()
	assert.Nil(t, zone)
	assert.Equal(t, res, false)

	cl, res := ccm.Clusters()
	assert.Nil(t, cl)
	assert.Equal(t, res, false)

	route, res := ccm.Routes()
	assert.Nil(t, route)
	assert.Equal(t, res, false)

	pName := ccm.ProviderName()
	assert.Equal(t, pName, ProviderName)

	clID := ccm.HasClusterID()
	assert.Equal(t, clID, true)
}
