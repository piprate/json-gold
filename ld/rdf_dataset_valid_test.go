// Copyright 2015-2017 Piprate Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ld_test

import (
	"fmt"
	"testing"

	. "github.com/piprate/json-gold/ld"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const invalidHTTPIRI = "http://-bad-/p"

// validityDoc has n nodes that share a valid predicate, an invalid predicate,
// a valid datatype and an invalid datatype, so every IRI is seen many times.
func validityDoc(n int) map[string]interface{} {
	nodes := make([]interface{}, 0, n)
	for i := 0; i < n; i++ {
		nodes = append(nodes, map[string]interface{}{
			"@id":                      fmt.Sprintf("http://example.com/n%d", i),
			"http://example.com/name":  fmt.Sprintf("node %d", i),
			invalidHTTPIRI:             "dropped",
			"http://example.com/when":  map[string]interface{}{"@value": "2020-01-01", "@type": "http://www.w3.org/2001/XMLSchema#date"},
			"http://example.com/wrong": map[string]interface{}{"@value": "x", "@type": "http://-bad-/dt"},
			"http://example.com/link":  map[string]interface{}{"@id": "http://example.com/target"},
		})
	}
	return map[string]interface{}{"@graph": nodes}
}

// ToRDF remembers IRI validity within a call; the statements it keeps must be
// exactly those Quad.Valid accepts.
func TestToRDFDropsInvalidIRIsConsistently(t *testing.T) {
	require.False(t, IsURL(invalidHTTPIRI))

	const n = 20
	ds, err := NewJsonLdProcessor().ToRDF(validityDoc(n), NewJsonLdOptions(""))
	require.NoError(t, err)
	quads := ds.(*RDFDataset).GetQuads("@default")

	// name, when and link survive for every node; the invalid predicate and
	// the literal with an invalid datatype are dropped every time.
	assert.Len(t, quads, 3*n)
	for _, q := range quads {
		assert.True(t, q.Valid(), "kept an invalid quad: %v", q)
	}
}

func BenchmarkToRDFRepeatedIRIs(b *testing.B) {
	doc := validityDoc(200)
	proc := NewJsonLdProcessor()
	opts := NewJsonLdOptions("")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := proc.ToRDF(doc, opts); err != nil {
			b.Fatal(err)
		}
	}
}
