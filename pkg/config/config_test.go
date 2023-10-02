package config

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Set up test data
	expectedConfig := appConfig{
		ConfigVar: "test_config_var",
	}

	// Call LoadConfig and check the result
	err := LoadConfig("./testdata/config")
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}
	if Config != expectedConfig {
		t.Errorf("Loaded config does not match expected config. Got %v, expected %v", Config, expectedConfig)
	}
	if Config.ConfigVar != expectedConfig.ConfigVar {
		t.Errorf("Loaded config value ConfigVar does not match expected. Got %v, expected %v", Config.ConfigVar, expectedConfig.ConfigVar)
	}
}
