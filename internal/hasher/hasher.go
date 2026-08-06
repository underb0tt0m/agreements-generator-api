package hasher

import (
	"agreements-generator/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

type Hasher interface {
	Hash(data string) ([]byte, error)
	Compare(hash []byte, data string) error
}

type hasher struct {
	cost int
}

func New(cost int) Hasher {
	return &hasher{cost: cost}
}

func (h *hasher) Hash(data string) ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(data), h.cost)
	if err != nil {
		return nil, domain.ErrInternal.Wrap("can't hash string", err)
	}

	return hash, nil
}

func (h *hasher) Compare(hash []byte, data string) error {
	if err := bcrypt.CompareHashAndPassword(hash, []byte(data)); err != nil {
		return domain.ErrUnauthorized.Wrap("invalid password", err)
	}
	return nil
}
