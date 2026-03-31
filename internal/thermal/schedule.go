package thermal

import (
	"time"
)

// OccupancyState represents the current occupancy mode.
type OccupancyState int

const (
	Occupied   OccupancyState = iota
	Unoccupied
	Standby
)

// String returns a human-readable name for the occupancy state.
func (s OccupancyState) String() string {
	switch s {
	case Occupied:
		return "occupied"
	case Unoccupied:
		return "unoccupied"
	case Standby:
		return "standby"
	default:
		return "unknown"
	}
}

// DaySchedule describes the occupancy window for a single day.
type DaySchedule struct {
	AlwaysOff bool
	Start     time.Duration // from midnight
	End       time.Duration // from midnight
}

// Schedule maps weekdays to occupancy schedules.
type Schedule struct {
	Days map[time.Weekday]DaySchedule
}

// DefaultSchedule returns a typical office schedule (M-F 6:00-18:00).
func DefaultSchedule() *Schedule {
	weekday := DaySchedule{Start: 6 * time.Hour, End: 18 * time.Hour}
	weekend := DaySchedule{AlwaysOff: true}

	return &Schedule{
		Days: map[time.Weekday]DaySchedule{
			time.Monday:    weekday,
			time.Tuesday:   weekday,
			time.Wednesday: weekday,
			time.Thursday:  weekday,
			time.Friday:    weekday,
			time.Saturday:  weekend,
			time.Sunday:    weekend,
		},
	}
}

// StateAt returns the occupancy state for the given time.
func (s *Schedule) StateAt(t time.Time) OccupancyState {
	if s == nil {
		return Occupied
	}

	day, ok := s.Days[t.Weekday()]
	if !ok {
		return Unoccupied
	}

	if day.AlwaysOff {
		return Unoccupied
	}

	// Time of day as duration from midnight
	tod := time.Duration(t.Hour())*time.Hour +
		time.Duration(t.Minute())*time.Minute +
		time.Duration(t.Second())*time.Second

	if tod >= day.Start && tod < day.End {
		return Occupied
	}

	return Unoccupied
}
