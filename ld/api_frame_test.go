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
	"testing"

	. "github.com/piprate/json-gold/ld"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFrameFlag(t *testing.T) {
	assert.Equal(t, true, GetFrameFlag(
		map[string]interface{}{"test": []interface{}{true, false}},
		"test",
		false,
	),
	)

	assert.Equal(t, true, GetFrameFlag(
		map[string]interface{}{
			"test": map[string]interface{}{
				"@value": true,
			},
		},
		"test",
		false,
	),
	)

	assert.Equal(t, true, GetFrameFlag(
		map[string]interface{}{
			"test": map[string]interface{}{
				"@value": "true",
			},
		},
		"test",
		false,
	),
	)

	assert.Equal(t, false, GetFrameFlag(
		map[string]interface{}{
			"test": map[string]interface{}{
				"@value": "false",
			},
		},
		"test",
		true,
	),
	)

	assert.Equal(t, true, GetFrameFlag(
		map[string]interface{}{"test": true},
		"test",
		false,
	),
	)

	assert.Equal(t, false, GetFrameFlag(
		map[string]interface{}{"test": "not_boolean"},
		"test",
		false,
	),
	)
}

func TestJsonLdApi_Frame(t *testing.T) {
	var input any
	err := json.Unmarshal([]byte(shaclValidationResult), &input)
	require.NoError(t, err)
	var frame any
	err = json.Unmarshal([]byte(shaclValidationResultFrame), &frame)
	require.NoError(t, err)
	framed, err := NewJsonLdProcessor().Frame(input, frame, nil)
	require.NoError(t, err)
	assert.Equal(t, []any{map[string]any{
		"sh:resultPath": "https://example.com/hasScrewable",
		"sh:sourceShape": map[string]any{
			"sh:or": map[string]any{
				"@list": []any{
					map[string]any{
						"sh:class": "https://example.com/Bolt",
						"type":     "sh:NodeShape",
					},
					map[string]any{
						"sh:class": "https://example.com/Screw",
						"type":     "sh:NodeShape",
					},
				},
			},
			"sh:path": "https://example.com/hasScrewable",
			"type":    "sh:PropertyShape",
		},
		"type": "sh:ValidationResult",
	}}, framed["@graph"])
}

const shaclValidationResult = `
[
  {
    "@id": "_:ccbca2cd103643858f1087647dd5399717617",
    "@type": [
      "http://www.w3.org/ns/shacl#ValidationResult"
    ],
    "http://www.w3.org/ns/shacl#resultPath": [
      {
        "@id": "https://example.com/hasScrewable"
      }
    ],
    "http://www.w3.org/ns/shacl#sourceShape": [
      {
        "@id": "_:node94090"
      }
    ]
  },
  {
    "@id": "_:node94090",
    "@type": [
      "http://www.w3.org/ns/shacl#PropertyShape"
    ],
    "http://www.w3.org/ns/shacl#or": [
      {
        "@list": [
          {
            "@id": "_:node94092"
          },
          {
            "@id": "_:node94094"
          }
        ]
      }
    ],
    "http://www.w3.org/ns/shacl#path": [
      {
        "@id": "https://example.com/hasScrewable"
      }
    ]
  },
  {
    "@id": "_:node94092",
    "@type": [
      "http://www.w3.org/ns/shacl#NodeShape"
    ],
    "http://www.w3.org/ns/shacl#class": [
      {
        "@id": "https://example.com/Bolt"
      }
    ]
  },
  {
    "@id": "_:node94094",
    "@type": [
      "http://www.w3.org/ns/shacl#NodeShape"
    ],
    "http://www.w3.org/ns/shacl#class": [
      {
        "@id": "https://example.com/Screw"
      }
    ]
  }
]`

const shaclValidationResultFrame = `{
  "@context": {
    "sh": "http://www.w3.org/ns/shacl#",
    "sh:resultPath": {
      "@id": "sh:resultPath",
      "@type": "@id"
    },
    "sh:path": {
      "@id": "sh:path",
      "@type": "@id"
    },
    "sh:class": {
      "@id": "sh:class",
      "@type": "@id"
    },
    "type": "@type",
    "value": "@value",
    "id": "@id"
  },
  "@type": "sh:ValidationResult"
}`
