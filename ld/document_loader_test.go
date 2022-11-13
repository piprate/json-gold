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
	"bytes"
	"testing"

	. "github.com/piprate/json-gold/ld"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDocument(t *testing.T) {
	dl := NewDefaultDocumentLoader(nil)

	rd, _ := dl.LoadDocument("testdata/expand/0002-in.jsonld")

	assert.Equal(t, "t1", rd.Document.(map[string]interface{})["@type"])
}

func loadBenchData(tb testing.TB) *RDFDataset {
	tb.Helper()

	dl := NewDefaultDocumentLoader(nil)
	rd, err := dl.LoadDocument("testdata/compact-manifest.jsonld")
	require.Nil(tb, err)
	proc := NewJsonLdProcessor()
	triples, err := proc.ToRDF(rd, NewJsonLdOptions(""))
	require.Nil(tb, err)
	return triples.(*RDFDataset)
}

func BenchmarkLoadNQuads(b *testing.B) {
	buf := bytes.NewBuffer(nil)
	err := (&NQuadRDFSerializer{}).SerializeTo(buf, loadBenchData(b))
	require.Nil(b, err)

	data := buf.Bytes()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = ParseNQuadsFrom(data)
		require.Nil(b, err)
	}
}

func TestParseLinkHeader(t *testing.T) {
	rval := ParseLinkHeader("<remote-doc/0010-context.jsonld>; rel=\"http://www.w3.org/ns/json-ld#context\"")

	assert.Equal(
		t,
		map[string][]map[string]string{
			"http://www.w3.org/ns/json-ld#context": {{
				"target": "remote-doc/0010-context.jsonld",
				"rel":    "http://www.w3.org/ns/json-ld#context",
			}},
		},
		rval,
	)
}

func TestCachingDocumentLoaderLoadDocument(t *testing.T) {
	cl := NewCachingDocumentLoader(NewDefaultDocumentLoader(nil))

	_ = cl.PreloadWithMapping(map[string]string{
		"http://www.example.com/expand/0002-in.jsonld": "testdata/expand/0002-in.jsonld",
	})

	rd, _ := cl.LoadDocument("http://www.example.com/expand/0002-in.jsonld")

	assert.Equal(t, "t1", rd.Document.(map[string]interface{})["@type"])
}
