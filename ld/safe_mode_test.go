package ld_test

import (
	"testing"

	. "github.com/piprate/json-gold/ld"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func safeModeTestDoc() map[string]interface{} {
	return map[string]interface{}{
		"@context": map[string]interface{}{
			"name": "http://schema.org/name",
		},
		"@id":       "http://example.com/alice",
		"name":      "Alice",
		"undefined": "not in the context",
	}
}

func TestSafeModeRejectsUndefinedTerms(t *testing.T) {
	proc := NewJsonLdProcessor()

	for _, alg := range []string{AlgorithmURDNA2015, AlgorithmURGNA2012} {
		opts := NewJsonLdOptions("")
		opts.SafeMode = true
		opts.Algorithm = alg
		opts.Format = "application/n-quads"

		_, err := proc.Expand(safeModeTestDoc(), opts)
		assert.Error(t, err, "Expand")

		_, err = proc.ToRDF(safeModeTestDoc(), opts)
		assert.Error(t, err, "ToRDF")

		_, err = proc.Normalize(safeModeTestDoc(), opts)
		assert.Error(t, err, "Normalize with %s", alg)
	}
}

func TestNormalizeWithoutSafeModeDropsUndefinedTerms(t *testing.T) {
	proc := NewJsonLdProcessor()
	opts := NewJsonLdOptions("")
	opts.Algorithm = AlgorithmURDNA2015
	opts.Format = "application/n-quads"

	out, err := proc.Normalize(safeModeTestDoc(), opts)
	require.NoError(t, err)
	assert.Equal(t, "<http://example.com/alice> <http://schema.org/name> \"Alice\" .\n", out)
}
