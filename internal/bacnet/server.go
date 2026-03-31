package bacnet

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"

	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"

	"github.com/VOLTTRON/sim-rtu/internal/config"
	"github.com/VOLTTRON/sim-rtu/internal/points"
)

// Server is a BACnet/IP server that exposes simulated device points
// over the BACnet protocol using UDP.
type Server struct {
	config  config.BACnetConfig
	devices map[int]*points.PointStore
	conn    net.PacketConn
	mu      sync.Mutex
	done    chan struct{}
}

// New creates a new BACnet server.
func New(cfg config.BACnetConfig) *Server {
	return &Server{
		config:  cfg,
		devices: make(map[int]*points.PointStore),
		done:    make(chan struct{}),
	}
}

// Start begins listening for BACnet/IP traffic on UDP.
func (s *Server) Start(ctx context.Context, devices map[int]*points.PointStore) error {
	s.mu.Lock()
	s.devices = devices
	s.mu.Unlock()

	addr := fmt.Sprintf("%s:%d", s.config.Interface, s.config.Port)
	conn, err := net.ListenPacket("udp4", addr)
	if err != nil {
		return fmt.Errorf("listen udp %s: %w", addr, err)
	}
	s.conn = conn

	slog.Info("BACnet/IP server started",
		"address", addr,
		"device_count", len(devices),
	)

	go s.readLoop(ctx)
	return nil
}

// Stop shuts down the BACnet server.
func (s *Server) Stop() error {
	close(s.done)
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

func (s *Server) readLoop(ctx context.Context) {
	buf := make([]byte, 1500)
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		default:
		}

		n, remoteAddr, err := s.conn.ReadFrom(buf)
		if err != nil {
			select {
			case <-s.done:
				return
			case <-ctx.Done():
				return
			default:
				slog.Error("BACnet read error", "error", err)
				continue
			}
		}

		if n < 8 {
			continue
		}

		go s.handlePacket(buf[:n], remoteAddr)
	}
}

func (s *Server) handlePacket(data []byte, remoteAddr net.Addr) {
	msg, err := bacnet.Parse(data)
	if err != nil {
		slog.Debug("BACnet parse error", "error", err, "remote", remoteAddr)
		return
	}

	svc := msg.GetService()
	switch svc {
	case services.ServiceUnconfirmedWhoIs:
		s.handleWhoIs(remoteAddr)
	case services.ServiceConfirmedReadProperty:
		rp, ok := msg.(*services.ConfirmedReadProperty)
		if !ok {
			return
		}
		// Check APDU type to distinguish RP from RPM
		if rp.APDU.Service == services.ServiceConfirmedReadPropMultiple {
			s.handleReadPropertyMultiple(rp, remoteAddr)
		} else {
			s.handleReadProperty(rp, remoteAddr)
		}
	case services.ServiceConfirmedReadPropMultiple:
		rp, ok := msg.(*services.ConfirmedReadProperty)
		if !ok {
			return
		}
		s.handleReadPropertyMultiple(rp, remoteAddr)
	case services.ServiceConfirmedWriteProperty:
		wp, ok := msg.(*services.ConfirmedWriteProperty)
		if !ok {
			return
		}
		s.handleWriteProperty(wp, remoteAddr)
	default:
		slog.Debug("BACnet unhandled service", "service", svc, "remote", remoteAddr)
	}
}

// handleWhoIs responds with IAm for each device.
func (s *Server) handleWhoIs(remoteAddr net.Addr) {
	s.mu.Lock()
	devIDs := make([]int, 0, len(s.devices))
	for id := range s.devices {
		devIDs = append(devIDs, id)
	}
	s.mu.Unlock()

	for _, devID := range devIDs {
		iamBytes, err := bacnet.NewIAm(uint32(devID), VendorIDSimulator)
		if err != nil {
			slog.Error("BACnet IAm encode error", "device_id", devID, "error", err)
			continue
		}
		if _, err := s.conn.WriteTo(iamBytes, remoteAddr); err != nil {
			slog.Error("BACnet IAm send error", "device_id", devID, "error", err)
		}
		slog.Debug("BACnet IAm sent", "device_id", devID, "remote", remoteAddr)
	}
}

