package points

import (
	"strings"
	"testing"
)

func TestParseRegistryReader_Schneider(t *testing.T) {
	csv := `Reference Point Name,Volttron Point Name,Units,Unit Details,BACnet Object Type,Property,Writable,Index,Write Priority,Notes
Occupied Heat Setpoint,OccupiedHeatingSetPoint,degreesFahrenheit,(default 72.0),analogValue,presentValue,TRUE,39,16,
Y1 Status,FirstStageCooling,Enum,0-1 (default 0),binaryOutput,presentValue,TRUE,26,16,
`
	defs, err := ParseRegistryReader(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 2 {
		t.Fatalf("expected 2 defs, got %d", len(defs))
	}

	tests := []struct {
		name      string
		def       PointDefinition
		wantVN    string
		wantIdx   int
		wantWP    int
		wantDef   float64
		wantWrite bool
		wantObj   string
	}{
		{
			name:      "heating setpoint",
			def:       defs[0],
			wantVN:    "OccupiedHeatingSetPoint",
			wantIdx:   39,
			wantWP:    16,
			wantDef:   72.0,
			wantWrite: true,
			wantObj:   "analogValue",
		},
		{
			name:      "first stage cooling",
			def:       defs[1],
			wantVN:    "FirstStageCooling",
			wantIdx:   26,
			wantWP:    16,
			wantDef:   0.0,
			wantWrite: true,
			wantObj:   "binaryOutput",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.def.VolttronName != tc.wantVN {
				t.Errorf("VolttronName = %q, want %q", tc.def.VolttronName, tc.wantVN)
			}
			if tc.def.Index != tc.wantIdx {
				t.Errorf("Index = %d, want %d", tc.def.Index, tc.wantIdx)
			}
			if tc.def.WritePriority == nil || *tc.def.WritePriority != tc.wantWP {
				t.Errorf("WritePriority = %v, want %d", tc.def.WritePriority, tc.wantWP)
			}
			if tc.def.DefaultValue == nil || *tc.def.DefaultValue != tc.wantDef {
				t.Errorf("DefaultValue = %v, want %f", tc.def.DefaultValue, tc.wantDef)
			}
			if tc.def.Writable != tc.wantWrite {
				t.Errorf("Writable = %v, want %v", tc.def.Writable, tc.wantWrite)
			}
			if tc.def.BACnetObjectType != tc.wantObj {
				t.Errorf("BACnetObjectType = %q, want %q", tc.def.BACnetObjectType, tc.wantObj)
			}
			if !tc.def.Active {
				t.Error("Active should be true for non-openstat format")
			}
		})
	}
}

func TestParseRegistryReader_Openstat(t *testing.T) {
	csv := `Reference Point Name,Volttron Point Name,Units,Unit Details,BACnet Object Type,Property,Writable,Index,Write Priority,Notes,active
RoomT,ZoneTemperature,degreesFahrenheit,,analogInput,presentValue,TRUE,6,8,Room temperature,TRUE
AO_3,OutdoorDamperSignal,percent,(default 0.0),analogOutput,presentValue,TRUE,3,8,ECON,FALSE
`
	defs, err := ParseRegistryReader(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 2 {
		t.Fatalf("expected 2 defs, got %d", len(defs))
	}

	if defs[0].VolttronName != "ZoneTemperature" {
		t.Errorf("first def VolttronName = %q, want ZoneTemperature", defs[0].VolttronName)
	}
	if !defs[0].Active {
		t.Error("first def should be active")
	}
	if defs[1].Active {
		t.Error("second def should not be active")
	}
	if defs[1].DefaultValue == nil || *defs[1].DefaultValue != 0.0 {
		t.Errorf("second def DefaultValue = %v, want 0.0", defs[1].DefaultValue)
	}
}

func TestParseRegistryReader_Dent(t *testing.T) {
	csv := `Reference Point Name,Volttron Point Name,Units,Unit Details,BACnet Object Type,Property,Writable,Index,Write Priority,Notes
Current Avg Element A,Current,amperes,,analogInput,presentValue,FALSE,1141,,
`
	defs, err := ParseRegistryReader(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 def, got %d", len(defs))
	}

	d := defs[0]
	if d.VolttronName != "Current" {
		t.Errorf("VolttronName = %q, want Current", d.VolttronName)
	}
	if d.Writable {
		t.Error("dent points should not be writable")
	}
	if d.WritePriority != nil {
		t.Error("WritePriority should be nil for non-writable point")
	}
	if d.DefaultValue != nil {
		t.Error("DefaultValue should be nil when no default in unit details")
	}
}

func TestParseRegistryReader_InvalidIndex(t *testing.T) {
	csv := `Reference Point Name,Volttron Point Name,Units,Unit Details,BACnet Object Type,Property,Writable,Index,Write Priority,Notes
Test,Test,,,analogInput,presentValue,FALSE,abc,,
`
	_, err := ParseRegistryReader(strings.NewReader(csv))
	if err == nil {
		t.Fatal("expected error for invalid index")
	}
}

func TestParseRegistryReader_EmptyInput(t *testing.T) {
	csv := `Reference Point Name,Volttron Point Name,Units,Unit Details,BACnet Object Type,Property,Writable,Index,Write Priority,Notes
`
	defs, err := ParseRegistryReader(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 0 {
		t.Fatalf("expected 0 defs, got %d", len(defs))
	}
}

func TestParseRegistryReader_DefaultValueParsing(t *testing.T) {
	tests := []struct {
		name        string
		unitDetails string
		wantDefault *float64
	}{
		{"positive default", "(default 72.0)", floatPtr(72.0)},
		{"negative default", "(default -40.0)", floatPtr(-40.0)},
		{"no default", "0-1", nil},
		{"empty", "", nil},
		{"fractional", "(default 14.0)", floatPtr(14.0)},
		{"long float", "(default 68.70000457763672)", floatPtr(68.70000457763672)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			csv := "Reference Point Name,Volttron Point Name,Units,Unit Details,BACnet Object Type,Property,Writable,Index,Write Priority,Notes\n" +
				"Test,Test,units," + tc.unitDetails + ",analogValue,presentValue,FALSE,1,,\n"

			defs, err := ParseRegistryReader(strings.NewReader(csv))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(defs) != 1 {
				t.Fatalf("expected 1 def, got %d", len(defs))
			}
			got := defs[0].DefaultValue
			if tc.wantDefault == nil {
				if got != nil {
					t.Errorf("DefaultValue = %v, want nil", *got)
				}
			} else {
				if got == nil {
					t.Errorf("DefaultValue = nil, want %v", *tc.wantDefault)
				} else if *got != *tc.wantDefault {
					t.Errorf("DefaultValue = %v, want %v", *got, *tc.wantDefault)
				}
			}
		})
	}
}

func floatPtr(f float64) *float64 {
	return &f
}
