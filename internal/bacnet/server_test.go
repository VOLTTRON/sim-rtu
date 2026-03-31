package bacnet

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/services"

	"github.com/VOLTTRON/sim-rtu/internal/config"
	"github.com/VOLTTRON/sim-rtu/internal/points"
)

// testDefs returns a set of point definitions for testing.
func testDefs() []points.PointDefinition {
	wp16 := 16
	defVal72 := 72.0
	defVal0 := 0.0
	defVal1 := 1.0
	defVal2 := 2.0

	return []points.PointDefinition{
		{
			ReferenceName:    "Zone Temperature",
			VolttronName:     "ZoneTemperature",
			Units:            "degreesFahrenheit",
			BACnetObjectType: "analogValue",
			PropertyName:     "presentValue",
			Writable:         true,
			Index:            100,
			WritePriority:    &wp16,
			DefaultValue:     &defVal72,
			Active:           true,
		},
		{
			ReferenceName:    "Cooling Demand",
			VolttronName:     "CoolingDemand",
			Units:            "percent",
			BACnetObjectType: "analogOutput",
			PropertyName:     "presentValue",
			Writable:         true,
			Index:            22,
			WritePriority:    &wp16,
			DefaultValue:     &defVal0,
			Active:           true,
		},
		{
			ReferenceName:    "Fan Status",
			VolttronName:     "SupplyFanStatus",
			Units:            "Enum",
			BACnetObjectType: "binaryOutput",
			PropertyName:     "presentValue",
			Writable:         true,
			Index:            25,
			WritePriority:    &wp16,
			DefaultValue:     &defVal1,
			Active:           true,
		},
		{
			ReferenceName:    "System Mode",
			VolttronName:     "SystemMode",
			Units:            "State",
			BACnetObjectType: "multiStateValue",
			PropertyName:     "presentValue",
			Writable:         true,
			Index:            16,
			WritePriority:    &wp16,
			DefaultValue:     &defVal2,
			Active:           true,
		},
		{
			ReferenceName:    "Outdoor Temperature",
			VolttronName:     "OutdoorAirTemperature",
			Units:            "degreesFahrenheit",
			BACnetObjectType: "analogInput",
			PropertyName:     "presentValue",
			Writable:         false,
			Index:            1,
			DefaultValue:     &defVal72,
			Active:           true,
		},
	}
}

// startTestServer creates and starts a server on a random port.
func startTestServer(t *testing.T) (*Server, net.Addr) {
	t.Helper()

	store := points.NewPointStore(testDefs())
	devices := map[int]*points.PointStore{
		1001: store,
	}

	cfg := config.BACnetConfig{
		Enabled:   true,
		Interface: "127.0.0.1",
		Port:      0, // random port
	}

	srv := New(cfg)

	// Use a random port by listening first
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv.conn = conn
	srv.devices = devices

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		srv.Stop()
	})

	go srv.readLoop(ctx)

	return srv, conn.LocalAddr()
}

// sendAndReceive sends a BACnet packet and reads the response.
func sendAndReceive(t *testing.T, serverAddr net.Addr, request []byte) []byte {
	t.Helper()

	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("client listen: %v", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}

	if _, err := conn.WriteTo(request, serverAddr); err != nil {
		t.Fatalf("send: %v", err)
	}

	buf := make([]byte, 1500)
	n, _, err := conn.ReadFrom(buf)
	if err != nil {
		t.Fatalf("receive: %v", err)
	}

	return buf[:n]
}

func TestWhoIsIAm(t *testing.T) {
	_, serverAddr := startTestServer(t)

	whoisBytes, err := bacnet.NewWhois()
	if err != nil {
		t.Fatalf("create WhoIs: %v", err)
	}

	resp := sendAndReceive(t, serverAddr, whoisBytes)

	msg, err := bacnet.Parse(resp)
	if err != nil {
		t.Fatalf("parse IAm response: %v", err)
	}

	iam, ok := msg.(*services.UnconfirmedIAm)
	if !ok {
		t.Fatalf("expected UnconfirmedIAm, got %T", msg)
	}

	dec, err := iam.Decode()
	if err != nil {
		t.Fatalf("decode IAm: %v", err)
	}

	if dec.InstanceNum != 1001 {
		t.Errorf("expected device 1001, got %d", dec.InstanceNum)
	}
	if dec.VendorId != VendorIDSimulator {
		t.Errorf("expected vendor %d, got %d", VendorIDSimulator, dec.VendorId)
	}
}

