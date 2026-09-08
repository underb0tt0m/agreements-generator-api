package encoder_json

import (
	"encoding/json"

	"agreements-generator/internal/encoder"
)

type jsonEncoder struct {
}

func New() encoder.Encoder {
	return &jsonEncoder{}
}

func (*jsonEncoder) Marshal(object any) ([]byte, error) {
	return json.Marshal(object)
}

func (*jsonEncoder) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
