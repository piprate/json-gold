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
	"encoding/json"
	"sort"
	"testing"

	. "github.com/piprate/json-gold/ld"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// collectPointers expands doc and returns, for every node object reported,
// the pointer it came from mapped to the @id it expanded to. Objects with no
// @id map to "".
func collectPointers(t *testing.T, doc string, base string) map[string]string {
	t.Helper()

	var input interface{}
	require.NoError(t, json.Unmarshal([]byte(doc), &input))

	seen := map[string]string{}
	opts := NewJsonLdOptions(base)
	opts.ExpandedElementHandler = func(pointer string, expanded interface{}) {
		m, isMap := expanded.(map[string]interface{})
		if !isMap {
			return
		}
		id, _ := m["@id"].(string)
		seen[pointer] = id
	}

	_, err := NewJsonLdProcessor().Expand(input, opts)
	require.NoError(t, err)
	return seen
}

func TestExpandedElementHandlerPointers(t *testing.T) {
	const doc = `{
		"@context": {"ex": "http://example.com/"},
		"@graph": [
			{"@id": "ex:alice", "ex:name": "Alice"},
			{"@id": "ex:bob", "ex:knows": {"@id": "ex:carol"}}
		]
	}`

	seen := collectPointers(t, doc, "")

	// The pointer is relative to the document handed to the processor, so the
	// document itself is "".
	assert.Contains(t, seen, "")
	assert.Equal(t, "http://example.com/alice", seen["/@graph/0"])
	assert.Equal(t, "http://example.com/bob", seen["/@graph/1"])
	assert.Equal(t, "http://example.com/carol", seen["/@graph/1/ex:knows"])
}

// A term that maps to @id is expanded by the processor, so a caller relating
// output to input does not have to reimplement context processing.
func TestExpandedElementHandlerResolvesThroughContext(t *testing.T) {
	const doc = `{
		"@context": {"id": "@id", "ex": "http://example.com/"},
		"id": "ex:aliased",
		"ex:name": "Aliased"
	}`

	seen := collectPointers(t, doc, "")
	assert.Equal(t, "http://example.com/aliased", seen[""])
}

// A relative identifier is resolved against the base, again by the processor.
func TestExpandedElementHandlerResolvesRelativeIDs(t *testing.T) {
	const doc = `{"@id": "relative", "http://example.com/name": "Relative"}`

	seen := collectPointers(t, doc, "http://example.com/")
	assert.Equal(t, "http://example.com/relative", seen[""])
}

// A scoped context is applied where it applies, which is the case a caller
// scanning the source itself cannot get right without reimplementing expansion.
func TestExpandedElementHandlerHonoursScopedContexts(t *testing.T) {
	const doc = `{
		"@context": {
			"ex": "http://example.com/",
			"child": {"@id": "ex:child", "@context": {"other": "http://elsewhere.example/"}}
		},
		"@id": "ex:outer",
		"child": {"@id": "other:inner"}
	}`

	seen := collectPointers(t, doc, "")
	assert.Equal(t, "http://example.com/outer", seen[""])
	assert.Equal(t, "http://elsewhere.example/inner", seen["/child"])
}

// A node object with no @id becomes a blank node during expansion. The pointer
// still identifies where it was written, which is the only way to relate it to
// the source at all.
func TestExpandedElementHandlerReportsNodesWithoutID(t *testing.T) {
	const doc = `{
		"@context": {"ex": "http://example.com/"},
		"@id": "ex:outer",
		"ex:child": {"ex:name": "anonymous"}
	}`

	seen := collectPointers(t, doc, "")
	require.Contains(t, seen, "/ex:child")
	assert.Empty(t, seen["/ex:child"], "an anonymous node has no @id yet")
}

