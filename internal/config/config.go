package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AppConfig is the top-level configuration for the simulator.
type AppConfig struct {
	Simulator SimulatorConfig `yaml:"simulator"`
	Devices   []DeviceConfig  `yaml:"devices"`
	BACnet    BACnetConfig    `yaml:"bacnet"`
	API       APIConfig       `yaml:"api"`
}

// SimulatorConfig controls the simulation timing.
type SimulatorConfig struct {
	TickInterval float64 `yaml:"tick_interval"` // seconds between ticks
	TimeScale    float64 `yaml:"time_scale"`    // simulation speed multiplier
}

// DeviceConfig describes a single simulated device.
type DeviceConfig struct {
	Name     string         `yaml:"name"`
	Type     string         `yaml:"type"` // "thermostat" or "power_meter"
	DeviceID int            `yaml:"device_id"`
	Registry string         `yaml:"registry"`
	Thermal  *ThermalConfig `yaml:"thermal,omitempty"`
	Weather  *WeatherConfig `yaml:"weather,omitempty"`
	Power    *PowerConfig   `yaml:"power,omitempty"`
}

// ThermalConfig describes the RC thermal model parameters.
type ThermalConfig struct {
	R                     float64 `yaml:"R"`
	C                     float64 `yaml:"C"`
	InitialZoneTemp       float64 `yaml:"initial_zone_temp"`
	CoolingCapacityStage1 float64 `yaml:"cooling_capacity_stage1"`
	CoolingCapacityStage2 float64 `yaml:"cooling_capacity_stage2"`
	HeatingCapacityStage1 float64 `yaml:"heating_capacity_stage1"`
	HeatingCapacityStage2 float64 `yaml:"heating_capacity_stage2"`
}

// WeatherConfig describes how outdoor temperature is generated.
type WeatherConfig struct {
	Type        string  `yaml:"type"`                   // "static" or "sine_wave"
	Mean        float64 `yaml:"mean,omitempty"`          // for sine_wave
	Amplitude   float64 `yaml:"amplitude,omitempty"`     // for sine_wave
	PhaseOffset float64 `yaml:"phase_offset,omitempty"`  // for sine_wave (hours)
	Temperature float64 `yaml:"temperature,omitempty"`   // for static
}

// PowerConfig describes the power meter simulation parameters.
type PowerConfig struct {
	BaseLoadKW         float64 `yaml:"base_load_kw"`
	HVACLoadPerStageKW float64 `yaml:"hvac_load_per_stage_kw"`
}

// BACnetConfig controls the BACnet/IP server.
type BACnetConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Interface string `yaml:"interface"`
	Port      int    `yaml:"port"`
}

// APIConfig controls the REST API server.
type APIConfig struct {
	Enabled bool   `yaml:"enabled"`
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	Token   string `yaml:"token,omitempty"` // optional bearer token for PUT/POST
}

// Load reads and parses a YAML configuration file.
func Load(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// Validate checks the configuration for required fields and consistency.
func (c *AppConfig) Validate() error {
	if c.Simulator.TickInterval <= 0 {
		return fmt.Errorf("simulator.tick_interval must be > 0")
	}
	if c.Simulator.TimeScale <= 0 {
		return fmt.Errorf("simulator.time_scale must be > 0")
	}

	if len(c.Devices) == 0 {
		return fmt.Errorf("at least one device is required")
	}

	deviceIDs := make(map[int]bool)
	for i, d := range c.Devices {
		if d.Name == "" {
			return fmt.Errorf("devices[%d].name is required", i)
		}
		if d.Type == "" {
			return fmt.Errorf("devices[%d].type is required", i)
		}
		if d.Type != "thermostat" && d.Type != "power_meter" {
			return fmt.Errorf("devices[%d].type must be 'thermostat' or 'power_meter', got %q", i, d.Type)
		}
		if d.DeviceID <= 0 {
			return fmt.Errorf("devices[%d].device_id must be > 0", i)
		}
		if deviceIDs[d.DeviceID] {
			return fmt.Errorf("devices[%d].device_id %d is duplicated", i, d.DeviceID)
		}
		deviceIDs[d.DeviceID] = true

		if d.Registry == "" {
			return fmt.Errorf("devices[%d].registry is required", i)
		}

		if d.Type == "thermostat" && d.Thermal == nil {
			return fmt.Errorf("devices[%d] is thermostat but missing thermal config", i)
		}
		if d.Type == "thermostat" && d.Thermal != nil {
			if d.Thermal.R <= 0 {
				return fmt.Errorf("devices[%d].thermal.R must be > 0", i)
			}
			if d.Thermal.C <= 0 {
				return fmt.Errorf("devices[%d].thermal.C must be > 0", i)
			}
		}
		if d.Type == "power_meter" && d.Power == nil {
			return fmt.Errorf("devices[%d] is power_meter but missing power config", i)
		}
	}

	if c.BACnet.Enabled && c.BACnet.Port <= 0 {
		return fmt.Errorf("bacnet.port must be > 0 when enabled")
	}
	if c.API.Enabled && c.API.Port <= 0 {
		return fmt.Errorf("api.port must be > 0 when enabled")
	}

	return nil
}
