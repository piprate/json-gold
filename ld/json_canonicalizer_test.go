// Copyright SecureKey Technologies Inc. All Rights Reserved.
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
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONCanonicalizerFunc_Transform(t *testing.T) {
	doc := `{
  "@context": {
    "ex": "http://example.org/vocab#"
  },
  "@id": "http://example.org/test#example",
  "@type": "ex:Foo",
  "ex:embed": {
    "@type": "ex:Bar",
	"ex:foo": "bar",
    "ex:jsonfield": [
      {
        "@type": "@json",
        "@value": {
          "1": {"f": {"f": "hi","F": 5} ," ": 56.0},
          "10": { },
          "": "empty",
          "a": { },
          "111": [ {"e": "yes","E": "no" } ],
          "A": { }
        }
      }
    ]
  }
}`

	var docMap map[string]interface{}

	err := json.Unmarshal([]byte(doc), &docMap)
	require.NoError(t, err)

	proc := NewJsonLdProcessor()
	ldOptions := NewJsonLdOptions("")
	ldOptions.Algorithm = AlgorithmURDNA2015
	ldOptions.Format = "application/n-quads"

	view, err := proc.Normalize(docMap, ldOptions)
	require.NoError(t, err)

	viewStr := view.(string)

	for _, s := range [...]string{"JSON literals not supported", "JSON Marshal error", "JSON Canonicalization error"} {
		if strings.Contains(viewStr, s) {
			t.Fatal("expected JSON-LD normalization to pass")
		}
	}
}
