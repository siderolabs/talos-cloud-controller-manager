package talos

import (
	"io"
	"os"
	"strings"

	yaml "gopkg.in/yaml.v3"

	"k8s.io/klog/v2"
)

type cloudConfig struct {
	Global struct {
		// Talos API endpoints.
		Endpoints []string `yaml:"endpoints,omitempty"`
		// Do not update foreign initialized node.
		SkipForeignNode bool `yaml:"skipForeignNode,omitempty"`
		// Prefer IPv6.
		PreferIPv6 bool `yaml:"preferIPv6,omitempty"`
	} `yaml:"global,omitempty"`
}

func readCloudConfig(config io.Reader) (cloudConfig, error) {
	cfg := cloudConfig{}

	if config != nil {
		if err := yaml.NewDecoder(config).Decode(&cfg); err != nil {
			return cloudConfig{}, err
		}
	}

	endpoints := os.Getenv("TALOS_ENDPOINTS")
	if endpoints != "" {
		cfg.Global.Endpoints = strings.Split(endpoints, ",")
	}

	klog.V(4).Infof("cloudConfig: %+v", cfg)

	return cfg, nil
}
