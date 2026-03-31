package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/VOLTTRON/sim-rtu/internal/config"
	"github.com/VOLTTRON/sim-rtu/internal/points"
	"github.com/VOLTTRON/sim-rtu/internal/power"
	"github.com/VOLTTRON/sim-rtu/internal/thermal"
)

// Device represents a single simulated device with its subsystems.
type Device struct {
	Config     config.DeviceConfig
	Store      *points.PointStore
	Thermal    *thermal.Model
	Controller *thermal.Controller
	Weather    thermal.WeatherProfile
	Schedule   *thermal.Schedule
	PowerMeter *power.Simulator
}

// Engine runs the simulation loop for all devices.
type Engine struct {
	config   *config.AppConfig
	devices  map[int]*Device
	elapsed  float64 // simulation time in hours
	mu       sync.RWMutex
	done     chan struct{}
	stopOnce sync.Once
}

// New creates a simulation engine from configuration.
func New(cfg *config.AppConfig) (*Engine, error) {
	e := &Engine{
		config:  cfg,
		devices: make(map[int]*Device),
		done:    make(chan struct{}),
	}

	for _, devCfg := range cfg.Devices {
		dev, err := buildDevice(devCfg)
		if err != nil {
			return nil, fmt.Errorf("build device %q: %w", devCfg.Name, err)
		}
		e.devices[devCfg.DeviceID] = dev
	}

	return e, nil
}

func buildDevice(cfg config.DeviceConfig) (*Device, error) {
	defs, err := points.ParseRegistry(cfg.Registry)
	if err != nil {
		return nil, fmt.Errorf("parse registry %s: %w", cfg.Registry, err)
	}

	dev := &Device{
		Config:   cfg,
		Store:    points.NewPointStore(defs),
		Schedule: thermal.DefaultSchedule(),
	}

	switch cfg.Type {
	case "thermostat":
		if cfg.Thermal != nil {
			m := thermal.NewModel(cfg.Thermal.R, cfg.Thermal.C)
			dev.Thermal = &m

			// Read deadband and proportional band defaults from store
			deadband := dev.Store.ReadFloat("DeadBand")
			if deadband == 0 {
				deadband = 3.0
			}
			propBand := dev.Store.ReadFloat("ProportionalBand")
			if propBand == 0 {
				propBand = 3.0
			}
			dev.Controller = thermal.NewController(deadband, propBand, 2*time.Minute)

			// Set initial zone temperature
			_ = dev.Store.SetInternal("ZoneTemperature", cfg.Thermal.InitialZoneTemp)
		}

		wp, err := thermal.NewWeatherProfile(cfg.Weather)
		if err != nil {
			return nil, fmt.Errorf("weather profile: %w", err)
		}
		dev.Weather = wp

	case "power_meter":
		dev.PowerMeter = power.NewSimulator(cfg.Power)
	}

	return dev, nil
}

// Start begins the simulation loop.
func (e *Engine) Start(ctx context.Context) error {
	tickInterval := time.Duration(e.config.Simulator.TickInterval * float64(time.Second))
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	slog.Info("engine started",
		"tick_interval", tickInterval,
		"time_scale", e.config.Simulator.TimeScale,
		"devices", len(e.devices),
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-e.done:
			return nil
		case <-ticker.C:
			dtHours := (e.config.Simulator.TickInterval * e.config.Simulator.TimeScale) / 3600.0
			e.tick(dtHours)
		}
	}
}

// Stop signals the engine to shut down. It is safe to call multiple times.
func (e *Engine) Stop() {
	e.stopOnce.Do(func() { close(e.done) })
}

// Devices returns a shallow copy of the device map so callers cannot
// race with the tick loop.
func (e *Engine) Devices() map[int]*Device {
	e.mu.RLock()
	defer e.mu.RUnlock()
	cp := make(map[int]*Device, len(e.devices))
	for k, v := range e.devices {
		cp[k] = v
	}
	return cp
}

// Device returns a single device by ID, or nil if not found.
func (e *Engine) Device(id int) *Device {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.devices[id]
}

// Elapsed returns the simulation time in hours.
func (e *Engine) Elapsed() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.elapsed
}

// tick advances the simulation by dt hours.
func (e *Engine) tick(dt float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.elapsed += dt
	now := time.Now()

	for _, dev := range e.devices {
		switch dev.Config.Type {
		case "thermostat":
			e.tickThermostat(dev, dt, now)
		case "power_meter":
			e.tickPowerMeter(dev, dt)
		}
	}
}

