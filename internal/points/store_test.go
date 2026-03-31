package points

import (
	"sync"
	"testing"
)

func makeTestDefs() []PointDefinition {
	wp8 := 8
	wp16 := 16
	defTemp := 72.0
	defCool := 0.0
	return []PointDefinition{
		{
			ReferenceName:    "Room Temperature",
			VolttronName:     "ZoneTemperature",
			Units:            "degreesFahrenheit",
			BACnetObjectType: "analogValue",
			PropertyName:     "presentValue",
			Writable:         true,
			Index:            100,
			WritePriority:    &wp16,
			DefaultValue:     &defTemp,
			Active:           true,
		},
		{
			ReferenceName:    "Y1 Status",
			VolttronName:     "FirstStageCooling",
			Units:            "Enum",
			BACnetObjectType: "binaryOutput",
			PropertyName:     "presentValue",
			Writable:         true,
			Index:            26,
			WritePriority:    &wp16,
			DefaultValue:     &defCool,
			Active:           true,
		},
		{
			ReferenceName:    "Current Avg",
			VolttronName:     "Current",
			Units:            "amperes",
			BACnetObjectType: "analogInput",
			PropertyName:     "presentValue",
			Writable:         false,
			Index:            1141,
			Active:           true,
		},
		{
			ReferenceName:    "OAT",
			VolttronName:     "OutdoorAirTemperature",
			Units:            "degreesFahrenheit",
			BACnetObjectType: "analogValue",
			PropertyName:     "presentValue",
			Writable:         true,
			Index:            29,
			WritePriority:    &wp8,
			Active:           true,
		},
	}
}

func TestPointStore_Read(t *testing.T) {
	ps := NewPointStore(makeTestDefs())

	tests := []struct {
		name    string
		point   string
		want    *float64
		wantErr bool
	}{
		{"existing with default", "ZoneTemperature", floatPtr(72.0), false},
		{"existing with zero default", "FirstStageCooling", floatPtr(0.0), false},
		{"existing without default", "Current", nil, false},
		{"non-existent", "DoesNotExist", nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ps.Read(tc.point)
			if (err != nil) != tc.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.want == nil {
				if got != nil {
					t.Errorf("Read() = %v, want nil", *got)
				}
			} else {
				if got == nil {
					t.Errorf("Read() = nil, want %v", *tc.want)
				} else if *got != *tc.want {
					t.Errorf("Read() = %v, want %v", *got, *tc.want)
				}
			}
		})
	}
}

