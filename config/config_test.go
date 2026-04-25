package config_test

import (
	"testing"
	"github.com/sharpsalt/Velox-In-Memory-Database/config"
)

func TestConfigLoad(t *testing.T) {
	// Simple sanity test to ensure config package initializes properly
	if config.Port == 0 {
		t.Errorf("Expected default Port to be non-zero, got %d", config.Port)
	}
	if config.Host == "" {
		t.Errorf("Expected default Host to be non-empty")
	}
}
