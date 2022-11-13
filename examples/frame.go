//go:build ignore
// +build ignore

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

package main

import (
	"github.com/piprate/json-gold/ld"
)

func main() {
	proc := ld.NewJsonLdProcessor()
	options := ld.NewJsonLdOptions("")

	doc := map[string]interface{}{
		"@context": map[string]interface{}{
			"dc":          "http://purl.org/dc/elements/1.1/",
			"ex":          "http://example.org/vocab#",
			"ex:contains": map[string]interface{}{"@type": "@id"},
		},
		"@graph": []interface{}{
			map[string]interface{}{
				"@id":         "http://example.org/test/#library",
				"@type":       "ex:Library",
				"ex:contains": "http://example.org/test#book",
			},
			map[string]interface{}{
				"@id":            "http://example.org/test#book",
				"@type":          "ex:Book",
				"dc:contributor": "Writer",
				"dc:title":       "My Book",
				"ex:contains":    "http://example.org/test#chapter",
			},
			map[string]interface{}{
				"@id":            "http://example.org/test#chapter",
				"@type":          "ex:Chapter",
				"dc:description": "Fun",
				"dc:title":       "Chapter One",
			},
		},
	}

	frame := map[string]interface{}{
		"@context": map[string]interface{}{
			"dc": "http://purl.org/dc/elements/1.1/",
			"ex": "http://example.org/vocab#",
		},
		"@type": "ex:Library",
		"ex:contains": map[string]interface{}{
			"@type": "ex:Book",
			"ex:contains": map[string]interface{}{
				"@type": "ex:Chapter",
			},
		},
	}

	framedDoc, err := proc.Frame(doc, frame, options)
	if err != nil {
		panic(err)
	}

	ld.PrintDocument("JSON-LD framing succeeded", framedDoc)
}