// handleReadProperty responds to a ReadProperty request.
func (s *Server) handleReadProperty(rp *services.ConfirmedReadProperty, remoteAddr net.Addr) {
	dec, err := decodeReadProperty(rp)
	if err != nil {
		slog.Debug("BACnet RP decode error", "error", err)
		s.sendError(services.ServiceConfirmedReadProperty, rp.APDU.InvokeID,
			objects.ErrorClassServices, objects.ErrorCodeServiceRequestDenied, remoteAddr)
		return
	}

	slog.Debug("BACnet ReadProperty",
		"object_type", dec.ObjectType,
		"instance", dec.InstanceNum,
		"property", dec.PropertyID,
	)

	// Handle device object
	if dec.ObjectType == objects.ObjectTypeDevice {
		s.handleDevicePropertyRead(dec, rp.APDU.InvokeID, remoteAddr)
		return
	}

	// Look up the point
	objTypeStr := bacnetTypeToString(dec.ObjectType)
	if objTypeStr == "" {
		s.sendError(services.ServiceConfirmedReadProperty, rp.APDU.InvokeID,
			objects.ErrorClassObject, objects.ErrorCodeUnknownObject, remoteAddr)
		return
	}

	store := s.findStoreForObject(objTypeStr, int(dec.InstanceNum))
	if store == nil {
		s.sendError(services.ServiceConfirmedReadProperty, rp.APDU.InvokeID,
			objects.ErrorClassObject, objects.ErrorCodeUnknownObject, remoteAddr)
		return
	}

	response, err := s.buildReadPropertyResponse(
		dec.ObjectType, dec.InstanceNum, dec.PropertyID,
		objTypeStr, store, rp.APDU.InvokeID,
	)
	if err != nil {
		slog.Debug("BACnet RP response build error", "error", err)
		s.sendError(services.ServiceConfirmedReadProperty, rp.APDU.InvokeID,
			objects.ErrorClassProperty, ErrorCodeUnknownProperty, remoteAddr)
		return
	}

	if _, err := s.conn.WriteTo(response, remoteAddr); err != nil {
		slog.Error("BACnet RP send error", "error", err)
	}
}

// handleWriteProperty handles a WriteProperty request.
func (s *Server) handleWriteProperty(wp *services.ConfirmedWriteProperty, remoteAddr net.Addr) {
	dec, err := decodeWriteProperty(wp)
	if err != nil {
		slog.Debug("BACnet WP decode error", "error", err)
		s.sendError(services.ServiceConfirmedWriteProperty, wp.APDU.InvokeID,
			objects.ErrorClassServices, objects.ErrorCodeServiceRequestDenied, remoteAddr)
		return
	}

	slog.Debug("BACnet WriteProperty",
		"object_type", dec.ObjectType,
		"instance", dec.InstanceNum,
		"property", dec.PropertyID,
		"priority", dec.Priority,
	)

	if dec.PropertyID != objects.PropertyIdPresentValue {
		s.sendError(services.ServiceConfirmedWriteProperty, wp.APDU.InvokeID,
			objects.ErrorClassProperty, ErrorCodeWriteAccessDenied, remoteAddr)
		return
	}

	objTypeStr := bacnetTypeToString(dec.ObjectType)
	if objTypeStr == "" {
		s.sendError(services.ServiceConfirmedWriteProperty, wp.APDU.InvokeID,
			objects.ErrorClassObject, objects.ErrorCodeUnknownObject, remoteAddr)
		return
	}

	store := s.findStoreForObject(objTypeStr, int(dec.InstanceNum))
	if store == nil {
		s.sendError(services.ServiceConfirmedWriteProperty, wp.APDU.InvokeID,
			objects.ErrorClassObject, objects.ErrorCodeUnknownObject, remoteAddr)
		return
	}

	if err := store.WriteByKey(objTypeStr, int(dec.InstanceNum), dec.Value, dec.Priority); err != nil {
		slog.Debug("BACnet WP store write error", "error", err)
		s.sendError(services.ServiceConfirmedWriteProperty, wp.APDU.InvokeID,
			objects.ErrorClassProperty, ErrorCodeWriteAccessDenied, remoteAddr)
		return
	}

	// Send SimpleACK
	sack, err := s.buildSimpleACK(services.ServiceConfirmedWriteProperty, wp.APDU.InvokeID)
	if err != nil {
		slog.Error("BACnet SACK encode error", "error", err)
		return
	}
	if _, err := s.conn.WriteTo(sack, remoteAddr); err != nil {
		slog.Error("BACnet SACK send error", "error", err)
	}
}

