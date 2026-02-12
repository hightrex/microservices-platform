package config

import (
	"testing"
)

type TestConfig struct {
	AppParam string `mapstructure:"app_param"`
}

func TestLoadDefaults(t *testing.T) {
	var cfg TestConfig
	// Should not error even if file missing
	err := Load("", "nonexistent", &cfg)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
}
