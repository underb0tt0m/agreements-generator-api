package encoder_json

import (
	"encoding/json"

	"agreements-generator/internal/encoder"
)

type json_encoder struct {
}

func New() encoder.Encoder {
	return &json_encoder{}
}

func (e *json_encoder) Marshal(object any) ([]byte, error) {
	return json.Marshal(object)
}
