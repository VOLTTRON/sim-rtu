package thermal

import (
	"testing"
	"time"
)

func TestSchedule_StateAt(t *testing.T) {
	sched := DefaultSchedule()

	tests := []struct {
		name string
		time time.Time
		want OccupancyState
	}{
		{
			name: "Monday morning occupied",
			time: time.Date(2025, 1, 6, 10, 0, 0, 0, time.UTC), // Monday
			want: Occupied,
		},
		{
			name: "Monday early morning unoccupied",
			time: time.Date(2025, 1, 6, 5, 0, 0, 0, time.UTC),
			want: Unoccupied,
		},
		{
			name: "Monday evening unoccupied",
			time: time.Date(2025, 1, 6, 19, 0, 0, 0, time.UTC),
			want: Unoccupied,
		},
		{
			name: "Saturday always unoccupied",
			time: time.Date(2025, 1, 4, 12, 0, 0, 0, time.UTC),
			want: Unoccupied,
		},
		{
			name: "Sunday always unoccupied",
			time: time.Date(2025, 1, 5, 12, 0, 0, 0, time.UTC),
			want: Unoccupied,
		},
		{
			name: "Friday at 6:00 occupied",
			time: time.Date(2025, 1, 10, 6, 0, 0, 0, time.UTC),
			want: Occupied,
		},
		{
			name: "Friday at 18:00 unoccupied",
			time: time.Date(2025, 1, 10, 18, 0, 0, 0, time.UTC),
			want: Unoccupied,
		},
		{
			name: "Wednesday at 5:59 unoccupied",
			time: time.Date(2025, 1, 8, 5, 59, 0, 0, time.UTC),
			want: Unoccupied,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sched.StateAt(tc.time)
			if got != tc.want {
				t.Errorf("StateAt(%v) = %v, want %v", tc.time, got, tc.want)
			}
		})
	}
}

func TestSchedule_NilSchedule(t *testing.T) {
	var s *Schedule
	got := s.StateAt(time.Now())
	if got != Occupied {
		t.Errorf("nil schedule should return Occupied, got %v", got)
	}
}

func TestOccupancyState_String(t *testing.T) {
	tests := []struct {
		state OccupancyState
		want  string
	}{
		{Occupied, "occupied"},
		{Unoccupied, "unoccupied"},
		{Standby, "standby"},
		{OccupancyState(99), "unknown"},
	}

	for _, tc := range tests {
		if got := tc.state.String(); got != tc.want {
			t.Errorf("OccupancyState(%d).String() = %q, want %q", tc.state, got, tc.want)
		}
	}
}