// handleReadPropertyMultiple handles a ReadPropertyMultiple request.
func (s *Server) handleReadPropertyMultiple(rp *services.ConfirmedReadProperty, remoteAddr net.Addr) {
	dec, err := rp.DecodeRPM()
	if err != nil {
		slog.Debug("BACnet RPM decode error", "error", err)
		s.sendError(services.ServiceConfirmedReadPropMultiple, rp.APDU.InvokeID,
			objects.ErrorClassServices, objects.ErrorCodeServiceRequestDenied, remoteAddr)
		return
	}

	slog.Debug("BACnet ReadPropertyMultiple",
		"object_type", dec.ObjectType,
		"instance", dec.InstanceNum,
		"properties", len(dec.Tags),
	)

	objTypeStr := bacnetTypeToString(dec.ObjectType)

	// Collect property IDs from the decoded tags
	propIDs := make([]uint16, 0, len(dec.Tags))
	for _, tag := range dec.Tags {
		if tag.TagClass && tag.TagNumber == 0 {
			if v, ok := tag.Value.(uint32); ok {
				propIDs = append(propIDs, uint16(v))
			}
		}
	}

	// If it's a device object, handle device properties
	if dec.ObjectType == objects.ObjectTypeDevice {
		response, err := s.buildRPMDeviceResponse(dec, propIDs, rp.APDU.InvokeID)
		if err != nil {
			slog.Debug("BACnet RPM device response error", "error", err)
			s.sendError(services.ServiceConfirmedReadPropMultiple, rp.APDU.InvokeID,
				objects.ErrorClassObject, objects.ErrorCodeUnknownObject, remoteAddr)
			return
		}
		if _, err := s.conn.WriteTo(response, remoteAddr); err != nil {
			slog.Error("BACnet RPM send error", "error", err)
		}
		return
	}

	if objTypeStr == "" {
		s.sendError(services.ServiceConfirmedReadPropMultiple, rp.APDU.InvokeID,
			objects.ErrorClassObject, objects.ErrorCodeUnknownObject, remoteAddr)
		return
	}

	store := s.findStoreForObject(objTypeStr, int(dec.InstanceNum))
	if store == nil {
		s.sendError(services.ServiceConfirmedReadPropMultiple, rp.APDU.InvokeID,
			objects.ErrorClassObject, objects.ErrorCodeUnknownObject, remoteAddr)
		return
	}

	response, err := s.buildRPMResponse(dec, propIDs, objTypeStr, store, rp.APDU.InvokeID)
	if err != nil {
		slog.Debug("BACnet RPM response build error", "error", err)
		s.sendError(services.ServiceConfirmedReadPropMultiple, rp.APDU.InvokeID,
			objects.ErrorClassObject, objects.ErrorCodeUnknownObject, remoteAddr)
		return
	}

	if _, err := s.conn.WriteTo(response, remoteAddr); err != nil {
		slog.Error("BACnet RPM send error", "error", err)
	}
}

// handleDevicePropertyRead handles ReadProperty for the Device object.
func (s *Server) handleDevicePropertyRead(dec ReadPropertyRequest, invokeID uint8, remoteAddr net.Addr) {
	deviceID := int(dec.InstanceNum)

	s.mu.Lock()
	_, exists := s.devices[deviceID]
	s.mu.Unlock()

	if !exists {
		s.sendError(services.ServiceConfirmedReadProperty, invokeID,
			objects.ErrorClassObject, objects.ErrorCodeUnknownObject, remoteAddr)
		return
	}

	var value interface{}
	switch dec.PropertyID {
	case objects.PropertyIdObjectIdentifier:
		// Return as object identifier - handled by CACK encoding
		value = float32(deviceID)
	case objects.PropertyIdObjectName:
		value = fmt.Sprintf("sim-device-%d", deviceID)
	case objects.PropertyIdObjectType:
		value = uint8(objects.ObjectTypeDevice)
	case objects.PropertyIdDescription:
		value = fmt.Sprintf("Simulated BACnet device %d", deviceID)
	default:
		s.sendError(services.ServiceConfirmedReadProperty, invokeID,
			objects.ErrorClassProperty, ErrorCodeUnknownProperty, remoteAddr)
		return
	}

	response, err := s.buildCACK(
		services.ServiceConfirmedReadProperty, invokeID,
		objects.ObjectTypeDevice, uint32(deviceID),
		dec.PropertyID, value,
	)
	if err != nil {
		slog.Error("BACnet device RP encode error", "error", err)
		return
	}
	if _, err := s.conn.WriteTo(response, remoteAddr); err != nil {
		slog.Error("BACnet device RP send error", "error", err)
	}
}

