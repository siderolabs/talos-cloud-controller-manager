package talos

import (
	"io"

	yaml "gopkg.in/yaml.v3"

	"github.com/siderolabs/talos-cloud-controller-manager/pkg/transformer"

	"k8s.io/klog/v2"
)

type cloudConfig struct {
	// Global configuration.
	Global cloudConfigGlobal `yaml:"global,omitempty"`
	// Node transformation configuration.
	Transformations []transformer.NodeTerm `yaml:"transformations,omitempty"`
}

type cloudConfigGlobal struct {
	// Approve Node Certificate Signing Request.
	ApproveNodeCSR bool `yaml:"approveNodeCSR,omitempty"`
	// Talos cluster name.
	ClusterName string `yaml:"clusterName,omitempty"`
	// Prefer IPv6.
	PreferIPv6 bool `yaml:"preferIPv6,omitempty"`
}

func readCloudConfig(config io.Reader) (cloudConfig, error) {
	cfg := cloudConfig{}

	if config != nil {
		if err := yaml.NewDecoder(config).Decode(&cfg); err != nil {
			return cloudConfig{}, err
		}
	}

	klog.V(4).InfoS("cloudConfig", "cfg", cfg)

	return cfg, nil
}
