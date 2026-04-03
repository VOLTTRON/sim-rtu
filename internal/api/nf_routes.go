package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/VOLTTRON/sim-rtu/internal/engine"
)

// --- NF API request types ---

// NFRequest is the top-level Normal Framework confirmed-service request.
type NFRequest struct {
	DeviceAddress NFDeviceAddress `json:"device_address"`
	Request       NFRequestBody  `json:"request"`
}

// NFDeviceAddress identifies the target BACnet device.
type NFDeviceAddress struct {
	DeviceID int `json:"deviceId"`
}

// NFRequestBody contains either a read or write operation.
type NFRequestBody struct {
	ReadPropertyMultiple *NFReadPropertyMultiple `json:"read_property_multiple,omitempty"`
	WriteProperty        *NFWriteProperty        `json:"write_property,omitempty"`
}

// NFReadPropertyMultiple describes a multi-property read request.
type NFReadPropertyMultiple struct {
	ListOfReadAccessSpecs []NFReadAccessSpec `json:"list_of_read_access_specifications"`
}

// NFReadAccessSpec identifies one object and its requested properties.
type NFReadAccessSpec struct {
	ObjectIdentifier         NFObjectIdentifier    `json:"object_identifier"`
	ListOfPropertyReferences []NFPropertyReference `json:"list_of_property_references"`
}

// NFObjectIdentifier is a BACnet object type + instance pair.
type NFObjectIdentifier struct {
	ObjectType string `json:"object_type"`
	Instance   int    `json:"instance"`
}

// NFPropertyReference identifies a single property to read.
type NFPropertyReference struct {
	PropertyIdentifier string `json:"property_identifier"`
	PropertyArrayIndex int    `json:"property_array_index"`
}

// NFWriteProperty describes a single-property write request.
type NFWriteProperty struct {
	ObjectIdentifier   NFObjectIdentifier `json:"object_identifier"`
	PropertyIdentifier string             `json:"property_identifier"`
	PropertyArrayIndex int                `json:"property_array_index"`
	PropertyValue      map[string]any     `json:"property_value"`
	Priority           int                `json:"priority"`
}

// --- NF API response types ---

// NFResponse wraps the acknowledgement for a confirmed-service call.
type NFResponse struct {
	Ack NFAck `json:"ack"`
}

// NFAck holds the specific ack payload for the operation type.
type NFAck struct {
	ReadPropertyMultiple *NFReadPropertyMultipleAck `json:"readPropertyMultiple,omitempty"`
	WriteProperty        *map[string]any            `json:"writeProperty,omitempty"`
}

// NFReadPropertyMultipleAck contains the results for a read-multiple.
type NFReadPropertyMultipleAck struct {
	ListOfReadAccessResults []NFReadAccessResult `json:"listOfReadAccessResults"`
}

// NFReadAccessResult pairs an object identifier with its property values.
type NFReadAccessResult struct {
	ObjectIdentifier NFObjectIdentifier `json:"objectIdentifier"`
	ListOfResults    []NFPropertyResult `json:"listOfResults"`
}

// NFPropertyResult holds a single property value in the NF encoding.
type NFPropertyResult struct {
	PropertyValue map[string]any `json:"propertyValue"`
}

// NFErrorResponse is the error body returned for NF API failures.
type NFErrorResponse struct {
	Error string `json:"error"`
}

// --- Object type mapping ---

// nfObjectTypeMap converts NF object type strings to internal BACnet types.
var nfObjectTypeMap = map[string]string{
	"OBJECT_TYPE_ANALOG_INPUT":       "analogInput",
	"OBJECT_TYPE_ANALOG_OUTPUT":      "analogOutput",
	"OBJECT_TYPE_ANALOG_VALUE":       "analogValue",
	"OBJECT_TYPE_BINARY_INPUT":       "binaryInput",
	"OBJECT_TYPE_BINARY_OUTPUT":      "binaryOutput",
	"OBJECT_TYPE_BINARY_VALUE":       "binaryValue",
	"OBJECT_TYPE_MULTI_STATE_INPUT":  "multiStateInput",
	"OBJECT_TYPE_MULTI_STATE_OUTPUT": "multiStateOutput",
	"OBJECT_TYPE_MULTI_STATE_VALUE":  "multiStateValue",
}

// --- Handler ---

