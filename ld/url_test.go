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
