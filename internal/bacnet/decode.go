package bacnet

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/services"
)

// longString is an APDUPayload that properly encodes strings longer than 4 bytes.
// The library's EncString only handles strings with total data length <= 4.
type longString struct {
	data []byte // charset byte + UTF-8 string
}

func newLongString(s string) *longString {
	return &longString{
		data: append([]byte{0x00}, []byte(s)...),
	}
}

func (ls *longString) MarshalBinary() ([]byte, error) {
	b := make([]byte, ls.MarshalLen())
	if err := ls.MarshalTo(b); err != nil {
		return nil, err
	}
	return b, nil
}

func (ls *longString) MarshalTo(b []byte) error {
	if len(b) < ls.MarshalLen() {
		return fmt.Errorf("buffer too short")
	}
	dataLen := len(ls.data)
	if dataLen < 5 {
		// Short form: tag byte with length in lower 3 bits
		b[0] = objects.TagCharacterString<<4 | uint8(dataLen)
		copy(b[1:], ls.data)
	} else if dataLen <= 253 {
		// Extended form: tag byte with 5 in lower 3 bits, then 1-byte length
		b[0] = objects.TagCharacterString<<4 | 5
		b[1] = uint8(dataLen)
		copy(b[2:], ls.data)
	} else {
		// Long extended form: tag byte with 5, then 0xFE, then 2-byte length
		b[0] = objects.TagCharacterString<<4 | 5
		b[1] = 0xFE
		binary.BigEndian.PutUint16(b[2:4], uint16(dataLen))
		copy(b[4:], ls.data)
	}
	return nil
}

func (ls *longString) MarshalLen() int {
	dataLen := len(ls.data)
	if dataLen < 5 {
		return 1 + dataLen
	} else if dataLen <= 253 {
		return 2 + dataLen
	}
	return 4 + dataLen
}

func (ls *longString) UnmarshalBinary(_ []byte) error {
	return nil // not needed for server responses
}

// longReal is an APDUPayload for encoding Real (float32) values with
// proper extended length encoding (4 bytes needs LVT=4, which works,
// but we provide it for consistency).
type longEnumerated struct {
	value uint8
}

func (le *longEnumerated) MarshalBinary() ([]byte, error) {
	return []byte{objects.TagEnumerated<<4 | 1, le.value}, nil
}

func (le *longEnumerated) MarshalTo(b []byte) error {
	b[0] = objects.TagEnumerated<<4 | 1
	b[1] = le.value
	return nil
}

func (le *longEnumerated) MarshalLen() int { return 2 }
func (le *longEnumerated) UnmarshalBinary(_ []byte) error { return nil }

// longReal properly encodes a BACnet Real (float32).
type longReal struct {
	value float32
}

func (lr *longReal) MarshalBinary() ([]byte, error) {
	b := make([]byte, 5)
	b[0] = objects.TagReal<<4 | 4
	binary.BigEndian.PutUint32(b[1:], math.Float32bits(lr.value))
	return b, nil
}

func (lr *longReal) MarshalTo(b []byte) error {
	b[0] = objects.TagReal<<4 | 4
	binary.BigEndian.PutUint32(b[1:], math.Float32bits(lr.value))
	return nil
}

func (lr *longReal) MarshalLen() int             { return 5 }
func (lr *longReal) UnmarshalBinary(_ []byte) error { return nil }

// longUnsigned properly encodes a BACnet Unsigned Integer.
type longUnsigned struct {
	value uint32
}

func (lu *longUnsigned) MarshalBinary() ([]byte, error) {
	b := make([]byte, lu.MarshalLen())
	if err := lu.MarshalTo(b); err != nil {
		return nil, err
	}
	return b, nil
}

func (lu *longUnsigned) MarshalTo(b []byte) error {
	switch {
	case lu.value <= 0xFF:
		b[0] = objects.TagUnsignedInteger<<4 | 1
		b[1] = byte(lu.value)
	case lu.value <= 0xFFFF:
		b[0] = objects.TagUnsignedInteger<<4 | 2
		binary.BigEndian.PutUint16(b[1:], uint16(lu.value))
	default:
		b[0] = objects.TagUnsignedInteger<<4 | 4
		binary.BigEndian.PutUint32(b[1:], lu.value)
	}
	return nil
}

func (lu *longUnsigned) MarshalLen() int {
	switch {
	case lu.value <= 0xFF:
		return 2
	case lu.value <= 0xFFFF:
		return 3
	default:
		return 5
	}
}

func (lu *longUnsigned) UnmarshalBinary(_ []byte) error { return nil }

// ReadPropertyRequest holds the decoded fields from a ReadProperty request.
type ReadPropertyRequest struct {
	ObjectType  uint16
	InstanceNum uint32
	PropertyID  uint16
}

