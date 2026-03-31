package thermal

import (
	"sync"
	"time"
)

// HVACState represents the current state of the HVAC system.
type HVACState struct {
	FirstStageCooling  bool
	SecondStageCooling bool
	FirstStageHeating  bool
	SecondStageHeating bool
	FanStatus          bool
	HeatingDemand      float64 // 0-100%
	CoolingDemand      float64 // 0-100%
	EconomizerDemand   float64 // 0-100%
}

// ActiveCoolingStages returns the number of active cooling stages (0, 1, or 2).
func (s HVACState) ActiveCoolingStages() int {
	n := 0
	if s.FirstStageCooling {
		n++
	}
	if s.SecondStageCooling {
		n++
	}
	return n
}

// ActiveHeatingStages returns the number of active heating stages (0, 1, or 2).
func (s HVACState) ActiveHeatingStages() int {
	n := 0
	if s.FirstStageHeating {
		n++
	}
	if s.SecondStageHeating {
		n++
	}
	return n
}

// TotalActiveStages returns the total active HVAC stages.
func (s HVACState) TotalActiveStages() int {
	return s.ActiveCoolingStages() + s.ActiveHeatingStages()
}

// Controller is a multi-stage thermostat controller with deadband
// and anti-short-cycle protection. It is safe for concurrent use.
type Controller struct {
	mu                 sync.RWMutex
	DeadBand           float64
	ProportionalBand   float64
	AntiShortCycleTime time.Duration

	lastStageChangeTime time.Time
	currentState        HVACState
	stageChanged        bool
}

// NewController creates a controller with the given parameters.
func NewController(deadBand, proportionalBand float64, antiShortCycleTime time.Duration) *Controller {
	return &Controller{
		DeadBand:           deadBand,
		ProportionalBand:   proportionalBand,
		AntiShortCycleTime: antiShortCycleTime,
	}
}

// Evaluate determines the new HVAC state based on zone conditions.
func (c *Controller) Evaluate(zoneTemp, coolingSetpoint, heatingSetpoint, outdoorTemp float64, now time.Time) HVACState {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Anti-short-cycle check
	if c.stageChanged && now.Sub(c.lastStageChangeTime) < c.AntiShortCycleTime {
		return c.currentState
	}

	var state HVACState
	coolThreshold := coolingSetpoint + c.DeadBand/2
	heatThreshold := heatingSetpoint - c.DeadBand/2

	if zoneTemp > coolingSetpoint {
		// Cooling mode
		deviation := zoneTemp - coolingSetpoint
		demand := clamp((deviation/c.ProportionalBand)*100, 0, 100)
		state.CoolingDemand = demand

		state.FirstStageCooling = true
		if demand > 50 {
			state.SecondStageCooling = true
		}

		// Economizer: use outdoor air if it's cooler than zone
		if outdoorTemp < zoneTemp {
			econDemand := clamp(((zoneTemp-outdoorTemp)/c.ProportionalBand)*100, 0, 100)
			state.EconomizerDemand = econDemand
		}
	} else if zoneTemp < heatingSetpoint {
		// Heating mode
		deviation := heatingSetpoint - zoneTemp
		demand := clamp((deviation/c.ProportionalBand)*100, 0, 100)
		state.HeatingDemand = demand

		state.FirstStageHeating = true
		if demand > 50 {
			state.SecondStageHeating = true
		}
	} else if zoneTemp > coolThreshold {
		// In deadband but leaning warm — keep stage 1 cooling if already on
		if c.currentState.FirstStageCooling {
			state.FirstStageCooling = true
			deviation := zoneTemp - coolingSetpoint
			state.CoolingDemand = clamp((deviation/c.ProportionalBand)*100, 0, 100)
		}
	} else if zoneTemp < heatThreshold {
		// In deadband but leaning cool — keep stage 1 heating if already on
		if c.currentState.FirstStageHeating {
			state.FirstStageHeating = true
			deviation := heatingSetpoint - zoneTemp
			state.HeatingDemand = clamp((deviation/c.ProportionalBand)*100, 0, 100)
		}
	}

	// Fan runs whenever any stage is active
	state.FanStatus = state.FirstStageCooling || state.SecondStageCooling ||
		state.FirstStageHeating || state.SecondStageHeating

	// Track stage changes for anti-short-cycle
	if stagesChanged(c.currentState, state) {
		c.lastStageChangeTime = now
		c.stageChanged = true
	}

	c.currentState = state
	return state
}

// CurrentState returns the controller's current HVAC state.
func (c *Controller) CurrentState() HVACState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentState
}

func stagesChanged(old, new HVACState) bool {
	return old.FirstStageCooling != new.FirstStageCooling ||
		old.SecondStageCooling != new.SecondStageCooling ||
		old.FirstStageHeating != new.FirstStageHeating ||
		old.SecondStageHeating != new.SecondStageHeating
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
