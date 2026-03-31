package points

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// defaultValueRe matches "(default X.Y)" in the Unit Details column.
var defaultValueRe = regexp.MustCompile(`\(default\s+([-\d.]+)\)`)

// ParseRegistry reads a CSV registry file and returns point definitions.
// It auto-detects the format (schneider, openstat, dent) based on the
// presence of an "active" header column.
func ParseRegistry(path string) ([]PointDefinition, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open registry %s: %w", path, err)
	}
	defer f.Close()

	return ParseRegistryReader(f)
}

// ParseRegistryReader parses a CSV registry from a reader.
func ParseRegistryReader(r io.Reader) ([]PointDefinition, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	colMap := buildColumnMap(header)
	hasActive := colMap["active"] >= 0

	var defs []PointDefinition
	lineNum := 1
	for {
		lineNum++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		def, err := parseRecord(record, colMap, hasActive)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		defs = append(defs, def)
	}

	return defs, nil
}

func buildColumnMap(header []string) map[string]int {
	m := map[string]int{
		"reference point name": -1,
		"volttron point name":  -1,
		"units":                -1,
		"unit details":         -1,
		"bacnet object type":   -1,
		"property":             -1,
		"writable":             -1,
		"index":                -1,
		"write priority":       -1,
		"notes":                -1,
		"active":               -1,
	}

	for i, h := range header {
		key := strings.ToLower(strings.TrimSpace(h))
		if _, ok := m[key]; ok {
			m[key] = i
		}
	}
	return m
}

func getField(record []string, colMap map[string]int, key string) string {
	idx := colMap[key]
	if idx < 0 || idx >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[idx])
}

func parseRecord(record []string, colMap map[string]int, hasActive bool) (PointDefinition, error) {
	var def PointDefinition

	def.ReferenceName = getField(record, colMap, "reference point name")
	def.VolttronName = getField(record, colMap, "volttron point name")
	def.Units = getField(record, colMap, "units")
	def.UnitDetails = getField(record, colMap, "unit details")
	def.BACnetObjectType = getField(record, colMap, "bacnet object type")
	def.PropertyName = getField(record, colMap, "property")
	def.Notes = getField(record, colMap, "notes")

	// Writable
	writableStr := strings.ToUpper(getField(record, colMap, "writable"))
	def.Writable = writableStr == "TRUE"

	// Index
	indexStr := getField(record, colMap, "index")
	if indexStr != "" {
		idx, err := strconv.Atoi(indexStr)
		if err != nil {
			return def, fmt.Errorf("invalid index %q: %w", indexStr, err)
		}
		def.Index = idx
	}

	// Write Priority
	wpStr := getField(record, colMap, "write priority")
	if wpStr != "" {
		wp, err := strconv.Atoi(wpStr)
		if err != nil {
			return def, fmt.Errorf("invalid write priority %q: %w", wpStr, err)
		}
		def.WritePriority = &wp
	}

	// Default value from Unit Details "(default X.Y)"
	if def.UnitDetails != "" {
		matches := defaultValueRe.FindStringSubmatch(def.UnitDetails)
		if len(matches) > 1 {
			v, err := strconv.ParseFloat(matches[1], 64)
			if err == nil {
				def.DefaultValue = &v
			}
		}
	}

	// Active column (openstat format)
	if hasActive {
		activeStr := strings.ToUpper(getField(record, colMap, "active"))
		def.Active = activeStr == "TRUE"
	} else {
		def.Active = true
	}

	return def, nil
}
