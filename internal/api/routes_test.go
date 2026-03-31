package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/VOLTTRON/sim-rtu/internal/config"
	"github.com/VOLTTRON/sim-rtu/internal/engine"
)

func testEngine(t *testing.T) *engine.Engine {
	t.Helper()
	cfg := &config.AppConfig{
		Simulator: config.SimulatorConfig{TickInterval: 1.0, TimeScale: 1.0},
		Devices: []config.DeviceConfig{
			{
				Name:     "Test-Thermo",
				Type:     "thermostat",
				DeviceID: 100,
				Registry: "../../configs/schneider.csv",
				Thermal: &config.ThermalConfig{
					R: 0.02, C: 1000,
					InitialZoneTemp:       72.0,
					CoolingCapacityStage1: 18000,
					CoolingCapacityStage2: 18000,
					HeatingCapacityStage1: 20000,
					HeatingCapacityStage2: 20000,
				},
				Weather: &config.WeatherConfig{Type: "static", Temperature: 85.0},
			},
		},
	}
	eng, err := engine.New(cfg)
	if err != nil {
		t.Fatalf("engine.New() error: %v", err)
	}
	return eng
}

func setupServer(t *testing.T) *http.ServeMux {
	t.Helper()
	eng := testEngine(t)
	mux := http.NewServeMux()
	s := &Server{engine: eng}
	s.registerRoutes(mux)
	return mux
}

func setupServerWithToken(t *testing.T, token string) http.Handler {
	t.Helper()
	eng := testEngine(t)
	mux := http.NewServeMux()
	s := &Server{engine: eng, token: token}
	s.registerRoutes(mux)
	return s.authMiddleware(mux)
}

func TestHandleListDevices(t *testing.T) {
	mux := setupServer(t)

	req := httptest.NewRequest("GET", "/api/v1/devices", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
}

func TestHandleListPoints(t *testing.T) {
	mux := setupServer(t)

	req := httptest.NewRequest("GET", "/api/v1/devices/100/points", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
}

func TestHandleGetPoint(t *testing.T) {
	mux := setupServer(t)

	// Existing point
	req := httptest.NewRequest("GET", "/api/v1/devices/100/points/ZoneTemperature", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Non-existent point
	req = httptest.NewRequest("GET", "/api/v1/devices/100/points/DoesNotExist", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleGetPoint_InvalidDevice(t *testing.T) {
	mux := setupServer(t)

	req := httptest.NewRequest("GET", "/api/v1/devices/999/points/ZoneTemperature", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleWritePoint(t *testing.T) {
	mux := setupServer(t)

	body := WriteRequest{Value: 74.0, Priority: 8}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/api/v1/devices/100/points/ZoneTemperature", bytes.NewReader(b))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestHandleWritePoint_InvalidPriority(t *testing.T) {
	mux := setupServer(t)

	body := WriteRequest{Value: 74.0, Priority: 0}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/api/v1/devices/100/points/ZoneTemperature", bytes.NewReader(b))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleWritePoint_NonWritable(t *testing.T) {
	mux := setupServer(t)

	// EffectiveSystemMode is multiStateInput — should check if writable
	body := WriteRequest{Value: 1.0, Priority: 8}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/api/v1/devices/100/points/EffectiveOccupancy", bytes.NewReader(b))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should be bad request since EffectiveOccupancy is not writable in schneider
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleStatus(t *testing.T) {
	mux := setupServer(t)

	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleWeatherOverride(t *testing.T) {
	mux := setupServer(t)

	body := map[string]interface{}{"temperature": 95.0}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/weather", bytes.NewReader(b))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
