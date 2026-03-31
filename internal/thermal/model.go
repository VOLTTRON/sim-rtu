package thermal

// Model implements an RC lumped-capacitance thermal model.
type Model struct {
	R float64 // thermal resistance (degF*hr/BTU)
	C float64 // thermal capacitance (BTU/degF)
}

// NewModel creates a thermal model with the given R and C values.
func NewModel(r, c float64) Model {
	return Model{R: r, C: c}
}

// Step computes the new zone temperature after dt hours.
// qHVAC is in BTU/hr (positive = heating, negative = cooling).
func (m Model) Step(zoneTemp, outdoorTemp, qHVAC, dtHours float64) float64 {
	dT := (dtHours/(m.R*m.C))*(outdoorTemp-zoneTemp) + (qHVAC*dtHours)/m.C
	return zoneTemp + dT
}
