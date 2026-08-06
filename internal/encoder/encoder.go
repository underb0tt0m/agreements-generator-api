package encoder

type Encoder interface {
	Marshal(object any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}
