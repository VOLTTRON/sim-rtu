package bacnet

import (
	"github.com/jonalfarlinga/bacnet/objects"
)

// VendorIDSimulator is the BACnet vendor ID used for the simulator.
// Using a private/experimental range vendor ID.
const VendorIDSimulator uint16 = 999

// BACnet error codes not defined in the library.
const (
	ErrorCodeUnknownProperty   uint8 = 32
	ErrorCodeWriteAccessDenied uint8 = 40
	ErrorCodeInvalidDataType   uint8 = 9
)

// objectTypeMap maps CSV BACnet object type strings to BACnet numeric types.
var objectTypeMap = map[string]uint16{
	"analogInput":     objects.ObjectTypeAnalogInput,
	"analogOutput":    objects.ObjectTypeAnalogOutput,
	"analogValue":     objects.ObjectTypeAnalogValue,
	"binaryInput":     objects.ObjectTypeBinaryInput,
	"binaryOutput":    objects.ObjectTypeBinaryOutput,
	"binaryValue":     objects.ObjectTypeBinaryValue,
	"multiStateInput": objects.ObjectTypeMultiStateInput,
	"multiStateValue": objects.ObjectTypeMultiStateValue,
}

// reverseObjectTypeMap maps BACnet numeric types back to CSV strings.
var reverseObjectTypeMap map[uint16]string

func init() {
	reverseObjectTypeMap = make(map[uint16]string, len(objectTypeMap))
	for k, v := range objectTypeMap {
		reverseObjectTypeMap[v] = k
	}
}

// bacnetTypeToString converts a BACnet object type number to a CSV string.
func bacnetTypeToString(objType uint16) string {
	return reverseObjectTypeMap[objType]
}

// stringToBACnetType converts a CSV object type string to a BACnet type number.
func stringToBACnetType(s string) (uint16, bool) {
	v, ok := objectTypeMap[s]
	return v, ok
}

// BACnet engineering units enumeration (subset used by the registries).
var unitsMap = map[string]uint16{
	"degreesFahrenheit":          64,
	"degreesCelsius":             62,
	"percent":                    98,
	"percentRelativeHumidity":    29,
	"noUnits":                    95,
	"deltaDegreesFahrenheit":     120,
	"deltaDegreesCelsius":        121,
	"minutes":                    72,
	"seconds":                    73,
	"hours":                      71,
	"amperes":                    3,
	"volts":                      5,
	"kilowatts":                  28,
	"watts":                      47,
	"hertz":                      27,
	"revolutionsPerMinute":       104,
}

// unitsStringToCode converts a units string from the CSV to a BACnet units code.
func unitsStringToCode(units string) uint16 {
	if code, ok := unitsMap[units]; ok {
		return code
	}
	return 95 // noUnits
}
