package ld

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonLdOptions_Copy(t *testing.T) {
	expected := JsonLdOptions{
		Base:                  "base",
		CompactArrays:         true,
		ProcessingMode:        JsonLd_1_1,
		DocumentLoader:        NewDefaultDocumentLoader(nil),
		Embed:                 EmbedLast,
		Explicit:              true,
		RequireAll:            true,
		FrameDefault:          true,
		OmitDefault:           true,
		OmitGraph:             true,
		UseRdfType:            true,
		UseNativeTypes:        true,
		ProduceGeneralizedRdf: true,
		InputFormat:           "input",
		Format:                "format",
		Algorithm:             AlgorithmURGNA2012,
		UseNamespaces:         true,
		OutputForm:            "output",
		SafeMode:              true,
	}
	assert.Equal(t, expected, *expected.Copy())
}
