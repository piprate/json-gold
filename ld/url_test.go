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
	"testing"

	. "github.com/piprate/json-gold/ld"
	"github.com/stretchr/testify/assert"
)

func TestJsonLdUrl(t *testing.T) {
	parsedURL := ParseURL("http://www.example.com")

	assert.Equal(t, "http:", parsedURL.Protocol)
	assert.Equal(t, "www.example.com", parsedURL.Host)
}

func TestRemoveBase(t *testing.T) {
	result := RemoveBase(
		"http://json-ld.org/test-suite/tests/compact-0045-in.jsonld",
		"http://json-ld.org/test-suite/parent-node",
	)
	assert.Equal(t, "../parent-node", result)

	result = RemoveBase(
		"http://example.com/",
		"http://example.com/relative-url",
	)
	assert.Equal(t, "relative-url", result)

	result = RemoveBase(
		"http://json-ld.org/test-suite/tests/compact-0066-in.jsonld",
		"http://json-ld.org/test-suite/",
	)
	assert.Equal(t, "../", result)

	result = RemoveBase(
		"http://example.com/api/things/1",
		"http://example.com/api/things/1",
	)
	assert.Equal(t, "1", result)
}

func TestResolve(t *testing.T) {
	assert.Equal(t, "http://example.com/a/b",
		Resolve("http://example.com/a/", "b"))
	assert.Equal(t, "http://example.com/b",
		Resolve("http://example.com/a", "b"))
	assert.Equal(t, "http://other.example/b",
		Resolve("http://example.com/a", "http://other.example/b"))
	assert.Equal(t, "http://example.com/a?q=1",
		Resolve("http://example.com/a#frag", "?q=1"))
	assert.Equal(t, "http://example.com/a", Resolve("http://example.com/a", ""))
	assert.Equal(t, "b", Resolve("", "b"))
}

// Resolve discarded the error from url.Parse and dereferenced the nil URL it
// returns alongside it, so an unparseable base or reference panicked instead of
// being rejected. Reachable from Expand and ToRDF through an @id, @base or
// @vocab, i.e. from any untrusted document.
func TestResolveWithUnparseableInput(t *testing.T) {
	// An invalid reference: url.Parse fails on the incomplete percent-escape,
	// and ResolveReference was called with the resulting nil.
	assert.NotPanics(t, func() {
		assert.Equal(t, "%", Resolve("http://example.com/", "%"))
	})
	assert.NotPanics(t, func() {
		assert.Equal(t, "%zz", Resolve("http://example.com/", "%zz"))
	})
	assert.NotPanics(t, func() {
		assert.Equal(t, "\x7f", Resolve("http://example.com/", "\x7f"))
	})

	// An invalid base: uri itself was nil.
	assert.NotPanics(t, func() {
		assert.Equal(t, "b", Resolve("%", "b"))
	})
	// The query branch returns before the reference is parsed, so an invalid
	// base reaches it too.
	assert.NotPanics(t, func() {
		assert.Equal(t, "?q=1", Resolve("%", "?q=1"))
	})
}

// The panic was reachable from the public API, which is what made it a problem
// for anything parsing documents it did not write.
func TestExpandWithUnparseableID(t *testing.T) {
	doc := map[string]interface{}{
		"@id":                     "%",
		"http://example.com/prop": "value",
	}

	assert.NotPanics(t, func() {
		_, _ = NewJsonLdProcessor().Expand(doc, NewJsonLdOptions("http://example.com/"))
	})

	opts := NewJsonLdOptions("http://example.com/")
	opts.Format = "application/n-quads"
	assert.NotPanics(t, func() {
		_, _ = NewJsonLdProcessor().ToRDF(doc, opts)
	})
}
