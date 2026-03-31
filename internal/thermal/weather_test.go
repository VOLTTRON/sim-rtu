package thermal

import (
	"math"
	"testing"

	"github.com/VOLTTRON/sim-rtu/internal/config"
)

func TestConstantWeather(t *testing.T) {
	w := ConstantWeather{Temperature: 85.0}

	tests := []struct {
		name string
		time float64
	}{
		{"time 0", 0},
		{"time 12", 12},
		{"time 24", 24},
		{"time 100", 100},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := w.TemperatureAt(tc.time)
			if got != 85.0 {
				t.Errorf("TemperatureAt(%v) = %v, want 85.0", tc.time, got)
			}
		})
	}
}

func TestSineWaveWeather(t *testing.T) {
	w := SineWaveWeather{Mean: 85.0, Amplitude: 15.0, PhaseOffset: 14.0}

	tests := []struct {
		name      string
		time      float64
		wantApprx float64
		tolerance float64
	}{
		{
			name:      "at phase offset (sin=0)",
			time:      14.0,
			wantApprx: 85.0,
			tolerance: 0.001,
		},
		{
			name:      "6 hours after phase (sin=1 peak)",
			time:      20.0,
			wantApprx: 100.0, // 85 + 15
			tolerance: 0.001,
		},
		{
			name:      "6 hours before phase (sin=-1 trough)",
			time:      8.0,
			wantApprx: 70.0, // 85 - 15
			tolerance: 0.001,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := w.TemperatureAt(tc.time)
			if math.Abs(got-tc.wantApprx) > tc.tolerance {
				t.Errorf("TemperatureAt(%v) = %v, want ~%v", tc.time, got, tc.wantApprx)
			}
		})
	}
}

func TestSineWaveWeather_Period(t *testing.T) {
	w := SineWaveWeather{Mean: 80.0, Amplitude: 10.0, PhaseOffset: 0}

	// Temperature at t should equal temperature at t+24
	t0 := w.TemperatureAt(5.0)
	t24 := w.TemperatureAt(29.0)
	if math.Abs(t0-t24) > 0.001 {
		t.Errorf("24h period not maintained: T(5)=%v, T(29)=%v", t0, t24)
	}
}

func TestNewWeatherProfile(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.WeatherConfig
		wantErr bool
		check   func(WeatherProfile) bool
	}{
		{
			name:    "nil config defaults to constant 72",
			cfg:     nil,
			wantErr: false,
			check:   func(w WeatherProfile) bool { return w.TemperatureAt(0) == 72.0 },
		},
		{
			name:    "static type",
			cfg:     &config.WeatherConfig{Type: "static", Temperature: 90.0},
			wantErr: false,
			check:   func(w WeatherProfile) bool { return w.TemperatureAt(0) == 90.0 },
		},
		{
			name:    "constant type alias",
			cfg:     &config.WeatherConfig{Type: "constant", Temperature: 60.0},
			wantErr: false,
			check:   func(w WeatherProfile) bool { return w.TemperatureAt(0) == 60.0 },
		},
		{
			name:    "sine_wave type",
			cfg:     &config.WeatherConfig{Type: "sine_wave", Mean: 85.0, Amplitude: 15.0, PhaseOffset: 14.0},
			wantErr: false,
			check:   func(w WeatherProfile) bool { return math.Abs(w.TemperatureAt(14.0)-85.0) < 0.001 },
		},
		{
			name:    "unknown type",
			cfg:     &config.WeatherConfig{Type: "unknown"},
			wantErr: true,
			check:   nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, err := NewWeatherProfile(tc.cfg)
			if (err != nil) != tc.wantErr {
				t.Errorf("NewWeatherProfile() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.check != nil && !tc.check(w) {
				t.Error("weather profile check failed")
			}
		})
	}
}
