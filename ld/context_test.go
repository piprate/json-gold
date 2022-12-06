package ld

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestContext_Parse(t *testing.T) {
	expectedError := errors.New("failed")
	opts := NewJsonLdOptions("")
	opts.DocumentLoader = errorDocumentLoader{err: expectedError}

	t.Run("DocumentLoader can't resolve @context URL", func(t *testing.T) {
		ctx := NewContext(nil, opts)
		_, err := ctx.Parse("http://example.org/foo.ldjson")
		jsonLDError := new(JsonLdError)
		require.ErrorAs(t, err, &jsonLDError)
		assert.Equal(t, LoadingRemoteContextFailed, jsonLDError.Code)
		assert.ErrorIs(t, err, expectedError, "DocumentLoader error is not wrapped")
	})
	t.Run("DocumentLoader can't resolve @import", func(t *testing.T) {
		ctx := NewContext(nil, opts)
		_, err := ctx.Parse(map[string]interface{}{
			"@import": "http://example.org/foo.ldjson",
		})
		jsonLDError := new(JsonLdError)
		require.ErrorAs(t, err, &jsonLDError)
		assert.Equal(t, LoadingRemoteContextFailed, jsonLDError.Code)
		assert.ErrorIs(t, err, expectedError, "DocumentLoader error is not wrapped")
	})
}

type errorDocumentLoader struct {
	err error
}

func (l errorDocumentLoader) LoadDocument(u string) (*RemoteDocument, error) {
	return nil, l.err
}
