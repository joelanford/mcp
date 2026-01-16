package types

import (
	"encoding/json"
	"os"
)

// OutputFormat specifies the response output format.
type OutputFormat string

const (
	OutputFormatJSON    OutputFormat = "json"
	OutputFormatCompact OutputFormat = "compact"
)

// GlobalOutputFormat controls the default output format for MCP responses.
var GlobalOutputFormat = OutputFormatCompact

func init() {
	if os.Getenv("MCP_OUTPUT_FORMAT") == "json" {
		GlobalOutputFormat = OutputFormatJSON
	}
}

// CompactMarshaler is implemented by types that support compact text output.
type CompactMarshaler interface {
	MarshalCompact() string
}

// MarshalResponse marshals a value to string, using compact format if available.
func MarshalResponse(v any) (string, error) {
	if GlobalOutputFormat == OutputFormatCompact {
		if cm, ok := v.(CompactMarshaler); ok {
			return cm.MarshalCompact(), nil
		}
	}
	data, err := json.Marshal(v)
	return string(data), err
}