// WritePropertyRequest holds the decoded fields from a WriteProperty request.
type WritePropertyRequest struct {
	ObjectType  uint16
	InstanceNum uint32
	PropertyID  uint16
	Value       float64
	Priority    int
}

// decodeReadProperty manually decodes a ConfirmedReadProperty request,
// working around the library's Decode() bug where it maps PropertyID
// to context tag 2 instead of context tag 1.
func decodeReadProperty(rp *services.ConfirmedReadProperty) (ReadPropertyRequest, error) {
	req := ReadPropertyRequest{}

	for _, obj := range rp.APDU.Objects {
		encObj, ok := obj.(*objects.Object)
		if !ok {
			continue
		}

		// Skip opening/closing tags
		if encObj.Length == 6 || encObj.Length == 7 {
			continue
		}

		if !encObj.TagClass {
			continue
		}

		switch encObj.TagNumber {
		case 0: // Object Identifier
			oid, err := objects.DecObjectIdentifier(obj)
			if err != nil {
				return req, fmt.Errorf("decode object ID: %w", err)
			}
			req.ObjectType = oid.ObjectType
			req.InstanceNum = oid.InstanceNumber
		case 1: // Property Identifier
			propID, err := objects.DecUnsignedInteger(obj)
			if err != nil {
				return req, fmt.Errorf("decode property ID: %w", err)
			}
			req.PropertyID = uint16(propID)
		}
	}

	return req, nil
}

// decodeWriteProperty manually decodes a ConfirmedWriteProperty request,
// working around the library's Decode() bug with object count validation.
func decodeWriteProperty(wp *services.ConfirmedWriteProperty) (WritePropertyRequest, error) {
	req := WritePropertyRequest{
		Priority: 16, // default BACnet priority
	}

	inValueContext := false

	for _, obj := range wp.APDU.Objects {
		encObj, ok := obj.(*objects.Object)
		if !ok {
			continue
		}

		// Track opening/closing tag 3 (property value)
		if encObj.Length == 6 && encObj.TagNumber == 3 {
			inValueContext = true
			continue
		}
		if encObj.Length == 7 && encObj.TagNumber == 3 {
			inValueContext = false
			continue
		}
		// Skip other opening/closing tags
		if encObj.Length == 6 || encObj.Length == 7 {
			continue
		}

		if encObj.TagClass {
			switch encObj.TagNumber {
			case 0: // Object Identifier
				oid, err := objects.DecObjectIdentifier(obj)
				if err != nil {
					return req, fmt.Errorf("decode object ID: %w", err)
				}
				req.ObjectType = oid.ObjectType
				req.InstanceNum = oid.InstanceNumber
			case 1: // Property Identifier
				propID, err := objects.DecUnsignedInteger(obj)
				if err != nil {
					return req, fmt.Errorf("decode property ID: %w", err)
				}
				req.PropertyID = uint16(propID)
			case 4: // Priority
				priority, err := objects.DecUnsignedInteger(obj)
				if err != nil {
					return req, fmt.Errorf("decode priority: %w", err)
				}
				req.Priority = int(priority)
			}
		} else if inValueContext {
			// Application-tagged value inside the value context
			val, err := decodeAppTagValue(encObj, obj)
			if err != nil {
				return req, fmt.Errorf("decode value: %w", err)
			}
			req.Value = val
		}
	}

	return req, nil
}

// decodeAppTagValue extracts a float64 from an application-tagged object.
func decodeAppTagValue(encObj *objects.Object, obj objects.APDUPayload) (float64, error) {
	switch encObj.TagNumber {
	case objects.TagReal:
		v, err := objects.DecReal(obj)
		if err != nil {
			return 0, err
		}
		return float64(v), nil
	case objects.TagDouble:
		v, err := objects.DecDouble(obj)
		if err != nil {
			return 0, err
		}
		return v, nil
	case objects.TagUnsignedInteger:
		v, err := objects.DecUnsignedInteger(obj)
		if err != nil {
			return 0, err
		}
		return float64(v), nil
	case objects.TagSignedInteger:
		v, err := objects.DecSignedInteger(obj)
		if err != nil {
			return 0, err
		}
		return float64(v), nil
	case objects.TagEnumerated:
		v, err := objects.DecEnumerated(obj)
		if err != nil {
			return 0, err
		}
		return float64(v), nil
	case objects.TagBoolean:
		v, err := objects.DecBoolean(obj)
		if err != nil {
			return 0, err
		}
		if v {
			return 1.0, nil
		}
		return 0.0, nil
	default:
		return 0, fmt.Errorf("unsupported application tag %d", encObj.TagNumber)
	}
}
