package main

import "testing"

func TestFilterConfigs(t *testing.T) {
	cfgs := filter(Configs, "Preset", "Constant Rate Factor (CRF)")

	for _, cfg := range cfgs {
		if cfg.Name == "Preset" {
			t.Fatalf("Failed to filter out config \"Preset\"")
		} else if cfg.Name == "Constant Rate Factor (CRF)" {
			t.Fatalf("Failed to filter out config \"Constant Rate Factor (CRF)\"")
		}
	}
}

func TestGetVisibleConfigs(t *testing.T) {
	cfgs := getVisibleConfigs(Configs)

	if len(cfgs) == len(Configs) {
		t.Fatalf("Failed to get the correct visible configs. Expected VisibleConfigs len: %v. Got %v", 4, len(cfgs))
	}
}
