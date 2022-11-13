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

func TestGetCanonicalDouble(t *testing.T) {
	assert.Equal(t, "5.3E0", GetCanonicalDouble(5.3))
}