func (s *Server) handleNFConfirmedService(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit

	var req NFRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeNFError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	dev := s.engine.Device(req.DeviceAddress.DeviceID)
	if dev == nil {
		writeNFError(w, http.StatusNotFound, fmt.Sprintf("device %d not found", req.DeviceAddress.DeviceID))
		return
	}

	switch {
	case req.Request.ReadPropertyMultiple != nil:
		s.handleNFReadPropertyMultiple(w, dev, req.Request.ReadPropertyMultiple)
	case req.Request.WriteProperty != nil:
		s.handleNFWriteProperty(w, dev, req.Request.WriteProperty)
	default:
		writeNFError(w, http.StatusBadRequest, "request must contain read_property_multiple or write_property")
	}
}

func (s *Server) handleNFReadPropertyMultiple(w http.ResponseWriter, dev *engine.Device, rpm *NFReadPropertyMultiple) {
	results := make([]NFReadAccessResult, 0, len(rpm.ListOfReadAccessSpecs))

	for _, spec := range rpm.ListOfReadAccessSpecs {
		internalType, ok := nfObjectTypeMap[spec.ObjectIdentifier.ObjectType]
		if !ok {
			writeNFError(w, http.StatusBadRequest, fmt.Sprintf("unknown object type: %s", spec.ObjectIdentifier.ObjectType))
			return
		}

		val, err := dev.Store.ReadByKey(internalType, spec.ObjectIdentifier.Instance)
		if err != nil {
			writeNFError(w, http.StatusNotFound, err.Error())
			return
		}

		var realVal float64
		if val != nil {
			realVal = *val
		}

		propResults := make([]NFPropertyResult, 0, len(spec.ListOfPropertyReferences))
		for range spec.ListOfPropertyReferences {
			propResults = append(propResults, NFPropertyResult{
				PropertyValue: encodeNFValue(internalType, realVal),
			})
		}

		results = append(results, NFReadAccessResult{
			ObjectIdentifier: spec.ObjectIdentifier,
			ListOfResults:    propResults,
		})
	}

	resp := NFResponse{
		Ack: NFAck{
			ReadPropertyMultiple: &NFReadPropertyMultipleAck{
				ListOfReadAccessResults: results,
			},
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleNFWriteProperty(w http.ResponseWriter, dev *engine.Device, wp *NFWriteProperty) {
	internalType, ok := nfObjectTypeMap[wp.ObjectIdentifier.ObjectType]
	if !ok {
		writeNFError(w, http.StatusBadRequest, fmt.Sprintf("unknown object type: %s", wp.ObjectIdentifier.ObjectType))
		return
	}

	value, err := decodeNFWriteValue(wp.PropertyValue)
	if err != nil {
		writeNFError(w, http.StatusBadRequest, fmt.Sprintf("invalid property value: %v", err))
		return
	}

	priority := wp.Priority
	if priority < 1 || priority > 16 {
		priority = 16
	}

	if err := dev.Store.WriteByKey(internalType, wp.ObjectIdentifier.Instance, value, priority); err != nil {
		writeNFError(w, http.StatusBadRequest, err.Error())
		return
	}

	empty := map[string]any{}
	resp := NFResponse{
		Ack: NFAck{
			WriteProperty: &empty,
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Value encoding/decoding ---

func encodeNFValue(objectType string, value float64) map[string]any {
	switch {
	case isAnalogType(objectType):
		return map[string]any{"real": value}
	case isBinaryType(objectType):
		if value >= 1 {
			return map[string]any{"binarypv": "BINARYPV_ACTIVE"}
		}
		return map[string]any{"binarypv": "BINARYPV_INACTIVE"}
	case isMultiStateType(objectType):
		return map[string]any{"unsigned": int(value)}
	default:
		return map[string]any{"real": value}
	}
}

func decodeNFWriteValue(propertyValue map[string]any) (float64, error) {
	if v, ok := propertyValue["real"]; ok {
		return toFloat64(v)
	}
	if v, ok := propertyValue["enumerated"]; ok {
		return toFloat64(v)
	}
	if v, ok := propertyValue["unsigned"]; ok {
		return toFloat64(v)
	}
	if _, ok := propertyValue["null"]; ok {
		return 0, nil // null write = relinquish
	}
	return 0, fmt.Errorf("unknown property value format: %v", propertyValue)
}

func toFloat64(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case json.Number:
		return n.Float64()
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

func isAnalogType(objectType string) bool {
	return strings.HasPrefix(objectType, "analog")
}

func isBinaryType(objectType string) bool {
	return strings.HasPrefix(objectType, "binary")
}

func isMultiStateType(objectType string) bool {
	return strings.HasPrefix(objectType, "multiState")
}

func writeNFError(w http.ResponseWriter, status int, msg string) {
	slog.Warn("NF API error", "status", status, "error", msg)
	writeJSON(w, status, NFErrorResponse{Error: msg})
}