// RFC 6901 requires "~" and "/" to be escaped in a token, in that order.
func TestExpandedElementHandlerEscapesPointerTokens(t *testing.T) {
	const doc = `{
		"@context": {"http://example.com/a~b": {"@type": "@id"}, "http://example.com/c/d": {"@type": "@id"}},
		"@id": "http://example.com/outer",
		"http://example.com/a~b": {"@id": "http://example.com/tilde"},
		"http://example.com/c/d": {"@id": "http://example.com/slash"}
	}`

	seen := collectPointers(t, doc, "")

	pointers := make([]string, 0, len(seen))
	for p := range seen {
		pointers = append(pointers, p)
	}
	sort.Strings(pointers)

	assert.Contains(t, pointers, "/http:~1~1example.com~1a~0b")
	assert.Contains(t, pointers, "/http:~1~1example.com~1c~1d")
}

// A document that is a top-level array indexes from the root.
func TestExpandedElementHandlerTopLevelArray(t *testing.T) {
	const doc = `[
		{"@id": "http://example.com/first"},
		{"@id": "http://example.com/second"}
	]`

	seen := collectPointers(t, doc, "")
	assert.Equal(t, "http://example.com/first", seen["/0"])
	assert.Equal(t, "http://example.com/second", seen["/1"])
}

// The handler is optional, and leaving it unset must change nothing about the
// output — which is the property that makes it safe to add.
func TestExpandedElementHandlerDoesNotAffectOutput(t *testing.T) {
	const doc = `{
		"@context": {"ex": "http://example.com/"},
		"@graph": [
			{"@id": "ex:a", "ex:p": [{"@id": "ex:b"}, {"@value": "lit", "@language": "en"}]},
			{"ex:nested": {"ex:deep": {"@id": "ex:c"}}}
		]
	}`

	var input interface{}
	require.NoError(t, json.Unmarshal([]byte(doc), &input))

	plain, err := NewJsonLdProcessor().Expand(input, NewJsonLdOptions(""))
	require.NoError(t, err)

	opts := NewJsonLdOptions("")
	opts.ExpandedElementHandler = func(string, interface{}) {}
	observed, err := NewJsonLdProcessor().Expand(input, opts)
	require.NoError(t, err)

	assert.Equal(t, plain, observed)
}

func TestExpandedElementHandlerToRDF(t *testing.T) {
	const doc = `{
		"@context": {"ex": "http://example.com/"},
		"@id": "ex:subject",
		"ex:name": "Subject"
	}`

	var input interface{}
	require.NoError(t, json.Unmarshal([]byte(doc), &input))

	seen := map[string]string{}
	opts := NewJsonLdOptions("")
	opts.Format = "application/n-quads"
	opts.ExpandedElementHandler = func(pointer string, expanded interface{}) {
		if m, isMap := expanded.(map[string]interface{}); isMap {
			id, _ := m["@id"].(string)
			seen[pointer] = id
		}
	}

	// The callback is on expansion, which ToRDF performs, so it reaches
	// serialisation paths too.
	_, err := NewJsonLdProcessor().ToRDF(input, opts)
	require.NoError(t, err)
	assert.Equal(t, "http://example.com/subject", seen[""])
}

// benchExpandDoc is deliberately nested and repetitive, so that the pointer
// work — if there were any when no handler is installed — would show up.
func benchExpandDoc() interface{} {
	nodes := make([]interface{}, 0, 200)
	for i := 0; i < 200; i++ {
		nodes = append(nodes, map[string]interface{}{
			"@id":     "ex:node",
			"ex:name": "name",
			"ex:child": map[string]interface{}{
				"ex:grandchild": map[string]interface{}{"@id": "ex:deep"},
			},
			"ex:list": []interface{}{
				map[string]interface{}{"@id": "ex:a"},
				map[string]interface{}{"@id": "ex:b"},
			},
		})
	}
	return map[string]interface{}{
		"@context": map[string]interface{}{"ex": "http://example.com/"},
		"@graph":   nodes,
	}
}

func BenchmarkExpandWithoutHandler(b *testing.B) {
	input := benchExpandDoc()
	proc := NewJsonLdProcessor()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := proc.Expand(input, NewJsonLdOptions("")); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExpandWithHandler(b *testing.B) {
	input := benchExpandDoc()
	proc := NewJsonLdProcessor()
	opts := NewJsonLdOptions("")
	opts.ExpandedElementHandler = func(string, interface{}) {}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := proc.Expand(input, opts); err != nil {
			b.Fatal(err)
		}
	}
}
