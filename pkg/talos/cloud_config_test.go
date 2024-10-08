package talos

import (
	"strings"
	"testing"
)

func TestReadCloudConfigEmpty(t *testing.T) {
	cfg, err := readCloudConfig(nil)
	if err != nil {
		t.Errorf("Should not fail when no config is provided: %s", err)
	}

	if cfg.Global.PreferIPv6 {
		t.Errorf("%v is not default value of preferIPv6", cfg.Global.PreferIPv6)
	}
}

func TestReadCloudConfig(t *testing.T) {
	t.Setenv("TALOS_ENDPOINTS", "127.0.0.1,127.0.0.2")

	cfg, err := readCloudConfig(strings.NewReader(`
global:
  preferIPv6: true
transformations:
- name: cluster
  nodeSelector:
  - name: cluter-1
    matchExpressions:
    - key: platform
      operator: In
      values:
      - cluter
    annotations:
      cluster-platform: "{{ .Platform }}"
    labels:
      node-role.kubernetes.io/web: ""
`))
	if err != nil {
		t.Fatalf("Should succeed when a valid config is provided: %s", err)
	}

	if !cfg.Global.PreferIPv6 {
		t.Errorf("incorrect preferIPv6: %v", cfg.Global.PreferIPv6)
	}
}
