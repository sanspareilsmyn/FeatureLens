package message

import "errors"

var (
	ErrJSONUnmarshalFailed = errors.New("failed to unmarshal JSON message")
)
