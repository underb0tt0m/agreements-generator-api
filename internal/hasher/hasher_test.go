package hasher

import (
	"errors"
	"testing"

	"agreements-generator/internal/domain"
)

func TestHasher_Hash(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		h := hasher{cost: 10}

		_, err := h.Hash("success")

		if err != nil {
			t.Errorf("Hash(): got %v, expected: %v", err, nil)
		}
	})

	t.Run("Error", func(t *testing.T) {
		h := hasher{cost: 2 << 10}

		_, err := h.Hash("error")

		if err == nil {
			t.Errorf("Hash(): got %v, expected: %v", err, domain.ErrInternal)
		}

		if err != nil && !errors.Is(err, domain.ErrInternal) {
			t.Errorf("Hash(): got %v, expected: %v", err, domain.ErrInternal)
		}
	})
}

func TestHasher_Compare(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		h := hasher{cost: 10}

		hash, _ := h.Hash("word")

		err := h.Compare(hash, "word")

		if err != nil {
			t.Errorf("Hash(): got %v, expected: %v", err, nil)
		}
	})

	t.Run("Error", func(t *testing.T) {
		h := hasher{cost: 10}

		hash, _ := h.Hash("word")

		err := h.Compare(hash, "another word")

		if err == nil {
			t.Errorf("Compare(): got %v, expected: %v", err, domain.ErrHashComparing)
		}

		if err != nil && !errors.Is(err, domain.ErrHashComparing) {
			t.Errorf("Compare(): got %v, expected: %v", err, domain.ErrHashComparing)
		}
	})
}
