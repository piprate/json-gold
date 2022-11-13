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
