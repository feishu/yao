package x2j

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// ProcessXmlToJson converts XML to JSON
// Args:
//   - args[0]: XML string or byte array
//   - args[1]: (optional) safe encoding flag (default: false)
//
// Returns: JSON byte array
//
// Example:
//
//	process.New("utils.x2j.XmlToJson", "<root><name>test</name></root>").Run()
func ProcessXmlToJson(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	// Get XML input
	var xmlVal []byte
	switch v := process.Args[0].(type) {
	case string:
		xmlVal = []byte(v)
	case []byte:
		xmlVal = v
	default:
		exception.New("Invalid XML input type, expected string or []byte", 400).Throw()
	}

	// Get optional safe encoding flag
	safeEncoding := false
	if len(process.Args) > 1 {
		if se, ok := process.Args[1].(bool); ok {
			safeEncoding = se
		}
	}

	// Convert XML to JSON
	jsonVal, err := XmlToJson(xmlVal, safeEncoding)
	if err != nil {
		exception.New("Failed to convert XML to JSON: %v", 500, err.Error()).Throw()
	}

	return jsonVal
}

// ProcessXmlToMap converts XML to map
// Args:
//   - args[0]: XML string or byte array
//
// Returns: map[string]interface{}
//
// Example:
//
//	process.New("utils.x2j.XmlToMap", "<root><name>test</name></root>").Run()
func ProcessXmlToMap(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	// Get XML input
	var xmlVal []byte
	switch v := process.Args[0].(type) {
	case string:
		xmlVal = []byte(v)
	case []byte:
		xmlVal = v
	default:
		exception.New("Invalid XML input type, expected string or []byte", 400).Throw()
	}

	// Convert XML to map
	m, err := XmlToMap(xmlVal)
	if err != nil {
		exception.New("Failed to convert XML to map: %v", 500, err.Error()).Throw()
	}

	return m
}

// ProcessMapToXml converts map to XML
// Args:
//   - args[0]: map[string]interface{}
//
// Returns: XML byte array
//
// Example:
//
//	process.New("utils.x2j.MapToXml", map[string]interface{}{"name": "test"}).Run()
func ProcessMapToXml(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	// Get map input
	m := process.ArgsMap(0)

	// Convert map to XML
	xmlVal, err := MapToXml(m)
	if err != nil {
		exception.New("Failed to convert map to XML: %v", 500, err.Error()).Throw()
	}

	return xmlVal
}

// ProcessXmlValuesForTag extracts values for a specific XML tag
// Args:
//   - args[0]: XML string or byte array
//   - args[1]: tag name
//   - args[2...]: (optional) attribute key:value pairs
//
// Returns: []interface{} containing the values
//
// Example:
//
//	process.New("utils.x2j.XmlValuesForTag", xmlStr, "name", "-id:123").Run()
func ProcessXmlValuesForTag(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	// Get XML input
	var xmlVal []byte
	switch v := process.Args[0].(type) {
	case string:
		xmlVal = []byte(v)
	case []byte:
		xmlVal = v
	default:
		exception.New("Invalid XML input type, expected string or []byte", 400).Throw()
	}

	// Get tag name
	tag := process.ArgsString(1)

	// Get optional attributes
	attrs := []string{}
	for i := 2; i < len(process.Args); i++ {
		if attr, ok := process.Args[i].(string); ok {
			attrs = append(attrs, attr)
		}
	}

	// Extract values
	values, err := XmlValuesForTag(xmlVal, tag, attrs...)
	if err != nil {
		exception.New("Failed to extract values for tag: %v", 500, err.Error()).Throw()
	}

	return values
}

// ProcessXmlPathsForTag finds all paths for a specific XML tag
// Args:
//   - args[0]: XML string or byte array
//   - args[1]: tag name
//
// Returns: []string containing the paths
//
// Example:
//
//	process.New("utils.x2j.XmlPathsForTag", xmlStr, "name").Run()
func ProcessXmlPathsForTag(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	// Get XML input
	var xmlVal []byte
	switch v := process.Args[0].(type) {
	case string:
		xmlVal = []byte(v)
	case []byte:
		xmlVal = v
	default:
		exception.New("Invalid XML input type, expected string or []byte", 400).Throw()
	}

	// Get tag name
	tag := process.ArgsString(1)

	// Get paths
	paths, err := XmlPathsForTag(xmlVal, tag)
	if err != nil {
		exception.New("Failed to get paths for tag: %v", 500, err.Error()).Throw()
	}

	return paths
}

// ProcessXmlUpdateValsForPath updates values at a specific path in XML
// Args:
//   - args[0]: XML string or byte array
//   - args[1]: new value
//   - args[2]: dot-notation path
//   - args[3...]: (optional) subkey pairs
//
// Returns: Updated XML byte array
//
// Example:
//
//	process.New("utils.x2j.XmlUpdateValsForPath", xmlStr, "newValue", "root.name").Run()
func ProcessXmlUpdateValsForPath(process *process.Process) interface{} {
	process.ValidateArgNums(3)

	// Get XML input
	var xmlVal []byte
	switch v := process.Args[0].(type) {
	case string:
		xmlVal = []byte(v)
	case []byte:
		xmlVal = v
	default:
		exception.New("Invalid XML input type, expected string or []byte", 400).Throw()
	}

	// Get new value and path
	newValue := process.Args[1]
	path := process.ArgsString(2)

	// Get optional subkeys
	subkeys := []string{}
	for i := 3; i < len(process.Args); i++ {
		if subkey, ok := process.Args[i].(string); ok {
			subkeys = append(subkeys, subkey)
		}
	}

	// Update values
	updatedXml, err := XmlUpdateValsForPath(xmlVal, newValue, path, subkeys...)
	if err != nil {
		exception.New("Failed to update XML values: %v", 500, err.Error()).Throw()
	}

	return updatedXml
}
