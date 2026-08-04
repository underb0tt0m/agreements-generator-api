package encoder

type Encoder interface {
	Marshal(object any) ([]byte, error)
}
