package points

import (
	"fmt"
)

// PriorityArray implements a BACnet 16-level priority array.
// Slot 1 is highest priority, slot 16 is lowest.
type PriorityArray struct {
	slots        [16]*float64
	defaultValue *float64
}

// NewPriorityArray creates a priority array with an optional default value.
func NewPriorityArray(defaultValue *float64) *PriorityArray {
	return &PriorityArray{defaultValue: defaultValue}
}

// Write sets a value at the given priority level (1-16).
func (pa *PriorityArray) Write(priority int, value float64) error {
	if priority < 1 || priority > 16 {
		return fmt.Errorf("priority %d out of range [1,16]", priority)
	}
	v := value
	pa.slots[priority-1] = &v
	return nil
}

// Relinquish clears the value at the given priority level (1-16).
func (pa *PriorityArray) Relinquish(priority int) error {
	if priority < 1 || priority > 16 {
		return fmt.Errorf("priority %d out of range [1,16]", priority)
	}
	pa.slots[priority-1] = nil
	return nil
}

// ActiveValue returns the value at the highest (lowest-numbered) occupied
// priority slot. If no slots are occupied, returns the default value.
func (pa *PriorityArray) ActiveValue() *float64 {
	for _, v := range pa.slots {
		if v != nil {
			return v
		}
	}
	return pa.defaultValue
}

// Slots returns a copy of all 16 priority slots for inspection.
func (pa *PriorityArray) Slots() [16]*float64 {
	var out [16]*float64
	for i, v := range pa.slots {
		if v != nil {
			cp := *v
			out[i] = &cp
		}
	}
	return out
}
