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

	if len(cfg.Global.Endpoints) != 0 {
		t.Errorf("incorrect endpoints: %s", cfg.Global.Endpoints)
	}

	if cfg.Global.PreferIPv6 {
		t.Errorf("%v is not default value of preferIPv6", cfg.Global.PreferIPv6)
	}

	if cfg.Global.ApproveNodeCSR {
		t.Errorf("%v is not default value of ApproveNodeCSR", cfg.Global.ApproveNodeCSR)
	}
}

func TestReadCloudConfig(t *testing.T) {
	t.Setenv("TALOS_ENDPOINTS", "127.0.0.1,127.0.0.2")

	cfg, err := readCloudConfig(strings.NewReader(`
global:
    approveNodeCSR: true
    preferIPv6: true
`))
	if err != nil {
		t.Fatalf("Should succeed when a valid config is provided: %s", err)
	}

	if len(cfg.Global.Endpoints) != 2 {
		t.Errorf("incorrect endpoints: %s", cfg.Global.Endpoints)
	}

	if !cfg.Global.PreferIPv6 {
		t.Errorf("incorrect preferIPv6: %v", cfg.Global.PreferIPv6)
	}

	if !cfg.Global.ApproveNodeCSR {
		t.Errorf("incorrect ApproveNodeCSR: %v", cfg.Global.ApproveNodeCSR)
	}
}