func TestPointStore_ReadByKey(t *testing.T) {
	ps := NewPointStore(makeTestDefs())

	v, err := ps.ReadByKey("analogValue", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v == nil || *v != 72.0 {
		t.Errorf("ReadByKey() = %v, want 72.0", v)
	}

	_, err = ps.ReadByKey("analogValue", 9999)
	if err == nil {
		t.Error("expected error for non-existent key")
	}
}

func TestPointStore_Write(t *testing.T) {
	ps := NewPointStore(makeTestDefs())

	// Write to writable point
	if err := ps.Write("ZoneTemperature", 75.0, 8); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	v, _ := ps.Read("ZoneTemperature")
	if v == nil || *v != 75.0 {
		t.Errorf("after Write, Read() = %v, want 75.0", v)
	}

	// Write to non-writable point
	err := ps.Write("Current", 10.0, 8)
	if err == nil {
		t.Error("expected error writing to non-writable point")
	}

	// Write to non-existent point
	err = ps.Write("DoesNotExist", 10.0, 8)
	if err == nil {
		t.Error("expected error writing to non-existent point")
	}

	// Write with invalid priority
	err = ps.Write("ZoneTemperature", 10.0, 0)
	if err == nil {
		t.Error("expected error for invalid priority")
	}
}

func TestPointStore_WriteByKey(t *testing.T) {
	ps := NewPointStore(makeTestDefs())

	if err := ps.WriteByKey("analogValue", 100, 80.0, 8); err != nil {
		t.Fatalf("WriteByKey() error: %v", err)
	}

	v, _ := ps.ReadByKey("analogValue", 100)
	if v == nil || *v != 80.0 {
		t.Errorf("after WriteByKey, ReadByKey() = %v, want 80.0", v)
	}

	// Non-existent key
	err := ps.WriteByKey("analogValue", 9999, 1.0, 8)
	if err == nil {
		t.Error("expected error for non-existent key")
	}
}

func TestPointStore_SetInternal(t *testing.T) {
	ps := NewPointStore(makeTestDefs())

	// SetInternal on non-writable point should work (simulation writes)
	if err := ps.SetInternal("Current", 25.5); err != nil {
		t.Fatalf("SetInternal() error: %v", err)
	}

	v, _ := ps.Read("Current")
	if v == nil || *v != 25.5 {
		t.Errorf("after SetInternal, Read() = %v, want 25.5", v)
	}

	// Non-existent point
	err := ps.SetInternal("DoesNotExist", 1.0)
	if err == nil {
		t.Error("expected error for non-existent point")
	}
}

func TestPointStore_ReadAll(t *testing.T) {
	ps := NewPointStore(makeTestDefs())

	all := ps.ReadAll()
	// Should have ZoneTemperature and FirstStageCooling (have defaults)
	if _, ok := all["ZoneTemperature"]; !ok {
		t.Error("ReadAll missing ZoneTemperature")
	}
	if _, ok := all["FirstStageCooling"]; !ok {
		t.Error("ReadAll missing FirstStageCooling")
	}
	// Current has no default
	if _, ok := all["Current"]; ok {
		t.Error("ReadAll should not include points without values")
	}

	// Modifying returned map should not affect store
	v := all["ZoneTemperature"]
	*v = 999.0
	got, _ := ps.Read("ZoneTemperature")
	if got == nil || *got != 72.0 {
		t.Error("modifying ReadAll result should not affect store")
	}
}

func TestPointStore_Definitions(t *testing.T) {
	defs := makeTestDefs()
	ps := NewPointStore(defs)

	got := ps.Definitions()
	if len(got) != len(defs) {
		t.Errorf("Definitions() returned %d, want %d", len(got), len(defs))
	}
}

func TestPointStore_ReadFloat(t *testing.T) {
	ps := NewPointStore(makeTestDefs())

	if got := ps.ReadFloat("ZoneTemperature"); got != 72.0 {
		t.Errorf("ReadFloat() = %v, want 72.0", got)
	}
	if got := ps.ReadFloat("Current"); got != 0.0 {
		t.Errorf("ReadFloat() = %v, want 0.0 for nil value", got)
	}
	if got := ps.ReadFloat("DoesNotExist"); got != 0.0 {
		t.Errorf("ReadFloat() = %v, want 0.0 for non-existent", got)
	}
}

func TestPointStore_ReadBool(t *testing.T) {
	ps := NewPointStore(makeTestDefs())

	// ZoneTemperature default 72.0 > 0
	if !ps.ReadBool("ZoneTemperature") {
		t.Error("ReadBool(ZoneTemperature) should be true")
	}
	// FirstStageCooling default 0.0
	if ps.ReadBool("FirstStageCooling") {
		t.Error("ReadBool(FirstStageCooling) should be false")
	}
}

func TestPointStore_ConcurrentAccess(t *testing.T) {
	ps := NewPointStore(makeTestDefs())

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func(val float64) {
			defer wg.Done()
			_ = ps.Write("ZoneTemperature", val, 16)
		}(float64(i))
		go func() {
			defer wg.Done()
			_, _ = ps.Read("ZoneTemperature")
		}()
		go func() {
			defer wg.Done()
			_ = ps.ReadAll()
		}()
	}
	wg.Wait()
}

func TestPointStore_PriorityInteraction(t *testing.T) {
	ps := NewPointStore(makeTestDefs())

	// Write at priority 8 (higher than default at 16)
	if err := ps.Write("ZoneTemperature", 65.0, 8); err != nil {
		t.Fatal(err)
	}
	v, _ := ps.Read("ZoneTemperature")
	if v == nil || *v != 65.0 {
		t.Errorf("after priority 8 write, got %v want 65.0", v)
	}

	// Write at priority 16 should not change active value
	if err := ps.Write("ZoneTemperature", 80.0, 16); err != nil {
		t.Fatal(err)
	}
	v, _ = ps.Read("ZoneTemperature")
	if v == nil || *v != 65.0 {
		t.Errorf("priority 8 should still win, got %v want 65.0", v)
	}
}
