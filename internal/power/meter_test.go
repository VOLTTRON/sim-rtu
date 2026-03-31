package power

import (
	"math"
	"testing"

	"github.com/VOLTTRON/sim-rtu/internal/config"
)

func TestNewSimulator(t *testing.T) {
	cfg := &config.PowerConfig{BaseLoadKW: 5.0, HVACLoadPerStageKW: 3.0}
	s := NewSimulator(cfg)

	if s.BaseLoadKW != 5.0 {
		t.Errorf("BaseLoadKW = %v, want 5.0", s.BaseLoadKW)
	}
	if s.HVACLoadPerStageKW != 3.0 {
		t.Errorf("HVACLoadPerStageKW = %v, want 3.0", s.HVACLoadPerStageKW)
	}
	if s.NominalVoltage != 120.0 {
		t.Errorf("NominalVoltage = %v, want 120.0", s.NominalVoltage)
	}
	if s.NominalFrequency != 60.0 {
		t.Errorf("NominalFrequency = %v, want 60.0", s.NominalFrequency)
	}
}

func TestNewSimulator_NilConfig(t *testing.T) {
	s := NewSimulator(nil)
	if s.BaseLoadKW != 0 {
		t.Errorf("BaseLoadKW = %v, want 0", s.BaseLoadKW)
	}
}

func TestSimulator_Compute_BaseLoad(t *testing.T) {
	cfg := &config.PowerConfig{BaseLoadKW: 5.0, HVACLoadPerStageKW: 3.0}
	s := NewSimulator(cfg)

	reading := s.Compute(0, 1.0/3600.0) // 1 second

	if math.Abs(reading.TotalPowerKW-5.0) > 0.001 {
		t.Errorf("TotalPowerKW = %v, want ~5.0", reading.TotalPowerKW)
	}

	// Phase powers should sum to approximately total
	phaseSum := reading.PowerA + reading.PowerB + reading.PowerC
	if math.Abs(phaseSum-reading.TotalPowerKW) > 0.001 {
		t.Errorf("phase sum %v != total %v", phaseSum, reading.TotalPowerKW)
	}
}

func TestSimulator_Compute_WithHVAC(t *testing.T) {
	cfg := &config.PowerConfig{BaseLoadKW: 5.0, HVACLoadPerStageKW: 3.0}
	s := NewSimulator(cfg)

	tests := []struct {
		name       string
		stages     int
		wantTotalW float64
	}{
		{"no HVAC", 0, 5.0},
		{"1 stage", 1, 8.0},
		{"2 stages", 2, 11.0},
		{"4 stages", 4, 17.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reading := s.Compute(tc.stages, 0)
			if math.Abs(reading.TotalPowerKW-tc.wantTotalW) > 0.001 {
				t.Errorf("TotalPowerKW = %v, want %v", reading.TotalPowerKW, tc.wantTotalW)
			}
		})
	}
}

func TestSimulator_Compute_VoltageRange(t *testing.T) {
	cfg := &config.PowerConfig{BaseLoadKW: 5.0, HVACLoadPerStageKW: 3.0}
	s := NewSimulator(cfg)

	// Run multiple times to check noise stays in range
	for i := 0; i < 100; i++ {
		reading := s.Compute(1, 0)

		for _, v := range []float64{reading.VoltageAN, reading.VoltageBN, reading.VoltageCN} {
			if v < 118.0 || v > 122.0 {
				t.Errorf("L-N voltage %v out of expected range [118, 122]", v)
			}
		}
	}
}

func TestSimulator_Compute_FrequencyRange(t *testing.T) {
	cfg := &config.PowerConfig{BaseLoadKW: 5.0, HVACLoadPerStageKW: 3.0}
	s := NewSimulator(cfg)

	for i := 0; i < 100; i++ {
		reading := s.Compute(0, 0)
		if reading.Frequency < 59.9 || reading.Frequency > 60.1 {
			t.Errorf("Frequency %v out of expected range", reading.Frequency)
		}
	}
}

func TestSimulator_Compute_EnergyAccumulation(t *testing.T) {
	cfg := &config.PowerConfig{BaseLoadKW: 10.0, HVACLoadPerStageKW: 0}
	s := NewSimulator(cfg)

	// 10 kW for 1 hour = 10 kWh
	_ = s.Compute(0, 1.0)
	reading := s.Compute(0, 1.0)

	// Should be 20 kWh after 2 hours at 10 kW
	if math.Abs(reading.TotalKWH-20.0) > 0.001 {
		t.Errorf("TotalKWH = %v, want 20.0", reading.TotalKWH)
	}
}

func TestSimulator_Compute_PowerFactor(t *testing.T) {
	cfg := &config.PowerConfig{BaseLoadKW: 5.0, HVACLoadPerStageKW: 3.0}
	s := NewSimulator(cfg)

	reading := s.Compute(1, 0)
	if reading.PowerFactor != 0.95 {
		t.Errorf("PowerFactor = %v, want 0.95", reading.PowerFactor)
	}
}

func TestSimulator_Compute_CurrentsPositive(t *testing.T) {
	cfg := &config.PowerConfig{BaseLoadKW: 5.0, HVACLoadPerStageKW: 3.0}
	s := NewSimulator(cfg)

	reading := s.Compute(2, 0)

	if reading.CurrentA <= 0 || reading.CurrentB <= 0 || reading.CurrentC <= 0 {
		t.Errorf("currents should be positive: A=%v B=%v C=%v",
			reading.CurrentA, reading.CurrentB, reading.CurrentC)
	}
}
