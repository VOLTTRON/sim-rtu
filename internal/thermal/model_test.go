package thermal

import (
	"math"
	"testing"
)

func TestModel_Step(t *testing.T) {
	tests := []struct {
		name        string
		r, c        float64
		zoneTemp    float64
		outdoorTemp float64
		qHVAC       float64
		dtHours     float64
		wantApprox  float64
		tolerance   float64
	}{
		{
			name:        "no HVAC, same temps",
			r:           0.02,
			c:           1000,
			zoneTemp:    72.0,
			outdoorTemp: 72.0,
			qHVAC:       0,
			dtHours:     1.0,
			wantApprox:  72.0,
			tolerance:   0.001,
		},
		{
			name:        "no HVAC, warmer outside",
			r:           0.02,
			c:           1000,
			zoneTemp:    72.0,
			outdoorTemp: 100.0,
			qHVAC:       0,
			dtHours:     1.0,
			wantApprox:  73.4, // dT = (1/(0.02*1000))*(100-72) = 1.4
			tolerance:   0.001,
		},
		{
			name:        "cooling active",
			r:           0.02,
			c:           1000,
			zoneTemp:    78.0,
			outdoorTemp: 100.0,
			qHVAC:       -18000, // stage 1 cooling (negative = cooling)
			dtHours:     1.0 / 3600, // 1 second
			wantApprox:  78.0 + (1.0/3600)/(0.02*1000)*(100-78) + (-18000*(1.0/3600))/1000,
			tolerance:   0.01,
		},
		{
			name:        "heating active",
			r:           0.02,
			c:           1000,
			zoneTemp:    60.0,
			outdoorTemp: 30.0,
			qHVAC:       20000, // heating
			dtHours:     1.0 / 3600,
			wantApprox:  60.0 + (1.0/3600)/(0.02*1000)*(30-60) + (20000*(1.0/3600))/1000,
			tolerance:   0.01,
		},
		{
			name:        "zero dt",
			r:           0.02,
			c:           1000,
			zoneTemp:    72.0,
			outdoorTemp: 100.0,
			qHVAC:       -18000,
			dtHours:     0,
			wantApprox:  72.0,
			tolerance:   0.001,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewModel(tc.r, tc.c)
			got := m.Step(tc.zoneTemp, tc.outdoorTemp, tc.qHVAC, tc.dtHours)
			if math.Abs(got-tc.wantApprox) > tc.tolerance {
				t.Errorf("Step() = %v, want ~%v (tolerance %v)", got, tc.wantApprox, tc.tolerance)
			}
		})
	}
}

func TestModel_Immutability(t *testing.T) {
	m := NewModel(0.02, 1000)
	_ = m.Step(72, 100, 0, 1)
	// Model should not be modified by Step
	if m.R != 0.02 || m.C != 1000 {
		t.Error("Step should not modify model parameters")
	}
}