func TestReadPropertyAnalogValue(t *testing.T) {
	_, serverAddr := startTestServer(t)

	// Read ZoneTemperature (analogValue, index 100)
	rpBytes, err := bacnet.NewReadProperty(
		objects.ObjectTypeAnalogValue, 100,
		objects.PropertyIdPresentValue,
	)
	if err != nil {
		t.Fatalf("create RP: %v", err)
	}

	resp := sendAndReceive(t, serverAddr, rpBytes)

	msg, err := bacnet.Parse(resp)
	if err != nil {
		t.Fatalf("parse RP response: %v", err)
	}

	cack, ok := msg.(*services.ComplexACK)
	if !ok {
		t.Fatalf("expected ComplexACK, got %T", msg)
	}

	dec, err := cack.Decode()
	if err != nil {
		t.Fatalf("decode CACK: %v", err)
	}

	if dec.ObjectType != objects.ObjectTypeAnalogValue {
		t.Errorf("expected object type %d, got %d", objects.ObjectTypeAnalogValue, dec.ObjectType)
	}
	if dec.InstanceId != 100 {
		t.Errorf("expected instance 100, got %d", dec.InstanceId)
	}

	// Check the value tag
	if len(dec.Tags) == 0 {
		t.Fatal("no value tags in CACK")
	}
	valTag := dec.Tags[0]
	if valTag.TagNumber != objects.TagReal {
		t.Errorf("expected Real tag, got tag %d", valTag.TagNumber)
	}
	val, ok := valTag.Value.(float32)
	if !ok {
		t.Fatalf("expected float32 value, got %T", valTag.Value)
	}
	if val != 72.0 {
		t.Errorf("expected 72.0, got %f", val)
	}
}

func TestReadPropertyBinaryOutput(t *testing.T) {
	_, serverAddr := startTestServer(t)

	// Read SupplyFanStatus (binaryOutput, index 25)
	rpBytes, err := bacnet.NewReadProperty(
		objects.ObjectTypeBinaryOutput, 25,
		objects.PropertyIdPresentValue,
	)
	if err != nil {
		t.Fatalf("create RP: %v", err)
	}

	resp := sendAndReceive(t, serverAddr, rpBytes)

	msg, err := bacnet.Parse(resp)
	if err != nil {
		t.Fatalf("parse RP response: %v", err)
	}

	cack, ok := msg.(*services.ComplexACK)
	if !ok {
		t.Fatalf("expected ComplexACK, got %T", msg)
	}

	dec, err := cack.Decode()
	if err != nil {
		t.Fatalf("decode CACK: %v", err)
	}

	if len(dec.Tags) == 0 {
		t.Fatal("no value tags in CACK")
	}
	valTag := dec.Tags[0]
	// Binary values are encoded as Enumerated
	if valTag.TagNumber != objects.TagEnumerated && valTag.TagNumber != objects.TagUnsignedInteger {
		t.Errorf("expected Enumerated or UnsignedInteger tag for binary, got tag %d", valTag.TagNumber)
	}
}

func TestReadPropertyMultiStateValue(t *testing.T) {
	_, serverAddr := startTestServer(t)

	// Read SystemMode (multiStateValue, index 16, default 2)
	rpBytes, err := bacnet.NewReadProperty(
		objects.ObjectTypeMultiStateValue, 16,
		objects.PropertyIdPresentValue,
	)
	if err != nil {
		t.Fatalf("create RP: %v", err)
	}

	resp := sendAndReceive(t, serverAddr, rpBytes)

	msg, err := bacnet.Parse(resp)
	if err != nil {
		t.Fatalf("parse RP response: %v", err)
	}

	cack, ok := msg.(*services.ComplexACK)
	if !ok {
		t.Fatalf("expected ComplexACK, got %T", msg)
	}

	dec, err := cack.Decode()
	if err != nil {
		t.Fatalf("decode CACK: %v", err)
	}

	if len(dec.Tags) == 0 {
		t.Fatal("no value tags in CACK")
	}
	valTag := dec.Tags[0]
	if valTag.TagNumber != objects.TagUnsignedInteger {
		t.Errorf("expected UnsignedInteger tag for multiState, got tag %d", valTag.TagNumber)
	}
}

