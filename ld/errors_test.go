package ld

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonLdError_Unwrap(t *testing.T) {
	t.Run("Details is error", func(t *testing.T) {
		err := errors.New("failed")
		assert.Equal(t, err, NewJsonLdError(UnknownError, err).Unwrap())
	})
	t.Run("Details is not an error", func(t *testing.T) {
		assert.Nil(t, NewJsonLdError(UnknownError, "failed").Unwrap())
	})
	t.Run("Details is nil", func(t *testing.T) {
		assert.Nil(t, NewJsonLdError(UnknownError, nil).Unwrap())
	})
}
