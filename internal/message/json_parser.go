package message

import (
	"encoding/json"
	"fmt"
)

// ParseDynamicJSON parses JSON data from a byte slice into a DynamicMessage map.
// It returns ErrJSONUnmarshalFailed (wrapping the original error) if unmarshalling fails.
func ParseDynamicJSON(data []byte) (DynamicMessage, error) {
	var msg DynamicMessage

	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrJSONUnmarshalFailed, err)
	}
	return msg, nil
}