// findStoreForObject searches across all devices for a matching object.
func (s *Server) findStoreForObject(objectType string, index int) *points.PointStore {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, store := range s.devices {
		if _, ok := store.DefinitionByKey(objectType, index); ok {
			return store
		}
	}
	return nil
}

// buildReadPropertyResponse constructs a ComplexACK for ReadProperty.
func (s *Server) buildReadPropertyResponse(
	objType uint16, instance uint32, propID uint16,
	objTypeStr string, store *points.PointStore, invokeID uint8,
) ([]byte, error) {
	def, ok := store.DefinitionByKey(objTypeStr, int(instance))
	if !ok {
		return nil, fmt.Errorf("object %s[%d] not found", objTypeStr, instance)
	}

	switch propID {
	case objects.PropertyIdPresentValue:
		val, err := store.ReadByKey(objTypeStr, int(instance))
		if err != nil {
			return nil, err
		}

		var presentValue interface{}
		if val == nil {
			presentValue = float32(0)
		} else {
			presentValue = encodeValueForType(objTypeStr, *val)
		}

		return s.buildCACK(
			services.ServiceConfirmedReadProperty, invokeID,
			objType, instance, propID, presentValue,
		)

	case objects.PropertyIdObjectIdentifier:
		return s.buildCACK(
			services.ServiceConfirmedReadProperty, invokeID,
			objType, instance, propID, float32(instance),
		)

	case objects.PropertyIdObjectName:
		return s.buildCACK(
			services.ServiceConfirmedReadProperty, invokeID,
			objType, instance, propID, def.VolttronName,
		)

	case objects.PropertyIdObjectType:
		return s.buildCACK(
			services.ServiceConfirmedReadProperty, invokeID,
			objType, instance, propID, uint8(objType),
		)

	case objects.PropertyIdDescription:
		desc := def.ReferenceName
		if desc == "" {
			desc = def.VolttronName
		}
		return s.buildCACK(
			services.ServiceConfirmedReadProperty, invokeID,
			objType, instance, propID, desc,
		)

	case objects.PropertyIdUnits:
		unitCode := unitsStringToCode(def.Units)
		return s.buildCACK(
			services.ServiceConfirmedReadProperty, invokeID,
			objType, instance, propID, uint8(unitCode),
		)

	case objects.PropertyIdStatusFlags:
		return s.buildStatusFlagsCACK(invokeID, objType, instance)

	default:
		return nil, fmt.Errorf("unknown property %d", propID)
	}
}

// buildRPMResponse builds a ComplexACK for ReadPropertyMultiple.
// The RPM response contains object ID, then for each property: opening tag 1,
// context 2 (property ID), opening tag 4 (value), value, closing tag 4, closing tag 1.
func (s *Server) buildRPMResponse(
	dec services.ConfirmedReadPropMultDec,
	propIDs []uint16,
	objTypeStr string, store *points.PointStore, invokeID uint8,
) ([]byte, error) {
	objs := make([]objects.APDUPayload, 0, 3+len(propIDs)*5)

	// Object identifier
	objs = append(objs, objects.EncObjectIdentifier(true, 0, dec.ObjectType, dec.InstanceNum))

	// List of results opening tag
	objs = append(objs, objects.EncOpeningTag(1))

	for _, propID := range propIDs {
		// Property identifier
		objs = append(objs, objects.ContextTag(2, objects.EncUnsignedInteger(uint(propID))))

		// Property value opening tag
		objs = append(objs, objects.EncOpeningTag(4))

		valObj, err := s.getPropertyValueObject(propID, dec.ObjectType, dec.InstanceNum, objTypeStr, store)
		if err != nil {
			// On error, encode an error result instead
			objs = append(objs, &longEnumerated{value: uint8(objects.ErrorClassProperty)})
			objs = append(objs, &longEnumerated{value: ErrorCodeUnknownProperty})
		} else {
			objs = append(objs, valObj)
		}

		// Property value closing tag
		objs = append(objs, objects.EncClosingTag(4))
	}

	// List of results closing tag
	objs = append(objs, objects.EncClosingTag(1))

	return s.buildCACKWithObjects(services.ServiceConfirmedReadPropMultiple, invokeID, objs)
}

