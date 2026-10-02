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

package ld

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// keywords is the set IsKeyword accepted before it was rewritten as a switch.
var keywords = []string{
	"@always", "@base", "@container", "@context", "@default",
	"@direction", "@embed", "@explicit", "@first", "@json",
	"@id", "@included", "@index", "@graph", "@import",
	"@language", "@last", "@list", "@nest", "@never",
	"@none", "@null", "@omitDefault", "@once", "@prefix",
	"@preserve", "@propagate", "@protected", "@requireAll", "@reverse",
	"@set", "@type", "@value", "@version", "@vocab",
}

func TestIsKeyword(t *testing.T) {
	for _, k := range keywords {
		assert.True(t, IsKeyword(k), k)
		// The same text as a non-string value is not a keyword.
		assert.False(t, IsKeyword([]byte(k)), "[]byte %s", k)
	}
	for _, v := range []interface{}{
		nil, "", "@", "@@", "id", "type", "@ID", "@Type", "@types", "@i", "@idx",
		" @id", "@id ", "@vocab/", "http://schema.org/name", "_:b0",
		1, 1.5, true, map[string]interface{}{"@id": "x"}, []interface{}{"@id"},
	} {
		assert.False(t, IsKeyword(v), "%#v", v)
	}
}

var keywordSink bool

// A realistic mix: mostly IRIs and terms, some keywords.
var keywordBenchKeys = []interface{}{
	"@id", "@type", "http://schema.org/name", "name", "@value", "@context",
	"https://schema.org/geo", "@language", "@graph", "description", "@list", 42,
}

func BenchmarkIsKeyword(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, k := range keywordBenchKeys {
			keywordSink = IsKeyword(k)
		}
	}
}
