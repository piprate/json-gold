// Copyright 2026 Siemens AG
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
)

var benchInput string = `
<http://example.org/subject> <http://example.org/predicate> <http://example.org/object> .

_:b0 <http://example.org/predicate> <http://example.org/object> .

<http://example.org/subject> <http://example.org/predicate> _:b0 .

<http://example.org/subject> <http://example.org/predicate> "literal value" .

<http://example.org/subject> <http://example.org/predicate> "Hello World"@en .

<http://example.org/subject> <http://example.org/predicate> "42"^^<http://www.w3.org/2001/XMLSchema#integer> .

<http://example.org/subject> <http://example.org/predicate> "Line 1\\nLine 2\\tTabbed" .

<http://example.org/subject> <http://example.org/predicate> <http://example.org/object> <http://example.org/graph> .

<http://example.org/subject> <http://example.org/predicate> <http://example.org/object> _:graph .

<http://example.org/subject> <http://example.org/predicate> "Quote: \"nested\" and backslash: \\\\" .
`

func BenchmarkParseNQuadsFrom(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseNQuads(benchInput)
		if err != nil {
			b.Fatalf("failed to parse benchInput: %s", err)
		}
	}
}
