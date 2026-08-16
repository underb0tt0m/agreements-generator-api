package hasher

import (
	"fmt"

	"agreements-generator/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

//go:generate mockgen -source=hasher.go -destination=../mocks/hasher.go -package=mocks
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
		return nil, fmt.Errorf("error during hashing password: %v, %w", err, domain.ErrInternal)
	}

	return hash, nil
}

func (h *hasher) Compare(hash []byte, data string) error {
	if err := bcrypt.CompareHashAndPassword(hash, []byte(data)); err != nil {
		return fmt.Errorf("error during hash comparing: %v, %w", err, domain.ErrHashComparing)
	}
	return nil
}