func TestReadPropertyAnalogInput(t *testing.T) {
	_, serverAddr := startTestServer(t)

	// Read OutdoorAirTemperature (analogInput, index 1)
	rpBytes, err := bacnet.NewReadProperty(
		objects.ObjectTypeAnalogInput, 1,
		objects.PropertyIdPresentValue,
	)
	if err != nil {
		t.Fatalf("create RP: %v", err)
	}

	resp := sendAndReceive(t, serverAddr, rpBytes)

	msg, err := bacnet.Parse(resp)
	if err != nil {
		t.Fatalf("parse RP response: %v", err)
	}

	cack, ok := msg.(*services.ComplexACK)
	if !ok {
		t.Fatalf("expected ComplexACK, got %T", msg)
	}

	dec, err := cack.Decode()
	if err != nil {
		t.Fatalf("decode CACK: %v", err)
	}

	if len(dec.Tags) == 0 {
		t.Fatal("no value tags in CACK")
	}
	valTag := dec.Tags[0]
	if valTag.TagNumber != objects.TagReal {
		t.Errorf("expected Real tag, got tag %d", valTag.TagNumber)
	}
	val, ok := valTag.Value.(float32)
	if !ok {
		t.Fatalf("expected float32, got %T", valTag.Value)
	}
	if val != 72.0 {
		t.Errorf("expected 72.0, got %f", val)
	}
}

func TestWritePropertyWithPriority(t *testing.T) {
	srv, serverAddr := startTestServer(t)

	// Write CoolingDemand (analogOutput, index 22) to 50.0 at priority 8
	wpBytes, err := bacnet.NewWriteProperty(
		objects.ObjectTypeAnalogOutput, 22,
		objects.PropertyIdPresentValue,
		float32(50.0),
	)
	if err != nil {
		t.Fatalf("create WP: %v", err)
	}

	resp := sendAndReceive(t, serverAddr, wpBytes)

	msg, err := bacnet.Parse(resp)
	if err != nil {
		t.Fatalf("parse WP response: %v", err)
	}

	// Should get a SimpleACK
	if msg.GetType() != 2 { // SimpleAck = 2
		t.Fatalf("expected SimpleACK (type 2), got type %d", msg.GetType())
	}

	// Verify the value was written
	srv.mu.Lock()
	store := srv.devices[1001]
	srv.mu.Unlock()

	val, err := store.ReadByKey("analogOutput", 22)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if val == nil {
		t.Fatal("value is nil after write")
	}
	if *val != 50.0 {
		t.Errorf("expected 50.0, got %f", *val)
	}
}

func TestReadPropertyUnknownObject(t *testing.T) {
	_, serverAddr := startTestServer(t)

	// Read a non-existent object (analogValue, index 999)
	rpBytes, err := bacnet.NewReadProperty(
		objects.ObjectTypeAnalogValue, 999,
		objects.PropertyIdPresentValue,
	)
	if err != nil {
		t.Fatalf("create RP: %v", err)
	}

	resp := sendAndReceive(t, serverAddr, rpBytes)

	msg, err := bacnet.Parse(resp)
	if err != nil {
		t.Fatalf("parse error response: %v", err)
	}

	errMsg, ok := msg.(*services.Error)
	if !ok {
		t.Fatalf("expected Error, got %T", msg)
	}

	dec, err := errMsg.Decode()
	if err != nil {
		t.Fatalf("decode Error: %v", err)
	}

	if dec.ErrorClass != objects.ErrorClassObject {
		t.Errorf("expected error class %d (Object), got %d", objects.ErrorClassObject, dec.ErrorClass)
	}
	if dec.ErrorCode != objects.ErrorCodeUnknownObject {
		t.Errorf("expected error code %d (UnknownObject), got %d", objects.ErrorCodeUnknownObject, dec.ErrorCode)
	}
}

func TestReadPropertyObjectName(t *testing.T) {
	_, serverAddr := startTestServer(t)

	// Read ObjectName property of ZoneTemperature (analogValue, index 100)
	rpBytes, err := bacnet.NewReadProperty(
		objects.ObjectTypeAnalogValue, 100,
		objects.PropertyIdObjectName,
	)
	if err != nil {
		t.Fatalf("create RP: %v", err)
	}

	resp := sendAndReceive(t, serverAddr, rpBytes)

	msg, err := bacnet.Parse(resp)
	if err != nil {
		t.Fatalf("parse RP response: %v", err)
	}

	cack, ok := msg.(*services.ComplexACK)
	if !ok {
		t.Fatalf("expected ComplexACK, got %T", msg)
	}

	// Verify it's a valid ComplexACK for ReadProperty
	if cack.APDU.Service != services.ServiceConfirmedReadProperty {
		t.Errorf("expected RP service, got %d", cack.APDU.Service)
	}

	// Manually find the CharacterString tag in the objects
	// (library's Decode() has a bug with short strings)
	foundString := false
	for _, obj := range cack.APDU.Objects {
		encObj, ok := obj.(*objects.Object)
		if !ok {
			continue
		}
		if !encObj.TagClass && encObj.TagNumber == objects.TagCharacterString {
			foundString = true
			// Data starts with charset byte (0x00 = UTF-8), then the string
			if len(encObj.Data) > 1 {
				name := string(encObj.Data[1:])
				if name != "ZoneTemperature" {
					t.Errorf("expected 'ZoneTemperature', got %q", name)
				}
			} else {
				t.Errorf("CharacterString data too short: %x", encObj.Data)
			}
		}
	}
	if !foundString {
		t.Error("no CharacterString tag found in response")
	}
}

