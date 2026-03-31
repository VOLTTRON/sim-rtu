package points

import (
	"testing"
)

func TestPriorityArray_Write(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		value    float64
		wantErr  bool
	}{
		{"valid priority 1", 1, 42.0, false},
		{"valid priority 16", 16, 99.0, false},
		{"valid priority 8", 8, 55.0, false},
		{"invalid priority 0", 0, 1.0, true},
		{"invalid priority 17", 17, 1.0, true},
		{"invalid priority -1", -1, 1.0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pa := NewPriorityArray(nil)
			err := pa.Write(tc.priority, tc.value)
			if (err != nil) != tc.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestPriorityArray_ActiveValue(t *testing.T) {
	tests := []struct {
		name       string
		defaultVal *float64
		writes     []struct{ priority int; value float64 }
		want       *float64
	}{
		{
			name:       "empty with no default",
			defaultVal: nil,
			writes:     nil,
			want:       nil,
		},
		{
			name:       "empty with default",
			defaultVal: floatPtr(42.0),
			writes:     nil,
			want:       floatPtr(42.0),
		},
		{
			name:       "single write",
			defaultVal: nil,
			writes:     []struct{ priority int; value float64 }{{8, 55.0}},
			want:       floatPtr(55.0),
		},
		{
			name:       "higher priority wins",
			defaultVal: nil,
			writes: []struct{ priority int; value float64 }{
				{16, 99.0},
				{8, 55.0},
			},
			want: floatPtr(55.0),
		},
		{
			name:       "lowest number is highest priority",
			defaultVal: nil,
			writes: []struct{ priority int; value float64 }{
				{16, 99.0},
				{1, 10.0},
				{8, 55.0},
			},
			want: floatPtr(10.0),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pa := NewPriorityArray(tc.defaultVal)
			for _, w := range tc.writes {
				if err := pa.Write(w.priority, w.value); err != nil {
					t.Fatalf("Write(%d, %f) error: %v", w.priority, w.value, err)
				}
			}

			got := pa.ActiveValue()
			if tc.want == nil {
				if got != nil {
					t.Errorf("ActiveValue() = %v, want nil", *got)
				}
			} else {
				if got == nil {
					t.Errorf("ActiveValue() = nil, want %v", *tc.want)
				} else if *got != *tc.want {
					t.Errorf("ActiveValue() = %v, want %v", *got, *tc.want)
				}
			}
		})
	}
}

func TestPriorityArray_Relinquish(t *testing.T) {
	pa := NewPriorityArray(floatPtr(100.0))

	if err := pa.Write(8, 50.0); err != nil {
		t.Fatal(err)
	}
	if err := pa.Write(16, 75.0); err != nil {
		t.Fatal(err)
	}

	// Active should be priority 8
	v := pa.ActiveValue()
	if v == nil || *v != 50.0 {
		t.Errorf("ActiveValue() = %v, want 50.0", v)
	}

	// Relinquish priority 8
	if err := pa.Relinquish(8); err != nil {
		t.Fatal(err)
	}

	// Active should fall to priority 16
	v = pa.ActiveValue()
	if v == nil || *v != 75.0 {
		t.Errorf("ActiveValue() = %v, want 75.0", v)
	}

	// Relinquish priority 16
	if err := pa.Relinquish(16); err != nil {
		t.Fatal(err)
	}

	// Active should fall to default
	v = pa.ActiveValue()
	if v == nil || *v != 100.0 {
		t.Errorf("ActiveValue() = %v, want 100.0 (default)", v)
	}

	// Relinquish invalid
	if err := pa.Relinquish(0); err == nil {
		t.Error("expected error for priority 0")
	}
	if err := pa.Relinquish(17); err == nil {
		t.Error("expected error for priority 17")
	}
}

func TestPriorityArray_Slots(t *testing.T) {
	pa := NewPriorityArray(nil)
	_ = pa.Write(1, 10.0)
	_ = pa.Write(8, 80.0)

	slots := pa.Slots()

	if slots[0] == nil || *slots[0] != 10.0 {
		t.Errorf("slot[0] = %v, want 10.0", slots[0])
	}
	if slots[7] == nil || *slots[7] != 80.0 {
		t.Errorf("slot[7] = %v, want 80.0", slots[7])
	}
	for i, s := range slots {
		if i != 0 && i != 7 && s != nil {
			t.Errorf("slot[%d] = %v, want nil", i, *s)
		}
	}

	// Verify returned slots are copies (immutable)
	*slots[0] = 999.0
	v := pa.ActiveValue()
	if v == nil || *v != 10.0 {
		t.Error("modifying returned slot should not affect internal state")
	}
}
