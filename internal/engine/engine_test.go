package engine

import (
	"context"
	"testing"
	"time"

	"github.com/VOLTTRON/sim-rtu/internal/config"
)

func testConfig() *config.AppConfig {
	return &config.AppConfig{
		Simulator: config.SimulatorConfig{
			TickInterval: 0.1,
			TimeScale:    60.0, // 1 second = 1 minute
		},
		Devices: []config.DeviceConfig{
			{
				Name:     "Test-Thermostat",
				Type:     "thermostat",
				DeviceID: 1,
				Registry: "../../configs/schneider.csv",
				Thermal: &config.ThermalConfig{
					R:                     0.02,
					C:                     1000,
					InitialZoneTemp:       72.0,
					CoolingCapacityStage1: 18000,
					CoolingCapacityStage2: 18000,
					HeatingCapacityStage1: 20000,
					HeatingCapacityStage2: 20000,
				},
				Weather: &config.WeatherConfig{
					Type:        "sine_wave",
					Mean:        85.0,
					Amplitude:   15.0,
					PhaseOffset: 14.0,
				},
			},
			{
				Name:     "Test-Meter",
				Type:     "power_meter",
				DeviceID: 2,
				Registry: "../../configs/dent.csv",
				Power: &config.PowerConfig{
					BaseLoadKW:         5.0,
					HVACLoadPerStageKW: 3.0,
				},
			},
		},
	}
}

func TestNew(t *testing.T) {
	cfg := testConfig()
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if len(e.devices) != 2 {
		t.Errorf("devices count = %d, want 2", len(e.devices))
	}

	// Check thermostat device
	dev := e.devices[1]
	if dev.Config.Name != "Test-Thermostat" {
		t.Errorf("device 1 name = %q, want Test-Thermostat", dev.Config.Name)
	}
	if dev.Thermal == nil {
		t.Error("thermostat should have thermal model")
	}
	if dev.Controller == nil {
		t.Error("thermostat should have controller")
	}
	if dev.Weather == nil {
		t.Error("thermostat should have weather profile")
	}

	// Check initial zone temp
	zt := dev.Store.ReadFloat("ZoneTemperature")
	if zt != 72.0 {
		t.Errorf("initial ZoneTemperature = %v, want 72.0", zt)
	}

	// Check power meter device
	meter := e.devices[2]
	if meter.PowerMeter == nil {
		t.Error("power meter device should have PowerMeter")
	}
}

func TestNew_InvalidRegistry(t *testing.T) {
	cfg := &config.AppConfig{
		Simulator: config.SimulatorConfig{TickInterval: 1.0, TimeScale: 1.0},
		Devices: []config.DeviceConfig{
			{
				Name:     "Bad",
				Type:     "thermostat",
				DeviceID: 1,
				Registry: "/nonexistent.csv",
				Thermal:  &config.ThermalConfig{R: 0.02, C: 1000},
			},
		},
	}
	_, err := New(cfg)
	if err == nil {
		t.Error("expected error for invalid registry path")
	}
}

func TestEngine_Tick(t *testing.T) {
	cfg := testConfig()
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Run a few ticks
	dt := 1.0 / 3600.0 // 1 second in hours
	for i := 0; i < 10; i++ {
		e.tick(dt)
	}

	// Zone temperature should have changed from initial 72.0
	dev := e.devices[1]
	zt := dev.Store.ReadFloat("ZoneTemperature")
	if zt == 72.0 {
		t.Error("ZoneTemperature should have changed after ticks")
	}

	// Elapsed should reflect ticks
	if e.Elapsed() == 0 {
		t.Error("Elapsed() should be > 0 after ticks")
	}
}

func TestEngine_StartStop(t *testing.T) {
	cfg := testConfig()
	cfg.Simulator.TickInterval = 0.01 // fast ticks for test

	e, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- e.Start(ctx)
	}()

	// Let it run briefly
	time.Sleep(50 * time.Millisecond)

	// Stop via context cancellation
	cancel()

	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Errorf("Start() returned unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("engine did not stop within timeout")
	}
}

func TestEngine_Devices(t *testing.T) {
	cfg := testConfig()
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	devices := e.Devices()
	if len(devices) != 2 {
		t.Errorf("Devices() returned %d devices, want 2", len(devices))
	}
}
