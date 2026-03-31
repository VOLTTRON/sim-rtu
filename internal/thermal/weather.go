package thermal

import (
	"fmt"
	"math"

	"github.com/VOLTTRON/sim-rtu/internal/config"
)

// WeatherProfile provides outdoor temperature at a given simulation time.
type WeatherProfile interface {
	TemperatureAt(simTimeHours float64) float64
}

// ConstantWeather returns a fixed temperature at all times.
type ConstantWeather struct {
	Temperature float64
}

// TemperatureAt returns the constant temperature.
func (c ConstantWeather) TemperatureAt(_ float64) float64 {
	return c.Temperature
}

// SineWaveWeather generates a sinusoidal temperature pattern over 24 hours.
type SineWaveWeather struct {
	Mean        float64
	Amplitude   float64
	PhaseOffset float64 // hours
}

// TemperatureAt returns the temperature at the given simulation time.
// Formula: Mean + Amplitude * sin(2*pi*(simTimeHours - PhaseOffset) / 24.0)
func (s SineWaveWeather) TemperatureAt(simTimeHours float64) float64 {
	return s.Mean + s.Amplitude*math.Sin(2*math.Pi*(simTimeHours-s.PhaseOffset)/24.0)
}

// NewWeatherProfile creates a WeatherProfile from configuration.
func NewWeatherProfile(cfg *config.WeatherConfig) (WeatherProfile, error) {
	if cfg == nil {
		return ConstantWeather{Temperature: 72.0}, nil
	}

	switch cfg.Type {
	case "static", "constant":
		return ConstantWeather{Temperature: cfg.Temperature}, nil
	case "sine_wave", "sinusoidal":
		return SineWaveWeather{
			Mean:        cfg.Mean,
			Amplitude:   cfg.Amplitude,
			PhaseOffset: cfg.PhaseOffset,
		}, nil
	default:
		return nil, fmt.Errorf("unknown weather type %q", cfg.Type)
	}
}
