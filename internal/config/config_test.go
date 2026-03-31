package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_DefaultConfig(t *testing.T) {
	// Find the default.yml relative to the project root
	// Tests run in the package dir, so we navigate up
	configPath := filepath.Join("..", "..", "configs", "default.yml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("configs/default.yml not found, skipping integration test")
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Simulator.TickInterval != 1.0 {
		t.Errorf("TickInterval = %v, want 1.0", cfg.Simulator.TickInterval)
	}
	if cfg.Simulator.TimeScale != 1.0 {
		t.Errorf("TimeScale = %v, want 1.0", cfg.Simulator.TimeScale)
	}
	if len(cfg.Devices) != 3 {
		t.Errorf("len(Devices) = %d, want 3", len(cfg.Devices))
	}

	// Check device types
	types := map[string]int{}
	for _, d := range cfg.Devices {
		types[d.Type]++
	}
	if types["thermostat"] != 2 {
		t.Errorf("thermostat count = %d, want 2", types["thermostat"])
	}
	if types["power_meter"] != 1 {
		t.Errorf("power_meter count = %d, want 1", types["power_meter"])
	}

	if !cfg.BACnet.Enabled {
		t.Error("BACnet should be enabled")
	}
	if cfg.BACnet.Port != 47808 {
		t.Errorf("BACnet.Port = %d, want 47808", cfg.BACnet.Port)
	}
	if !cfg.API.Enabled {
		t.Error("API should be enabled")
	}
	if cfg.API.Port != 8080 {
		t.Errorf("API.Port = %d, want 8080", cfg.API.Port)
	}
}

func TestLoad_NotFound(t *testing.T) {
	_, err := Load("/nonexistent/path.yml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "bad.yml")
	if err := os.WriteFile(tmpFile, []byte(":::invalid:::"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(tmpFile)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestValidate(t *testing.T) {
	validDevice := DeviceConfig{
		Name:     "test",
		Type:     "thermostat",
		DeviceID: 1,
		Registry: "test.csv",
		Thermal:  &ThermalConfig{R: 0.02, C: 1000},
	}

	tests := []struct {
		name    string
		cfg     AppConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: AppConfig{
				Simulator: SimulatorConfig{TickInterval: 1.0, TimeScale: 1.0},
				Devices:   []DeviceConfig{validDevice},
			},
			wantErr: false,
		},
		{
			name: "zero tick interval",
			cfg: AppConfig{
				Simulator: SimulatorConfig{TickInterval: 0, TimeScale: 1.0},
				Devices:   []DeviceConfig{validDevice},
			},
			wantErr: true,
		},
		{
			name: "zero time scale",
			cfg: AppConfig{
				Simulator: SimulatorConfig{TickInterval: 1.0, TimeScale: 0},
				Devices:   []DeviceConfig{validDevice},
			},
			wantErr: true,
		},
		{
			name: "no devices",
			cfg: AppConfig{
				Simulator: SimulatorConfig{TickInterval: 1.0, TimeScale: 1.0},
				Devices:   nil,
			},
			wantErr: true,
		},
		{
			name: "duplicate device IDs",
			cfg: AppConfig{
				Simulator: SimulatorConfig{TickInterval: 1.0, TimeScale: 1.0},
				Devices: []DeviceConfig{
					validDevice,
					{Name: "test2", Type: "thermostat", DeviceID: 1, Registry: "test.csv", Thermal: &ThermalConfig{}},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid device type",
			cfg: AppConfig{
				Simulator: SimulatorConfig{TickInterval: 1.0, TimeScale: 1.0},
				Devices:   []DeviceConfig{{Name: "test", Type: "invalid", DeviceID: 1, Registry: "test.csv"}},
			},
			wantErr: true,
		},
		{
			name: "thermostat without thermal",
			cfg: AppConfig{
				Simulator: SimulatorConfig{TickInterval: 1.0, TimeScale: 1.0},
				Devices:   []DeviceConfig{{Name: "test", Type: "thermostat", DeviceID: 1, Registry: "test.csv"}},
			},
			wantErr: true,
		},
		{
			name: "power_meter without power",
			cfg: AppConfig{
				Simulator: SimulatorConfig{TickInterval: 1.0, TimeScale: 1.0},
				Devices:   []DeviceConfig{{Name: "test", Type: "power_meter", DeviceID: 1, Registry: "test.csv"}},
			},
			wantErr: true,
		},
		{
			name: "bacnet enabled with invalid port",
			cfg: AppConfig{
				Simulator: SimulatorConfig{TickInterval: 1.0, TimeScale: 1.0},
				Devices:   []DeviceConfig{validDevice},
				BACnet:    BACnetConfig{Enabled: true, Port: 0},
			},
			wantErr: true,
		},
		{
			name: "api enabled with invalid port",
			cfg: AppConfig{
				Simulator: SimulatorConfig{TickInterval: 1.0, TimeScale: 1.0},
				Devices:   []DeviceConfig{validDevice},
				API:       APIConfig{Enabled: true, Port: 0},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