// buildRPMDeviceResponse builds a ComplexACK for RPM on a device object.
func (s *Server) buildRPMDeviceResponse(
	dec services.ConfirmedReadPropMultDec,
	propIDs []uint16, invokeID uint8,
) ([]byte, error) {
	deviceID := int(dec.InstanceNum)

	s.mu.Lock()
	_, exists := s.devices[deviceID]
	s.mu.Unlock()

	if !exists {
		return nil, fmt.Errorf("device %d not found", deviceID)
	}

	objs := make([]objects.APDUPayload, 0, 3+len(propIDs)*5)
	objs = append(objs, objects.EncObjectIdentifier(true, 0, objects.ObjectTypeDevice, uint32(deviceID)))
	objs = append(objs, objects.EncOpeningTag(1))

	for _, propID := range propIDs {
		objs = append(objs, objects.ContextTag(2, objects.EncUnsignedInteger(uint(propID))))
		objs = append(objs, objects.EncOpeningTag(4))

		switch propID {
		case objects.PropertyIdObjectIdentifier:
			objs = append(objs, objects.EncObjectIdentifier(false, objects.TagBACnetObjectIdentifier, objects.ObjectTypeDevice, uint32(deviceID)))
		case objects.PropertyIdObjectName:
			objs = append(objs, newLongString(fmt.Sprintf("sim-device-%d", deviceID)))
		case objects.PropertyIdObjectType:
			objs = append(objs, &longEnumerated{value: uint8(objects.ObjectTypeDevice)})
		case objects.PropertyIdDescription:
			objs = append(objs, newLongString(fmt.Sprintf("Simulated BACnet device %d", deviceID)))
		default:
			objs = append(objs, &longEnumerated{value: uint8(objects.ErrorClassProperty)})
			objs = append(objs, &longEnumerated{value: ErrorCodeUnknownProperty})
		}

		objs = append(objs, objects.EncClosingTag(4))
	}

	objs = append(objs, objects.EncClosingTag(1))

	return s.buildCACKWithObjects(services.ServiceConfirmedReadPropMultiple, invokeID, objs)
}

// getPropertyValueObject returns the encoded APDU payload for a property value.
func (s *Server) getPropertyValueObject(
	propID, objType uint16, instance uint32,
	objTypeStr string, store *points.PointStore,
) (objects.APDUPayload, error) {
	def, ok := store.DefinitionByKey(objTypeStr, int(instance))
	if !ok {
		return nil, fmt.Errorf("not found")
	}

	switch propID {
	case objects.PropertyIdPresentValue:
		val, err := store.ReadByKey(objTypeStr, int(instance))
		if err != nil {
			return nil, err
		}
		var v float64
		if val != nil {
			v = *val
		}
		return encodeValueObjectForType(objTypeStr, v), nil

	case objects.PropertyIdObjectIdentifier:
		return objects.EncObjectIdentifier(false, objects.TagBACnetObjectIdentifier, objType, instance), nil

	case objects.PropertyIdObjectName:
		return newLongString(def.VolttronName), nil

	case objects.PropertyIdObjectType:
		return &longEnumerated{value: uint8(objType)}, nil

	case objects.PropertyIdDescription:
		desc := def.ReferenceName
		if desc == "" {
			desc = def.VolttronName
		}
		return newLongString(desc), nil

	case objects.PropertyIdUnits:
		return &longEnumerated{value: uint8(unitsStringToCode(def.Units))}, nil

	case objects.PropertyIdStatusFlags:
		return objects.EncBitString([]bool{false, false, false, false}), nil

	default:
		return nil, fmt.Errorf("unknown property %d", propID)
	}
}

// buildCACK constructs a ComplexACK response with a single value.
// Uses custom encoding to handle strings longer than 4 bytes properly.
func (s *Server) buildCACK(service, invokeID uint8, objType uint16, instance uint32, propID uint16, value interface{}) ([]byte, error) {
	objs := make([]objects.APDUPayload, 5)
	objs[0] = objects.EncObjectIdentifier(true, 0, objType, instance)
	objs[1] = objects.ContextTag(1, objects.EncUnsignedInteger(uint(propID)))
	objs[2] = objects.EncOpeningTag(3)

	switch v := value.(type) {
	case float32:
		objs[3] = &longReal{value: v}
	case uint8:
		objs[3] = &longEnumerated{value: v}
	case uint16:
		objs[3] = &longUnsigned{value: uint32(v)}
	case string:
		objs[3] = newLongString(v)
	default:
		objs[3] = objects.EncReal(0)
	}

	objs[4] = objects.EncClosingTag(3)

	return s.buildCACKWithObjects(service, invokeID, objs)
}

