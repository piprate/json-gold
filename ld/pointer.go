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
	"strconv"
	"strings"
)

// ElementHandler is called during expansion with the JSON Pointer of a node
// object in the source document and the result of expanding it.
//
// The pointer follows RFC 6901 and is relative to the document handed to the
// processor: "" is the document itself, "/@graph/2" the third entry of its
// @graph, "/@graph/2/name" the "name" entry of that node.
//
// It is called for node objects only, and only for those expansion keeps.
// Value objects and list objects are not reported — they carry no identity to
// relate anything to — and neither is a node object that expansion discards,
// such as one consisting of nothing but an @id. What a handler sees is what
// came out.
//
// The handler is a way to relate expansion output back to the input document
// without the processor having to know why anyone would want that. Locating a
// pointer in the source text — as a line, a column or a byte offset — is a
// separate concern and deliberately not one this library takes on.
//
// The handler must not modify the value it is given. It is called during
// expansion, so a long-running handler slows expansion down.
type ElementHandler func(pointer string, expanded interface{})

// jsonPointerChild extends a pointer with one token, or returns "" when no
// handler is installed.
//
// The check lives here so that a build with no handler does no pointer work at
// all: no concatenation, no escaping, no allocation.
func jsonPointerChild(opts *JsonLdOptions, parent string, token string) string {
	if opts == nil || opts.ExpandedElementHandler == nil {
		return ""
	}
	return parent + "/" + escapeJSONPointerToken(token)
}

// jsonPointerIndex extends a pointer with an array index, or returns "" when no
// handler is installed.
//
// It exists separately from jsonPointerChild because the index has to be
// formatted, and an argument is evaluated before the call it is passed to —
// so formatting it at the call site would cost an allocation per element even
// with no handler installed, which is the one thing this must not do. An index
// also never needs escaping.
func jsonPointerIndex(opts *JsonLdOptions, parent string, index int) string {
	if opts == nil || opts.ExpandedElementHandler == nil {
		return ""
	}
	return parent + "/" + strconv.Itoa(index)
}

// isNodeObject reports whether an expanded map is a node object rather than a
// value or list object.
//
// After expansion the distinction is exactly this: @value marks a value object
// and @list a list object, and neither can carry an @id. Everything else that
// survives expansion is a node object, a graph object included.
func isNodeObject(expanded map[string]interface{}) bool {
	if _, isValue := expanded["@value"]; isValue {
		return false
	}
	_, isList := expanded["@list"]
	return !isList
}

// escapeJSONPointerToken applies the RFC 6901 escaping rules: "~" becomes "~0"
// and "/" becomes "~1". The order matters — escaping "/" first would turn the
// "~" it introduces into "~0".
func escapeJSONPointerToken(token string) string {
	if !strings.ContainsAny(token, "~/") {
		return token
	}
	return strings.ReplaceAll(strings.ReplaceAll(token, "~", "~0"), "/", "~1")
}