func (e *Engine) tickThermostat(dev *Device, dt float64, now time.Time) {
	if dev.Thermal == nil || dev.Controller == nil || dev.Weather == nil {
		return
	}

	tc := dev.Config.Thermal

	// Get outdoor temperature from weather profile
	outdoorTemp := dev.Weather.TemperatureAt(e.elapsed)
	_ = dev.Store.SetInternal("OutdoorAirTemperature", outdoorTemp)

	// Read current state from store
	zoneTemp := dev.Store.ReadFloat("ZoneTemperature")
	coolSP := dev.Store.ReadFloat("OccupiedCoolingSetPoint")
	heatSP := dev.Store.ReadFloat("OccupiedHeatingSetPoint")

	// Use unoccupied setpoints if unoccupied
	if dev.Schedule != nil {
		state := dev.Schedule.StateAt(now)
		if state == thermal.Unoccupied {
			uCool := dev.Store.ReadFloat("UnoccupiedCoolingSetPoint")
			uHeat := dev.Store.ReadFloat("UnoccupiedHeatingSetPoint")
			if uCool > 0 {
				coolSP = uCool
			}
			if uHeat > 0 {
				heatSP = uHeat
			}
		}
	}

	// Fallback setpoints
	if coolSP == 0 {
		coolSP = 75.0
	}
	if heatSP == 0 {
		heatSP = 72.0
	}

	// Run controller
	hvac := dev.Controller.Evaluate(zoneTemp, coolSP, heatSP, outdoorTemp, now)

	// Write staging outputs to store
	boolToFloat := func(b bool) float64 {
		if b {
			return 1.0
		}
		return 0.0
	}

	_ = dev.Store.SetInternal("FirstStageCooling", boolToFloat(hvac.FirstStageCooling))
	_ = dev.Store.SetInternal("SecondStageCooling", boolToFloat(hvac.SecondStageCooling))
	_ = dev.Store.SetInternal("FirstStageHeating", boolToFloat(hvac.FirstStageHeating))
	_ = dev.Store.SetInternal("SupplyFanStatus", boolToFloat(hvac.FanStatus))
	_ = dev.Store.SetInternal("HeatingDemand", hvac.HeatingDemand)
	_ = dev.Store.SetInternal("CoolingDemand", hvac.CoolingDemand)
	_ = dev.Store.SetInternal("EconomizerDemand", hvac.EconomizerDemand)

	// Compute Q_hvac from staging and capacities
	var qHVAC float64
	if hvac.FirstStageCooling && tc != nil {
		qHVAC -= tc.CoolingCapacityStage1
	}
	if hvac.SecondStageCooling && tc != nil {
		qHVAC -= tc.CoolingCapacityStage2
	}
	if hvac.FirstStageHeating && tc != nil {
		qHVAC += tc.HeatingCapacityStage1
	}
	if hvac.SecondStageHeating && tc != nil {
		qHVAC += tc.HeatingCapacityStage2
	}

	// Step thermal model
	newZoneTemp := dev.Thermal.Step(zoneTemp, outdoorTemp, qHVAC, dt)
	_ = dev.Store.SetInternal("ZoneTemperature", newZoneTemp)
}

func (e *Engine) tickPowerMeter(dev *Device, dt float64) {
	if dev.PowerMeter == nil {
		return
	}

	// For standalone power meters, count stages based on linked thermostats
	// For now, use base load only (0 active stages)
	activeStages := 0

	// Try to find linked thermostat stages from other devices
	for _, other := range e.devices {
		if other.Config.Type == "thermostat" && other.Controller != nil {
			state := other.Controller.CurrentState()
			activeStages += state.TotalActiveStages()
		}
	}

	reading := dev.PowerMeter.Compute(activeStages, dt)

	// Write all power points to store (ignore errors for missing points)
	_ = dev.Store.SetInternal("Current", (reading.CurrentA+reading.CurrentB+reading.CurrentC)/3)
	_ = dev.Store.SetInternal("CurrentA", reading.CurrentA)
	_ = dev.Store.SetInternal("CurrentB", reading.CurrentB)
	_ = dev.Store.SetInternal("CurrentC", reading.CurrentC)
	_ = dev.Store.SetInternal("VoltageAN", reading.VoltageAN)
	_ = dev.Store.SetInternal("VoltageBN", reading.VoltageBN)
	_ = dev.Store.SetInternal("VoltageCN", reading.VoltageCN)
	_ = dev.Store.SetInternal("VoltageAB", reading.VoltageAB)
	_ = dev.Store.SetInternal("VoltageBC", reading.VoltageBC)
	_ = dev.Store.SetInternal("VoltageCA", reading.VoltageCA)
	_ = dev.Store.SetInternal("WholeBuildingPower", reading.TotalPowerKW)
	_ = dev.Store.SetInternal("PowerA", reading.PowerA)
	_ = dev.Store.SetInternal("PowerB", reading.PowerB)
	_ = dev.Store.SetInternal("PowerC", reading.PowerC)
	_ = dev.Store.SetInternal("PowerFactor", reading.PowerFactor)
	_ = dev.Store.SetInternal("Frequency", reading.Frequency)
	_ = dev.Store.SetInternal("VoltageN", (reading.VoltageAN+reading.VoltageBN+reading.VoltageCN)/3)
	_ = dev.Store.SetInternal("VoltageLL", (reading.VoltageAB+reading.VoltageBC+reading.VoltageCA)/3)
}
