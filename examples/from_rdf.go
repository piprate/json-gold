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

	triples := `
		<http://example.com/Subj1> <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <http://example.com/Type> .
		<http://example.com/Subj1> <http://example.com/prop1> <http://example.com/Obj1> .
		<http://example.com/Subj1> <http://example.com/prop2> "Plain" .
		<http://example.com/Subj1> <http://example.com/prop2> "2012-05-12"^^<http://www.w3.org/2001/XMLSchema#date> .
		<http://example.com/Subj1> <http://example.com/prop2> "English"@en .
	`

	doc, err := proc.FromRDF(triples, options)
	if err != nil {
		panic(err)
	}

	ld.PrintDocument("JSON-LD output", doc)
}
