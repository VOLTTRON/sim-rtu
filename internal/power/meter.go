package power

import (
	"math"
	"math/rand/v2"

	"github.com/VOLTTRON/sim-rtu/internal/config"
)

const (
	defaultNominalVoltage   = 120.0
	defaultNominalFrequency = 60.0
	defaultPowerFactor      = 0.95
	voltageNoise            = 0.02  // 2% noise
	currentNoise            = 0.03  // 3% noise
	frequencyNoise          = 0.001 // 0.1% noise
)

// MeterReading contains all simulated power meter readings.
type MeterReading struct {
	CurrentA  float64
	CurrentB  float64
	CurrentC  float64
	VoltageAN float64
	VoltageBN float64
	VoltageCN float64
	VoltageAB float64
	VoltageBC float64
	VoltageCA float64

	TotalPowerKW float64
	PowerA       float64
	PowerB       float64
	PowerC       float64
	PowerFactor  float64
	Frequency    float64
	TotalKWH     float64
}

// Simulator generates realistic three-phase power meter readings.
type Simulator struct {
	BaseLoadKW         float64
	HVACLoadPerStageKW float64
	NominalVoltage     float64
	NominalFrequency   float64

	cumulativeKWH float64
}

// NewSimulator creates a power meter simulator from configuration.
func NewSimulator(cfg *config.PowerConfig) *Simulator {
	s := &Simulator{
		NominalVoltage:   defaultNominalVoltage,
		NominalFrequency: defaultNominalFrequency,
	}
	if cfg != nil {
		s.BaseLoadKW = cfg.BaseLoadKW
		s.HVACLoadPerStageKW = cfg.HVACLoadPerStageKW
	}
	return s
}

// Compute calculates a meter reading given the number of active HVAC stages
// and the elapsed time since the last reading (in hours).
func (s *Simulator) Compute(activeHVACStages int, elapsedHours float64) MeterReading {
	totalKW := s.BaseLoadKW + float64(activeHVACStages)*s.HVACLoadPerStageKW

	// Distribute load across three phases with slight imbalance
	phaseFactors := [3]float64{
		1.0 + (rand.Float64()-0.5)*0.06, // +/- 3%
		1.0 + (rand.Float64()-0.5)*0.06,
		1.0 + (rand.Float64()-0.5)*0.06,
	}
	sum := phaseFactors[0] + phaseFactors[1] + phaseFactors[2]

	powerA := totalKW * phaseFactors[0] / sum
	powerB := totalKW * phaseFactors[1] / sum
	powerC := totalKW * phaseFactors[2] / sum

	// Voltages with noise
	voltAN := s.NominalVoltage * (1 + (rand.Float64()-0.5)*voltageNoise)
	voltBN := s.NominalVoltage * (1 + (rand.Float64()-0.5)*voltageNoise)
	voltCN := s.NominalVoltage * (1 + (rand.Float64()-0.5)*voltageNoise)

	// Line-to-line voltages (approx sqrt(3) * L-N)
	sqrt3 := math.Sqrt(3)
	voltAB := (voltAN + voltBN) / 2 * sqrt3
	voltBC := (voltBN + voltCN) / 2 * sqrt3
	voltCA := (voltCN + voltAN) / 2 * sqrt3

	// Currents: P = V * I * PF => I = P / (V * PF)
	pf := defaultPowerFactor
	currentA := (powerA * 1000) / (voltAN * pf) // convert kW to W
	currentB := (powerB * 1000) / (voltBN * pf)
	currentC := (powerC * 1000) / (voltCN * pf)

	// Add current noise
	currentA *= 1 + (rand.Float64()-0.5)*currentNoise
	currentB *= 1 + (rand.Float64()-0.5)*currentNoise
	currentC *= 1 + (rand.Float64()-0.5)*currentNoise

	// Frequency with noise
	freq := s.NominalFrequency * (1 + (rand.Float64()-0.5)*frequencyNoise)

	// Accumulate energy
	s.cumulativeKWH += totalKW * elapsedHours

	return MeterReading{
		CurrentA:     currentA,
		CurrentB:     currentB,
		CurrentC:     currentC,
		VoltageAN:    voltAN,
		VoltageBN:    voltBN,
		VoltageCN:    voltCN,
		VoltageAB:    voltAB,
		VoltageBC:    voltBC,
		VoltageCA:    voltCA,
		TotalPowerKW: totalKW,
		PowerA:       powerA,
		PowerB:       powerB,
		PowerC:       powerC,
		PowerFactor:  pf,
		Frequency:    freq,
		TotalKWH:     s.cumulativeKWH,
	}
}
