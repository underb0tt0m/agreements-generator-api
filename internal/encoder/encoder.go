package encoder

//go:generate mockgen -source=encoder.go -destination=../mocks/encoder.go -package=mocks
type Encoder interface {
	Marshal(object any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}