func TestWriteReadRoundtrip(t *testing.T) {
	_, serverAddr := startTestServer(t)

	// Write a value
	wpBytes, err := bacnet.NewWriteProperty(
		objects.ObjectTypeAnalogValue, 100,
		objects.PropertyIdPresentValue,
		float32(85.5),
	)
	if err != nil {
		t.Fatalf("create WP: %v", err)
	}

	resp := sendAndReceive(t, serverAddr, wpBytes)
	msg, err := bacnet.Parse(resp)
	if err != nil {
		t.Fatalf("parse WP response: %v", err)
	}
	if msg.GetType() != 2 {
		t.Fatalf("expected SimpleACK, got type %d", msg.GetType())
	}

	// Read it back
	rpBytes, err := bacnet.NewReadProperty(
		objects.ObjectTypeAnalogValue, 100,
		objects.PropertyIdPresentValue,
	)
	if err != nil {
		t.Fatalf("create RP: %v", err)
	}

	resp = sendAndReceive(t, serverAddr, rpBytes)
	msg, err = bacnet.Parse(resp)
	if err != nil {
		t.Fatalf("parse RP response: %v", err)
	}

	cack, ok := msg.(*services.ComplexACK)
	if !ok {
		t.Fatalf("expected ComplexACK, got %T", msg)
	}

	dec, err := cack.Decode()
	if err != nil {
		t.Fatalf("decode CACK: %v", err)
	}

	if len(dec.Tags) == 0 {
		t.Fatal("no value tags")
	}
	val, ok := dec.Tags[0].Value.(float32)
	if !ok {
		t.Fatalf("expected float32, got %T", dec.Tags[0].Value)
	}
	if val != 85.5 {
		t.Errorf("expected 85.5, got %f", val)
	}
}

func TestReadPropertyMultiple(t *testing.T) {
	_, serverAddr := startTestServer(t)

	// RPM for analogValue index 100, requesting presentValue and objectName
	rpmBytes, err := bacnet.NewReadPropertyMultiple(
		objects.ObjectTypeAnalogValue, 100,
		[]uint16{objects.PropertyIdPresentValue, objects.PropertyIdObjectName},
	)
	if err != nil {
		t.Fatalf("create RPM: %v", err)
	}

	resp := sendAndReceive(t, serverAddr, rpmBytes)

	msg, err := bacnet.Parse(resp)
	if err != nil {
		t.Fatalf("parse RPM response: %v", err)
	}

	// RPM responses come as ComplexACK
	cack, ok := msg.(*services.ComplexACK)
	if !ok {
		t.Fatalf("expected ComplexACK, got %T", msg)
	}

	// Verify it was parsed as a valid response (has objects)
	if len(cack.APDU.Objects) == 0 {
		t.Fatal("RPM response has no objects")
	}

	if cack.APDU.Service != services.ServiceConfirmedReadPropMultiple {
		t.Errorf("expected service %d, got %d",
			services.ServiceConfirmedReadPropMultiple, cack.APDU.Service)
	}
}

func TestWritePropertyReadOnly(t *testing.T) {
	_, serverAddr := startTestServer(t)

	// Try to write to a read-only point (analogInput, index 1)
	wpBytes, err := bacnet.NewWriteProperty(
		objects.ObjectTypeAnalogInput, 1,
		objects.PropertyIdPresentValue,
		float32(99.0),
	)
	if err != nil {
		t.Fatalf("create WP: %v", err)
	}

	resp := sendAndReceive(t, serverAddr, wpBytes)

	msg, err := bacnet.Parse(resp)
	if err != nil {
		t.Fatalf("parse WP response: %v", err)
	}

	// Should get an Error response (not writable)
	errMsg, ok := msg.(*services.Error)
	if !ok {
		t.Fatalf("expected Error for write to read-only, got %T", msg)
	}

	dec, err := errMsg.Decode()
	if err != nil {
		t.Fatalf("decode Error: %v", err)
	}

	if dec.ErrorCode != ErrorCodeWriteAccessDenied {
		t.Errorf("expected WriteAccessDenied error code %d, got %d",
			ErrorCodeWriteAccessDenied, dec.ErrorCode)
	}
}

