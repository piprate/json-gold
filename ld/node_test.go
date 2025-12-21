// Copyright 2025 Siemens AG
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
	"github.com/stretchr/testify/require"
)

func TestRdfToObject_NativeTypes(t *testing.T) {
	testCases := []struct {
		name     string
		literal  Literal
		expected map[string]any
	}{
		{
			name:     "Boolean true",
			literal:  Literal{Value: "true", Datatype: XSDBoolean},
			expected: map[string]any{"@value": true},
		},
		{
			name:     "Boolean false",
			literal:  Literal{Value: "false", Datatype: XSDBoolean},
			expected: map[string]any{"@value": false},
		},
		{
			name:     "Boolean True",
			literal:  Literal{Value: "True", Datatype: XSDBoolean},
			expected: map[string]any{"@value": "True", "@type": XSDBoolean},
		},
		{
			name:     "Float",
			literal:  Literal{Value: "3.141", Datatype: XSDFloat},
			expected: map[string]any{"@value": float64(3.141)},
		},
		{
			name:     "Double",
			literal:  Literal{Value: "2.71828", Datatype: XSDDouble},
			expected: map[string]any{"@value": float64(2.71828)},
		},
		{
			name:     "Integer",
			literal:  Literal{Value: "42", Datatype: XSDInteger},
			expected: map[string]any{"@value": int64(42)},
		},
		{
			name:     "String without @type",
			literal:  Literal{Value: "hello world", Datatype: XSDString},
			expected: map[string]any{"@value": "hello world"},
		},
		{
			name:     "Decimal",
			literal:  Literal{Value: "3.141", Datatype: XSDDecimal},
			expected: map[string]any{"@value": "3.141", "@type": XSDDecimal},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			converted, err := RdfToObject(tc.literal, true)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, converted)
		})
	}
}
