package points

// PointDefinition describes a single BACnet point parsed from a CSV registry.
type PointDefinition struct {
	ReferenceName    string
	VolttronName     string
	Units            string
	UnitDetails      string
	BACnetObjectType string // "analogInput", "analogOutput", "analogValue", "binaryInput", "binaryOutput", "binaryValue", "multiStateInput", "multiStateValue"
	PropertyName     string
	Writable         bool
	Index            int
	WritePriority    *int
	DefaultValue     *float64
	Notes            string
	Active           bool // from openstat.csv extra column; true by default for other formats
}

// ObjectKey uniquely identifies a BACnet object by type and index.
type ObjectKey struct {
	ObjectType string
	Index      int
}