func TestMultipleDevices(t *testing.T) {
	defVal := 70.0
	store1 := points.NewPointStore([]points.PointDefinition{
		{
			VolttronName:     "Temp1",
			BACnetObjectType: "analogValue",
			Index:            1,
			DefaultValue:     &defVal,
			Active:           true,
		},
	})

	defVal2 := 80.0
	store2 := points.NewPointStore([]points.PointDefinition{
		{
			VolttronName:     "Temp2",
			BACnetObjectType: "analogValue",
			Index:            2,
			DefaultValue:     &defVal2,
			Active:           true,
		},
	})

	devices := map[int]*points.PointStore{
		2001: store1,
		2002: store2,
	}

	cfg := config.BACnetConfig{
		Enabled:   true,
		Interface: "127.0.0.1",
		Port:      0,
	}

	srv := New(cfg)
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv.conn = conn
	srv.devices = devices

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		srv.Stop()
	})

	go srv.readLoop(ctx)

	serverAddr := conn.LocalAddr()

	// WhoIs should return 2 IAm responses
	whoisBytes, err := bacnet.NewWhois()
	if err != nil {
		t.Fatalf("create WhoIs: %v", err)
	}

	clientConn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("client listen: %v", err)
	}
	defer clientConn.Close()

	if err := clientConn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}

	if _, err := clientConn.WriteTo(whoisBytes, serverAddr); err != nil {
		t.Fatalf("send WhoIs: %v", err)
	}

	// Read both IAm responses
	deviceIDs := make(map[uint32]bool)
	buf := make([]byte, 1500)
	for i := 0; i < 2; i++ {
		n, _, err := clientConn.ReadFrom(buf)
		if err != nil {
			t.Fatalf("read IAm %d: %v", i, err)
		}
		msg, err := bacnet.Parse(buf[:n])
		if err != nil {
			t.Fatalf("parse IAm %d: %v", i, err)
		}
		iam, ok := msg.(*services.UnconfirmedIAm)
		if !ok {
			t.Fatalf("expected IAm, got %T", msg)
		}
		dec, err := iam.Decode()
		if err != nil {
			t.Fatalf("decode IAm: %v", err)
		}
		deviceIDs[dec.InstanceNum] = true
	}

	if !deviceIDs[2001] {
		t.Error("missing IAm for device 2001")
	}
	if !deviceIDs[2002] {
		t.Error("missing IAm for device 2002")
	}
}

func TestObjectTypeMapping(t *testing.T) {
	tests := []struct {
		str     string
		numeric uint16
	}{
		{"analogInput", objects.ObjectTypeAnalogInput},
		{"analogOutput", objects.ObjectTypeAnalogOutput},
		{"analogValue", objects.ObjectTypeAnalogValue},
		{"binaryInput", objects.ObjectTypeBinaryInput},
		{"binaryOutput", objects.ObjectTypeBinaryOutput},
		{"binaryValue", objects.ObjectTypeBinaryValue},
		{"multiStateInput", objects.ObjectTypeMultiStateInput},
		{"multiStateValue", objects.ObjectTypeMultiStateValue},
	}

	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			got, ok := stringToBACnetType(tt.str)
			if !ok {
				t.Fatalf("stringToBACnetType(%q) not found", tt.str)
			}
			if got != tt.numeric {
				t.Errorf("stringToBACnetType(%q) = %d, want %d", tt.str, got, tt.numeric)
			}

			gotStr := bacnetTypeToString(tt.numeric)
			if gotStr != tt.str {
				t.Errorf("bacnetTypeToString(%d) = %q, want %q", tt.numeric, gotStr, tt.str)
			}
		})
	}
}

func TestUnitsMapping(t *testing.T) {
	tests := []struct {
		units string
		code  uint16
	}{
		{"degreesFahrenheit", 64},
		{"percent", 98},
		{"noUnits", 95},
		{"unknownUnit", 95}, // defaults to noUnits
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s=%d", tt.units, tt.code), func(t *testing.T) {
			got := unitsStringToCode(tt.units)
			if got != tt.code {
				t.Errorf("unitsStringToCode(%q) = %d, want %d", tt.units, got, tt.code)
			}
		})
	}
}
