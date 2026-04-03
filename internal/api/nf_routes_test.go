package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const nfEndpoint = "/api/v2/bacnet/confirmed-service"

func nfReadRequest(deviceID int, specs []NFReadAccessSpec) NFRequest {
	return NFRequest{
		DeviceAddress: NFDeviceAddress{DeviceID: deviceID},
		Request: NFRequestBody{
			ReadPropertyMultiple: &NFReadPropertyMultiple{
				ListOfReadAccessSpecs: specs,
			},
		},
	}
}

func nfWriteRequest(deviceID int, wp NFWriteProperty) NFRequest {
	return NFRequest{
		DeviceAddress: NFDeviceAddress{DeviceID: deviceID},
		Request: NFRequestBody{
			WriteProperty: &wp,
		},
	}
}

func doNFRequest(t *testing.T, mux *http.ServeMux, req NFRequest) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	r := httptest.NewRequest("POST", nfEndpoint, bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func TestNFReadPropertyMultiple_AnalogValue(t *testing.T) {
	mux := setupServer(t)

	// ZoneTemperature is analogValue index 100, default ~68.7
	req := nfReadRequest(100, []NFReadAccessSpec{
		{
			ObjectIdentifier: NFObjectIdentifier{
				ObjectType: "OBJECT_TYPE_ANALOG_VALUE",
				Instance:   100,
			},
			ListOfPropertyReferences: []NFPropertyReference{
				{
					PropertyIdentifier: "PROPERTY_IDENTIFIER_PRESENT_VALUE",
					PropertyArrayIndex: 4294967295,
				},
			},
		},
	})

	w := doNFRequest(t, mux, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp NFResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Ack.ReadPropertyMultiple == nil {
		t.Fatal("expected readPropertyMultiple in ack")
	}

	results := resp.Ack.ReadPropertyMultiple.ListOfReadAccessResults
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].ObjectIdentifier.ObjectType != "OBJECT_TYPE_ANALOG_VALUE" {
		t.Errorf("object type = %q, want OBJECT_TYPE_ANALOG_VALUE", results[0].ObjectIdentifier.ObjectType)
	}
	if results[0].ObjectIdentifier.Instance != 100 {
		t.Errorf("instance = %d, want 100", results[0].ObjectIdentifier.Instance)
	}
	if len(results[0].ListOfResults) != 1 {
		t.Fatalf("got %d property results, want 1", len(results[0].ListOfResults))
	}

	pv := results[0].ListOfResults[0].PropertyValue
	if _, ok := pv["real"]; !ok {
		t.Errorf("expected 'real' key in property value, got %v", pv)
	}
}

func TestNFReadPropertyMultiple_BinaryOutput(t *testing.T) {
	mux := setupServer(t)

	// SupplyFanStatus is binaryOutput index 25, default 1 (ACTIVE)
	req := nfReadRequest(100, []NFReadAccessSpec{
		{
			ObjectIdentifier: NFObjectIdentifier{
				ObjectType: "OBJECT_TYPE_BINARY_OUTPUT",
				Instance:   25,
			},
			ListOfPropertyReferences: []NFPropertyReference{
				{
					PropertyIdentifier: "PROPERTY_IDENTIFIER_PRESENT_VALUE",
					PropertyArrayIndex: 4294967295,
				},
			},
		},
	})

	w := doNFRequest(t, mux, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp NFResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	results := resp.Ack.ReadPropertyMultiple.ListOfReadAccessResults
	pv := results[0].ListOfResults[0].PropertyValue
	bpv, ok := pv["binarypv"]
	if !ok {
		t.Fatalf("expected 'binarypv' key in property value, got %v", pv)
	}
	if bpv != "BINARYPV_ACTIVE" {
		t.Errorf("binarypv = %v, want BINARYPV_ACTIVE", bpv)
	}
}

func TestNFReadPropertyMultiple_MultiStateValue(t *testing.T) {
	mux := setupServer(t)

	// OccupancyCommand is multiStateValue index 10, default 2
	req := nfReadRequest(100, []NFReadAccessSpec{
		{
			ObjectIdentifier: NFObjectIdentifier{
				ObjectType: "OBJECT_TYPE_MULTI_STATE_VALUE",
				Instance:   10,
			},
			ListOfPropertyReferences: []NFPropertyReference{
				{
					PropertyIdentifier: "PROPERTY_IDENTIFIER_PRESENT_VALUE",
					PropertyArrayIndex: 4294967295,
				},
			},
		},
	})

	w := doNFRequest(t, mux, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp NFResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	results := resp.Ack.ReadPropertyMultiple.ListOfReadAccessResults
	pv := results[0].ListOfResults[0].PropertyValue
	if _, ok := pv["unsigned"]; !ok {
		t.Errorf("expected 'unsigned' key in property value, got %v", pv)
	}
}

func TestNFReadPropertyMultiple_MultipleObjects(t *testing.T) {
	mux := setupServer(t)

	req := nfReadRequest(100, []NFReadAccessSpec{
		{
			ObjectIdentifier: NFObjectIdentifier{
				ObjectType: "OBJECT_TYPE_ANALOG_VALUE",
				Instance:   100, // ZoneTemperature
			},
			ListOfPropertyReferences: []NFPropertyReference{
				{PropertyIdentifier: "PROPERTY_IDENTIFIER_PRESENT_VALUE", PropertyArrayIndex: 4294967295},
			},
		},
		{
			ObjectIdentifier: NFObjectIdentifier{
				ObjectType: "OBJECT_TYPE_ANALOG_VALUE",
				Instance:   101, // OutdoorAirTemperature
			},
			ListOfPropertyReferences: []NFPropertyReference{
				{PropertyIdentifier: "PROPERTY_IDENTIFIER_PRESENT_VALUE", PropertyArrayIndex: 4294967295},
			},
		},
	})

	w := doNFRequest(t, mux, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp NFResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	results := resp.Ack.ReadPropertyMultiple.ListOfReadAccessResults
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
}

func TestNFWriteProperty_AnalogValue(t *testing.T) {
	mux := setupServer(t)

	// Write to ZoneTemperature (analogValue index 100)
	req := nfWriteRequest(100, NFWriteProperty{
		ObjectIdentifier: NFObjectIdentifier{
			ObjectType: "OBJECT_TYPE_ANALOG_VALUE",
			Instance:   100,
		},
		PropertyIdentifier: "PROPERTY_IDENTIFIER_PRESENT_VALUE",
		PropertyArrayIndex: 4294967295,
		PropertyValue: map[string]any{
			"@type": "type.googleapis.com/normalgw.bacnet.v2.ApplicationDataValue",
			"real":  72.5,
		},
		Priority: 16,
	})

	w := doNFRequest(t, mux, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp NFResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Ack.WriteProperty == nil {
		t.Fatal("expected writeProperty in ack")
	}

	// Verify the value was written by reading it back
	readReq := nfReadRequest(100, []NFReadAccessSpec{
		{
			ObjectIdentifier: NFObjectIdentifier{
				ObjectType: "OBJECT_TYPE_ANALOG_VALUE",
				Instance:   100,
			},
			ListOfPropertyReferences: []NFPropertyReference{
				{PropertyIdentifier: "PROPERTY_IDENTIFIER_PRESENT_VALUE", PropertyArrayIndex: 4294967295},
			},
		},
	})

	w2 := doNFRequest(t, mux, readReq)
	var readResp NFResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &readResp); err != nil {
		t.Fatalf("unmarshal read: %v", err)
	}

	pv := readResp.Ack.ReadPropertyMultiple.ListOfReadAccessResults[0].ListOfResults[0].PropertyValue
	got, ok := pv["real"].(float64)
	if !ok {
		t.Fatalf("expected real float64, got %v (%T)", pv["real"], pv["real"])
	}
	if got != 72.5 {
		t.Errorf("read back value = %f, want 72.5", got)
	}
}

func TestNFWriteProperty_BinaryValue(t *testing.T) {
	mux := setupServer(t)

	// Write to SupplyFanStatus (binaryOutput index 25)
	req := nfWriteRequest(100, NFWriteProperty{
		ObjectIdentifier: NFObjectIdentifier{
			ObjectType: "OBJECT_TYPE_BINARY_OUTPUT",
			Instance:   25,
		},
		PropertyIdentifier: "PROPERTY_IDENTIFIER_PRESENT_VALUE",
		PropertyArrayIndex: 4294967295,
		PropertyValue: map[string]any{
			"@type":      "type.googleapis.com/normalgw.bacnet.v2.ApplicationDataValue",
			"enumerated": 0,
		},
		Priority: 16,
	})

	w := doNFRequest(t, mux, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Read back and verify it's INACTIVE
	readReq := nfReadRequest(100, []NFReadAccessSpec{
		{
			ObjectIdentifier: NFObjectIdentifier{
				ObjectType: "OBJECT_TYPE_BINARY_OUTPUT",
				Instance:   25,
			},
			ListOfPropertyReferences: []NFPropertyReference{
				{PropertyIdentifier: "PROPERTY_IDENTIFIER_PRESENT_VALUE", PropertyArrayIndex: 4294967295},
			},
		},
	})

	w2 := doNFRequest(t, mux, readReq)
	var resp NFResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	pv := resp.Ack.ReadPropertyMultiple.ListOfReadAccessResults[0].ListOfResults[0].PropertyValue
	if pv["binarypv"] != "BINARYPV_INACTIVE" {
		t.Errorf("binarypv = %v, want BINARYPV_INACTIVE", pv["binarypv"])
	}
}

func TestNFWriteProperty_MultiStateValue(t *testing.T) {
	mux := setupServer(t)

	// Write to OccupancyCommand (multiStateValue index 10)
	req := nfWriteRequest(100, NFWriteProperty{
		ObjectIdentifier: NFObjectIdentifier{
			ObjectType: "OBJECT_TYPE_MULTI_STATE_VALUE",
			Instance:   10,
		},
		PropertyIdentifier: "PROPERTY_IDENTIFIER_PRESENT_VALUE",
		PropertyArrayIndex: 4294967295,
		PropertyValue: map[string]any{
			"@type":    "type.googleapis.com/normalgw.bacnet.v2.ApplicationDataValue",
			"unsigned": 1,
		},
		Priority: 16,
	})

	w := doNFRequest(t, mux, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Read back and verify
	readReq := nfReadRequest(100, []NFReadAccessSpec{
		{
			ObjectIdentifier: NFObjectIdentifier{
				ObjectType: "OBJECT_TYPE_MULTI_STATE_VALUE",
				Instance:   10,
			},
			ListOfPropertyReferences: []NFPropertyReference{
				{PropertyIdentifier: "PROPERTY_IDENTIFIER_PRESENT_VALUE", PropertyArrayIndex: 4294967295},
			},
		},
	})

	w2 := doNFRequest(t, mux, readReq)
	var resp NFResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	pv := resp.Ack.ReadPropertyMultiple.ListOfReadAccessResults[0].ListOfResults[0].PropertyValue
	// JSON numbers decode as float64
	got, ok := pv["unsigned"].(float64)
	if !ok {
		t.Fatalf("expected unsigned number, got %v (%T)", pv["unsigned"], pv["unsigned"])
	}
	if int(got) != 1 {
		t.Errorf("unsigned = %v, want 1", got)
	}
}

func TestNFConfirmedService_InvalidDeviceID(t *testing.T) {
	mux := setupServer(t)

	req := nfReadRequest(99999, []NFReadAccessSpec{
		{
			ObjectIdentifier: NFObjectIdentifier{
				ObjectType: "OBJECT_TYPE_ANALOG_VALUE",
				Instance:   100,
			},
			ListOfPropertyReferences: []NFPropertyReference{
				{PropertyIdentifier: "PROPERTY_IDENTIFIER_PRESENT_VALUE", PropertyArrayIndex: 4294967295},
			},
		},
	})

	w := doNFRequest(t, mux, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestNFConfirmedService_UnknownObjectType(t *testing.T) {
	mux := setupServer(t)

	req := nfReadRequest(100, []NFReadAccessSpec{
		{
			ObjectIdentifier: NFObjectIdentifier{
				ObjectType: "OBJECT_TYPE_BOGUS",
				Instance:   1,
			},
			ListOfPropertyReferences: []NFPropertyReference{
				{PropertyIdentifier: "PROPERTY_IDENTIFIER_PRESENT_VALUE", PropertyArrayIndex: 4294967295},
			},
		},
	})

	w := doNFRequest(t, mux, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestNFConfirmedService_UnknownInstance(t *testing.T) {
	mux := setupServer(t)

	req := nfReadRequest(100, []NFReadAccessSpec{
		{
			ObjectIdentifier: NFObjectIdentifier{
				ObjectType: "OBJECT_TYPE_ANALOG_VALUE",
				Instance:   99999,
			},
			ListOfPropertyReferences: []NFPropertyReference{
				{PropertyIdentifier: "PROPERTY_IDENTIFIER_PRESENT_VALUE", PropertyArrayIndex: 4294967295},
			},
		},
	})

	w := doNFRequest(t, mux, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestNFConfirmedService_EmptyRequest(t *testing.T) {
	mux := setupServer(t)

	req := NFRequest{
		DeviceAddress: NFDeviceAddress{DeviceID: 100},
		Request:       NFRequestBody{},
	}

	w := doNFRequest(t, mux, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestNFConfirmedService_InvalidJSON(t *testing.T) {
	mux := setupServer(t)

	r := httptest.NewRequest("POST", nfEndpoint, bytes.NewReader([]byte("not json")))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestNFWriteProperty_NonWritablePoint(t *testing.T) {
	mux := setupServer(t)

	// EffectiveOccupancy (analogValue index 26) is not writable
	req := nfWriteRequest(100, NFWriteProperty{
		ObjectIdentifier: NFObjectIdentifier{
			ObjectType: "OBJECT_TYPE_ANALOG_VALUE",
			Instance:   26,
		},
		PropertyIdentifier: "PROPERTY_IDENTIFIER_PRESENT_VALUE",
		PropertyArrayIndex: 4294967295,
		PropertyValue: map[string]any{
			"real": 1.0,
		},
		Priority: 16,
	})

	w := doNFRequest(t, mux, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestNFWriteProperty_UnknownValueFormat(t *testing.T) {
	mux := setupServer(t)

	req := nfWriteRequest(100, NFWriteProperty{
		ObjectIdentifier: NFObjectIdentifier{
			ObjectType: "OBJECT_TYPE_ANALOG_VALUE",
			Instance:   100,
		},
		PropertyIdentifier: "PROPERTY_IDENTIFIER_PRESENT_VALUE",
		PropertyArrayIndex: 4294967295,
		PropertyValue: map[string]any{
			"bogus_field": "hello",
		},
		Priority: 16,
	})

	w := doNFRequest(t, mux, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}
