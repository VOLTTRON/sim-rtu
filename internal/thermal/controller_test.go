package thermal

import (
	"testing"
	"time"
)

func TestController_Evaluate(t *testing.T) {
	now := time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name            string
		zoneTemp        float64
		coolingSetpoint float64
		heatingSetpoint float64
		outdoorTemp     float64
		wantCool1       bool
		wantCool2       bool
		wantHeat1       bool
		wantHeat2       bool
		wantFan         bool
	}{
		{
			name:            "cooling stage 1 only",
			zoneTemp:        76.0,
			coolingSetpoint: 75.0,
			heatingSetpoint: 72.0,
			outdoorTemp:     90.0,
			wantCool1:       true,
			wantCool2:       false,
			wantHeat1:       false,
			wantHeat2:       false,
			wantFan:         true,
		},
		{
			name:            "cooling both stages",
			zoneTemp:        80.0,
			coolingSetpoint: 75.0,
			heatingSetpoint: 72.0,
			outdoorTemp:     95.0,
			wantCool1:       true,
			wantCool2:       true,
			wantHeat1:       false,
			wantHeat2:       false,
			wantFan:         true,
		},
		{
			name:            "heating stage 1 only",
			zoneTemp:        71.0,
			coolingSetpoint: 75.0,
			heatingSetpoint: 72.0,
			outdoorTemp:     30.0,
			wantCool1:       false,
			wantCool2:       false,
			wantHeat1:       true,
			wantHeat2:       false,
			wantFan:         true,
		},
		{
			name:            "heating both stages",
			zoneTemp:        66.0,
			coolingSetpoint: 75.0,
			heatingSetpoint: 72.0,
			outdoorTemp:     20.0,
			wantCool1:       false,
			wantCool2:       false,
			wantHeat1:       true,
			wantHeat2:       true,
			wantFan:         true,
		},
		{
			name:            "in deadband, no action",
			zoneTemp:        73.0,
			coolingSetpoint: 75.0,
			heatingSetpoint: 72.0,
			outdoorTemp:     72.0,
			wantCool1:       false,
			wantCool2:       false,
			wantHeat1:       false,
			wantHeat2:       false,
			wantFan:         false,
		},
		{
			name:            "exactly at cooling setpoint",
			zoneTemp:        75.0,
			coolingSetpoint: 75.0,
			heatingSetpoint: 72.0,
			outdoorTemp:     80.0,
			wantCool1:       false, // not above setpoint
			wantCool2:       false,
			wantHeat1:       false,
			wantHeat2:       false,
			wantFan:         false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := NewController(3.0, 3.0, 2*time.Minute)
			state := c.Evaluate(tc.zoneTemp, tc.coolingSetpoint, tc.heatingSetpoint, tc.outdoorTemp, now)

			if state.FirstStageCooling != tc.wantCool1 {
				t.Errorf("FirstStageCooling = %v, want %v", state.FirstStageCooling, tc.wantCool1)
			}
			if state.SecondStageCooling != tc.wantCool2 {
				t.Errorf("SecondStageCooling = %v, want %v", state.SecondStageCooling, tc.wantCool2)
			}
			if state.FirstStageHeating != tc.wantHeat1 {
				t.Errorf("FirstStageHeating = %v, want %v", state.FirstStageHeating, tc.wantHeat1)
			}
			if state.SecondStageHeating != tc.wantHeat2 {
				t.Errorf("SecondStageHeating = %v, want %v", state.SecondStageHeating, tc.wantHeat2)
			}
			if state.FanStatus != tc.wantFan {
				t.Errorf("FanStatus = %v, want %v", state.FanStatus, tc.wantFan)
			}
		})
	}
}

func TestController_AntiShortCycle(t *testing.T) {
	c := NewController(3.0, 3.0, 2*time.Minute)
	now := time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC)

	// Start cooling
	state1 := c.Evaluate(78.0, 75.0, 72.0, 90.0, now)
	if !state1.FirstStageCooling {
		t.Fatal("should start cooling")
	}

	// Try to change within anti-short-cycle window (30 seconds later)
	state2 := c.Evaluate(71.0, 75.0, 72.0, 90.0, now.Add(30*time.Second))
	if state2.FirstStageCooling != state1.FirstStageCooling {
		t.Error("anti-short-cycle should prevent stage change within window")
	}

	// After anti-short-cycle window (3 minutes later)
	state3 := c.Evaluate(71.0, 75.0, 72.0, 30.0, now.Add(3*time.Minute))
	if state3.FirstStageCooling {
		t.Error("should have stopped cooling after anti-short-cycle window")
	}
	if !state3.FirstStageHeating {
		t.Error("should have started heating after anti-short-cycle window")
	}
}

func TestController_EconomizerDemand(t *testing.T) {
	c := NewController(3.0, 3.0, 0)
	now := time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC)

	// Outdoor temp cooler than zone — economizer should activate
	state := c.Evaluate(78.0, 75.0, 72.0, 70.0, now)
	if state.EconomizerDemand <= 0 {
		t.Error("economizer should be active when outdoor < zone")
	}

	// Outdoor temp warmer than zone — no economizer
	c2 := NewController(3.0, 3.0, 0)
	state2 := c2.Evaluate(78.0, 75.0, 72.0, 90.0, now)
	if state2.EconomizerDemand != 0 {
		t.Error("economizer should be inactive when outdoor > zone")
	}
}

func TestHVACState_ActiveStages(t *testing.T) {
	tests := []struct {
		name        string
		state       HVACState
		wantCooling int
		wantHeating int
		wantTotal   int
	}{
		{
			name:        "no stages",
			state:       HVACState{},
			wantCooling: 0,
			wantHeating: 0,
			wantTotal:   0,
		},
		{
			name:        "stage 1 cooling",
			state:       HVACState{FirstStageCooling: true},
			wantCooling: 1,
			wantHeating: 0,
			wantTotal:   1,
		},
		{
			name:        "both cooling stages",
			state:       HVACState{FirstStageCooling: true, SecondStageCooling: true},
			wantCooling: 2,
			wantHeating: 0,
			wantTotal:   2,
		},
		{
			name:        "both heating stages",
			state:       HVACState{FirstStageHeating: true, SecondStageHeating: true},
			wantCooling: 0,
			wantHeating: 2,
			wantTotal:   2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.state.ActiveCoolingStages(); got != tc.wantCooling {
				t.Errorf("ActiveCoolingStages() = %d, want %d", got, tc.wantCooling)
			}
			if got := tc.state.ActiveHeatingStages(); got != tc.wantHeating {
				t.Errorf("ActiveHeatingStages() = %d, want %d", got, tc.wantHeating)
			}
			if got := tc.state.TotalActiveStages(); got != tc.wantTotal {
				t.Errorf("TotalActiveStages() = %d, want %d", got, tc.wantTotal)
			}
		})
	}
}
