package points

import (
	"fmt"
	"sync"
)

// PointStore is a thread-safe store for BACnet point values.
type PointStore struct {
	mu          sync.RWMutex
	values      map[string]*float64       // by volttron name
	objectIndex map[ObjectKey]string       // ObjectKey -> volttron name
	priorities  map[string]*PriorityArray  // writable points only
	definitions map[string]PointDefinition // by volttron name
}

// NewPointStore creates a store populated with point definitions.
// Default values are loaded into the value map and priority arrays.
func NewPointStore(defs []PointDefinition) *PointStore {
	ps := &PointStore{
		values:      make(map[string]*float64, len(defs)),
		objectIndex: make(map[ObjectKey]string, len(defs)),
		priorities:  make(map[string]*PriorityArray),
		definitions: make(map[string]PointDefinition, len(defs)),
	}

	for _, d := range defs {
		ps.definitions[d.VolttronName] = d
		ps.objectIndex[ObjectKey{ObjectType: d.BACnetObjectType, Index: d.Index}] = d.VolttronName

		if d.DefaultValue != nil {
			v := *d.DefaultValue
			ps.values[d.VolttronName] = &v
		}

		if d.Writable {
			pa := NewPriorityArray(d.DefaultValue)
			if d.DefaultValue != nil && d.WritePriority != nil {
				_ = pa.Write(*d.WritePriority, *d.DefaultValue)
			}
			ps.priorities[d.VolttronName] = pa
		}
	}

	return ps
}

// Read returns the current value of a point by its Volttron name.
func (ps *PointStore) Read(name string) (*float64, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if _, ok := ps.definitions[name]; !ok {
		return nil, fmt.Errorf("point %q not found", name)
	}
	v := ps.values[name]
	if v == nil {
		return nil, nil
	}
	cp := *v
	return &cp, nil
}

// ReadByKey returns the current value by BACnet object type and index.
func (ps *PointStore) ReadByKey(objectType string, index int) (*float64, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	name, ok := ps.objectIndex[ObjectKey{ObjectType: objectType, Index: index}]
	if !ok {
		return nil, fmt.Errorf("object %s[%d] not found", objectType, index)
	}
	v := ps.values[name]
	if v == nil {
		return nil, nil
	}
	cp := *v
	return &cp, nil
}

// Write sets a point value through the priority array (external writes).
func (ps *PointStore) Write(name string, value float64, priority int) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	def, ok := ps.definitions[name]
	if !ok {
		return fmt.Errorf("point %q not found", name)
	}
	if !def.Writable {
		return fmt.Errorf("point %q is not writable", name)
	}

	pa, ok := ps.priorities[name]
	if !ok {
		return fmt.Errorf("no priority array for point %q", name)
	}

	if err := pa.Write(priority, value); err != nil {
		return fmt.Errorf("write priority for %q: %w", name, err)
	}

	active := pa.ActiveValue()
	ps.values[name] = active
	return nil
}

// WriteByKey sets a point value by BACnet object type and index.
func (ps *PointStore) WriteByKey(objectType string, index int, value float64, priority int) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	key := ObjectKey{ObjectType: objectType, Index: index}
	name, ok := ps.objectIndex[key]
	if !ok {
		return fmt.Errorf("object %s[%d] not found", objectType, index)
	}

	def := ps.definitions[name]
	if !def.Writable {
		return fmt.Errorf("point %q is not writable", name)
	}

	pa, ok := ps.priorities[name]
	if !ok {
		return fmt.Errorf("no priority array for point %q", name)
	}

	if err := pa.Write(priority, value); err != nil {
		return fmt.Errorf("write priority for %q: %w", name, err)
	}

	active := pa.ActiveValue()
	ps.values[name] = active
	return nil
}

// SetInternal sets a point value directly, bypassing the priority array.
// Used by the simulation engine to update sensor readings.
func (ps *PointStore) SetInternal(name string, value float64) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if _, ok := ps.definitions[name]; !ok {
		return fmt.Errorf("point %q not found", name)
	}
	v := value
	ps.values[name] = &v
	return nil
}

// ReadAll returns a snapshot of all current point values.
func (ps *PointStore) ReadAll() map[string]*float64 {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	out := make(map[string]*float64, len(ps.values))
	for k, v := range ps.values {
		if v != nil {
			cp := *v
			out[k] = &cp
		}
	}
	return out
}

// Definitions returns a copy of all point definitions.
func (ps *PointStore) Definitions() []PointDefinition {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	defs := make([]PointDefinition, 0, len(ps.definitions))
	for _, d := range ps.definitions {
		defs = append(defs, d)
	}
	return defs
}

// ReadFloat is a convenience method returning a float64 (0 if nil/missing).
func (ps *PointStore) ReadFloat(name string) float64 {
	v, err := ps.Read(name)
	if err != nil || v == nil {
		return 0
	}
	return *v
}

// ReadBool is a convenience method returning true if the value is > 0.
func (ps *PointStore) ReadBool(name string) bool {
	return ps.ReadFloat(name) > 0
}
