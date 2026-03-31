package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strconv"

	"github.com/VOLTTRON/sim-rtu/internal/engine"
)

// Response is the standard API response wrapper.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// PointValue represents a single point reading in API responses.
type PointValue struct {
	Name  string   `json:"name"`
	Value *float64 `json:"value"`
	Units string   `json:"units"`
}

// WriteRequest is the body for writing a point value.
type WriteRequest struct {
	Value    float64 `json:"value"`
	Priority int     `json:"priority"`
}

// DeviceInfo describes a device in API responses.
type DeviceInfo struct {
	DeviceID   int    `json:"device_id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	PointCount int    `json:"point_count"`
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/devices", s.handleListDevices)
	mux.HandleFunc("GET /api/v1/devices/{deviceID}/points", s.handleListPoints)
	mux.HandleFunc("GET /api/v1/devices/{deviceID}/points/{name}", s.handleGetPoint)
	mux.HandleFunc("PUT /api/v1/devices/{deviceID}/points/{name}", s.handleWritePoint)
	mux.HandleFunc("GET /api/v1/status", s.handleStatus)
	mux.HandleFunc("POST /api/v1/weather", s.handleWeatherOverride)
}

func (s *Server) handleListDevices(w http.ResponseWriter, _ *http.Request) {
	devices := s.engine.Devices()

	// Collect and sort IDs for deterministic ordering.
	ids := make([]int, 0, len(devices))
	for id := range devices {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	infos := make([]DeviceInfo, 0, len(devices))
	for _, id := range ids {
		dev := devices[id]
		infos = append(infos, DeviceInfo{
			DeviceID:   id,
			Name:       dev.Config.Name,
			Type:       dev.Config.Type,
			PointCount: len(dev.Store.Definitions()),
		})
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: infos})
}

func (s *Server) handleListPoints(w http.ResponseWriter, r *http.Request) {
	dev, ok := s.getDevice(w, r)
	if !ok {
		return
	}

	all := dev.Store.ReadAll()
	defs := dev.Store.Definitions()

	unitMap := make(map[string]string, len(defs))
	for _, d := range defs {
		unitMap[d.VolttronName] = d.Units
	}

	pvs := make([]PointValue, 0, len(defs))
	for _, d := range defs {
		pv := PointValue{Name: d.VolttronName, Units: d.Units}
		if v, ok := all[d.VolttronName]; ok {
			pv.Value = v
		}
		pvs = append(pvs, pv)
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: pvs})
}

func (s *Server) handleGetPoint(w http.ResponseWriter, r *http.Request) {
	dev, ok := s.getDevice(w, r)
	if !ok {
		return
	}

	name := r.PathValue("name")
	v, err := dev.Store.Read(name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, Response{Success: false, Error: err.Error()})
		return
	}

	// Look up units from definitions so the response is consistent with
	// handleListPoints.
	var units string
	for _, d := range dev.Store.Definitions() {
		if d.VolttronName == name {
			units = d.Units
			break
		}
	}

	pv := PointValue{Name: name, Value: v, Units: units}
	writeJSON(w, http.StatusOK, Response{Success: true, Data: pv})
}

func (s *Server) handleWritePoint(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit

	dev, ok := s.getDevice(w, r)
	if !ok {
		return
	}

	name := r.PathValue("name")

	var req WriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "invalid request body"})
		return
	}

	if req.Priority < 1 || req.Priority > 16 {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "priority must be 1-16"})
		return
	}

	if err := dev.Store.Write(name, req.Value, req.Priority); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true})
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	status := map[string]interface{}{
		"elapsed_hours": s.engine.Elapsed(),
		"device_count":  len(s.engine.Devices()),
	}
	writeJSON(w, http.StatusOK, Response{Success: true, Data: status})
}

func (s *Server) handleWeatherOverride(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit

	var req struct {
		Temperature float64 `json:"temperature"`
		DeviceID    *int    `json:"device_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "invalid request body"})
		return
	}

	// Override all thermostat devices or a specific one
	for id, dev := range s.engine.Devices() {
		if req.DeviceID != nil && id != *req.DeviceID {
			continue
		}
		if dev.Config.Type == "thermostat" {
			_ = dev.Store.SetInternal("OutdoorAirTemperature", req.Temperature)
		}
	}

	writeJSON(w, http.StatusOK, Response{Success: true})
}

func (s *Server) getDevice(w http.ResponseWriter, r *http.Request) (*engine.Device, bool) {
	idStr := r.PathValue("deviceID")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "invalid device ID"})
		return nil, false
	}

	dev := s.engine.Device(id)
	if dev == nil {
		writeJSON(w, http.StatusNotFound, Response{Success: false, Error: "device not found"})
		return nil, false
	}

	return dev, true
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to write JSON response", "error", err)
	}
}
