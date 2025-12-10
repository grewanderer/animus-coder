package connectjson

import (
	"encoding/json"

	"github.com/bufbuild/connect-go"
)

// Codec encodes/decodes generic Go structs as JSON for Connect handlers.
type Codec struct{}

func (Codec) Name() string {
	return "json"
}

func (Codec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (Codec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

var _ connect.Codec = (*Codec)(nil)