// buildCACKWithObjects constructs a ComplexACK with arbitrary objects.
func (s *Server) buildCACKWithObjects(service, invokeID uint8, objs []objects.APDUPayload) ([]byte, error) {
	bvlc := plumbing.NewBVLC(plumbing.BVLCFuncUnicast)
	npdu := plumbing.NewNPDU(false, false, false, false)
	c := services.NewComplexACK(bvlc, npdu)

	c.APDU.Service = service
	c.APDU.InvokeID = invokeID
	c.APDU.Objects = objs
	c.SetLength()

	return c.MarshalBinary()
}

// buildSimpleACK constructs a SimpleACK response.
func (s *Server) buildSimpleACK(service, invokeID uint8) ([]byte, error) {
	bvlc := plumbing.NewBVLC(plumbing.BVLCFuncUnicast)
	npdu := plumbing.NewNPDU(false, false, false, false)
	sa := services.NewSimpleACK(bvlc, npdu)

	sa.APDU.Service = service
	sa.APDU.InvokeID = invokeID
	sa.SetLength()

	return sa.MarshalBinary()
}

// buildStatusFlagsCACK builds a ComplexACK for StatusFlags property.
// StatusFlags is a BIT STRING of 4 bits: {in-alarm, fault, overridden, out-of-service}.
func (s *Server) buildStatusFlagsCACK(invokeID uint8, objType uint16, instance uint32) ([]byte, error) {
	bvlc := plumbing.NewBVLC(plumbing.BVLCFuncUnicast)
	npdu := plumbing.NewNPDU(false, false, false, false)
	c := services.NewComplexACK(bvlc, npdu)

	c.APDU.Service = services.ServiceConfirmedReadProperty
	c.APDU.InvokeID = invokeID

	objs := make([]objects.APDUPayload, 5)
	objs[0] = objects.EncObjectIdentifier(true, 0, objType, instance)
	objs[1] = objects.ContextTag(1, objects.EncUnsignedInteger(uint(objects.PropertyIdStatusFlags)))
	objs[2] = objects.EncOpeningTag(3)
	objs[3] = objects.EncBitString([]bool{false, false, false, false})
	objs[4] = objects.EncClosingTag(3)

	c.APDU.Objects = objs
	c.SetLength()

	return c.MarshalBinary()
}

// sendError sends a BACnet Error APDU to the remote address.
func (s *Server) sendError(service, invokeID, errClass, errCode uint8, remoteAddr net.Addr) {
	bvlc := plumbing.NewBVLC(plumbing.BVLCFuncUnicast)
	npdu := plumbing.NewNPDU(false, false, false, false)
	e := services.NewError(bvlc, npdu)

	e.APDU.Service = service
	e.APDU.InvokeID = invokeID
	e.APDU.Objects = services.ErrorObjects(errClass, errCode)
	e.SetLength()

	errBytes, err := e.MarshalBinary()
	if err != nil {
		slog.Error("BACnet Error encode error", "error", err)
		return
	}

	if _, err := s.conn.WriteTo(errBytes, remoteAddr); err != nil {
		slog.Error("BACnet Error send error", "error", err)
	}
}

// encodeValueForType returns the properly typed value for a CACK response
// based on the BACnet object type.
func encodeValueForType(objTypeStr string, val float64) interface{} {
	switch {
	case strings.HasPrefix(objTypeStr, "binary"):
		if val > 0 {
			return uint8(1) // encoded as Enumerated
		}
		return uint8(0)
	case strings.HasPrefix(objTypeStr, "multiState"):
		return uint16(uint32(val)) // encoded as UnsignedInteger
	default:
		return float32(val) // encoded as Real
	}
}

// encodeValueObjectForType returns the encoded APDU object for a value.
func encodeValueObjectForType(objTypeStr string, val float64) objects.APDUPayload {
	switch {
	case strings.HasPrefix(objTypeStr, "binary"):
		if val > 0 {
			return &longEnumerated{value: 1}
		}
		return &longEnumerated{value: 0}
	case strings.HasPrefix(objTypeStr, "multiState"):
		return &longUnsigned{value: uint32(val)}
	default:
		return &longReal{value: float32(val)}
	}
}
