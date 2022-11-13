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
	"log"
)

func main() {
	proc := ld.NewJsonLdProcessor()
	options := ld.NewJsonLdOptions("")

	// expanding remote document

	expanded, err := proc.Expand("http://json-ld.org/test-suite/tests/expand-0002-in.jsonld", options)
	if err != nil {
		log.Println("Error when expanding JSON-LD document:", err)
		return
	}

	ld.PrintDocument("JSON-LD expansion succeeded", expanded)

	// expanding in-memory document

	doc := map[string]interface{}{
		"@context":  "http://schema.org/",
		"@type":     "Person",
		"name":      "Jane Doe",
		"jobTitle":  "Professor",
		"telephone": "(425) 123-4567",
		"url":       "http://www.janedoe.com",
	}

	expanded, err = proc.Expand(doc, options)
	if err != nil {
		panic(err)
	}

	ld.PrintDocument("JSON-LD expansion succeeded", expanded)
}
