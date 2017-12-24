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
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// IsKeyword returns whether or not the given value is a keyword.
func IsKeyword(key interface{}) bool {
	if _, isString := key.(string); !isString {
		return false
	}
	return key == "@base" || key == "@context" || key == "@container" || key == "@default" ||
		key == "@embed" || key == "@explicit" || key == "@graph" || key == "@id" || key == "@index" ||
		key == "@language" || key == "@list" || key == "@omitDefault" || key == "@reverse" ||
		key == "@preserve" || key == "@set" || key == "@type" || key == "@value" || key == "@vocab"
}

// DeepCompare returns true if v1 equals v2.
func DeepCompare(v1 interface{}, v2 interface{}, listOrderMatters bool) bool {
	if v1 == nil {
		return v2 == nil
	} else if v2 == nil {
		return v1 == nil
	}

	m1, isMap1 := v1.(map[string]interface{})
	m2, isMap2 := v2.(map[string]interface{})
	l1, isList1 := v1.([]interface{})
	l2, isList2 := v2.([]interface{})
	if isMap1 && isMap2 {
		if len(m1) != len(m2) {
			return false
		}
		for _, key := range GetKeys(m1) {
			if val2, present := m2[key]; !present || !DeepCompare(m1[key], val2, listOrderMatters) {
				return false
			}
		}
		return true
	} else if isList1 && isList2 {
		if len(l1) != len(l2) {
			return false
		}
		// used to mark members of l2 that we have already matched to avoid
		// matching the same item twice for lists that have duplicates
		alreadyMatched := make([]bool, len(l2))
		for i := 0; i < len(l1); i++ {
			o1 := l1[i]
			gotMatch := false
			if listOrderMatters {
				gotMatch = DeepCompare(o1, l2[i], listOrderMatters)
			} else {
				for j := 0; j < len(l2); j++ {
					if !alreadyMatched[j] && DeepCompare(o1, l2[j], listOrderMatters) {
						alreadyMatched[j] = true
						gotMatch = true
						break
					}
				}
			}
			if !gotMatch {
				return false
			}
		}
		return true
	} else {
		if v1 != v2 {
			// perform additional checks. If the client code sets UseNumber() property
			// of json.Decoder to decode numbers (see https://golang.org/pkg/encoding/json/#Decoder.UseNumber ),
			// simple comparison will fail.
			return normalizeValue(v1) == normalizeValue(v2)
		} else {
			return true
		}
	}
}

// normalizeValue allows comparisons between json.Number and float/integer values.
func normalizeValue(v interface{}) string {
	floatVal, isFloat := v.(float64)

	if !isFloat {
		if number, isNumber := v.(json.Number); isNumber {
			var floatErr error
			floatVal, floatErr = number.Float64()
			if floatErr == nil {
				isFloat = true
			}
		}
	}
	if isFloat {
		return fmt.Sprintf("%f", floatVal)
	} else {
		return fmt.Sprintf("%s", v)
	}
}

func deepContains(values []interface{}, value interface{}) bool {
	for _, item := range values {
		if DeepCompare(item, value, false) {
			return true
		}
	}
	return false
}

// MergeValue adds a value to a subject. If the value is an array, all values in the array will be added.
func MergeValue(obj map[string]interface{}, key string, value interface{}) {
	if obj == nil {
		return
	}
	values, hasValues := obj[key].([]interface{})
	if !hasValues {
		values = make([]interface{}, 0)

	}
	valueMap, isMap := value.(map[string]interface{})
	_, valueContainsList := valueMap["@list"]
	if key == "@list" || (isMap && valueContainsList) || !deepContains(values, value) {
		values = append(values, value)
	}
	obj[key] = values
}

// IsAbsoluteIri returns true if the given value is an absolute IRI, false if not.
func IsAbsoluteIri(value string) bool {
	return strings.Contains(value, ":")
}

// IsNode returns true if the given value is a subject with properties.
//
// Note: A value is a subject if all of these hold true:
// 1. It is an Object.
// 2. It is not a @value, @set, or @list.
// 3. It has more than 1 key OR any existing key is not @id.
func IsNode(v interface{}) bool {
	vMap, isMap := v.(map[string]interface{})
	_, containsValue := vMap["@value"]
	_, containsSet := vMap["@set"]
	_, containsList := vMap["@list"]
	_, containsID := vMap["@id"]
	if isMap && !(containsValue || containsSet || containsList) {
		return len(vMap) > 1 || !containsID
	}
	return false
}

// IsNodeReference returns true if the given value is a subject reference.
func IsNodeReference(v interface{}) bool {
	// Note: A value is a subject reference if all of these hold true:
	// 1. It is an Object.
	// 2. It has a single key: @id.
	vMap, isMap := v.(map[string]interface{})
	_, containsID := vMap["@id"]
	return isMap && len(vMap) == 1 && containsID
}

// IsRelativeIri returns true if the given value is a relative IRI, false if not.
func IsRelativeIri(value string) bool {
	return !(IsKeyword(value) || IsAbsoluteIri(value))
}

// IsValue returns true if the given value is a JSON-LD value
func IsValue(v interface{}) bool {
	vMap, isMap := v.(map[string]interface{})
	_, containsValue := vMap["@value"]
	return isMap && containsValue
}

// IsBlankNode returns true if the given value is a blank node.
func IsBlankNodeValue(v interface{}) bool {
	// Note: A value is a blank node if all of these hold true:
	// 1. It is an Object.
	// 2. If it has an @id key its value begins with '_:'.
	// 3. It has no keys OR is not a @value, @set, or @list.
	vMap, isMap := v.(map[string]interface{})
	if isMap {
		id, containsID := vMap["@id"]
		if containsID {
			return strings.HasPrefix(id.(string), "_:")
		} else {
			_, containsValue := vMap["@value"]
			_, containsSet := vMap["@set"]
			_, containsList := vMap["@list"]
			return len(vMap) == 0 || !containsValue || containsSet || containsList
		}
	}
	return false
}

// CompareShortestLeast compares two strings first based on length and then lexicographically.
func CompareShortestLeast(a string, b string) bool {
	if len(a) < len(b) {
		return true
	} else if len(a) > len(b) {
		return false
	} else {
		return a < b
	}
}

// ShortestLeast is a struct which allows sorting using CompareShortestLeast function.
type ShortestLeast []string

func (s ShortestLeast) Len() int {
	return len(s)
}
func (s ShortestLeast) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ShortestLeast) Less(i, j int) bool {
	return CompareShortestLeast(s[i], s[j])
}

// RemovePreserve removes the @preserve keywords as the last step of the framing algorithm.
//
// ctx: the active context used to compact the input
// input: the framed, compacted output
// opts: the compaction options used
//
// Returns the resulting output.
func RemovePreserve(ctx *Context, input interface{}, opts *JsonLdOptions) (interface{}, error) {

	// recurse through arrays
	if inputList, isList := input.([]interface{}); isList {
		output := make([]interface{}, 0)
		for _, i := range inputList {
			result, _ := RemovePreserve(ctx, i, opts)
			// drop nulls from arrays
			if result != nil {
				output = append(output, result)
			}
		}
		input = output
	} else if inputMap, isMap := input.(map[string]interface{}); isMap {
		// remove @preserve
		if preserveVal, present := inputMap["@preserve"]; present {
			if preserveVal == "@null" {
				return nil, nil
			}
			return preserveVal, nil
		}

		// skip @values
		if _, hasValue := inputMap["@value"]; hasValue {
			return input, nil
		}

		// recurse through @lists
		if listVal, hasList := inputMap["@list"]; hasList {
			inputMap["@list"], _ = RemovePreserve(ctx, listVal, opts)
			return input, nil
		}

		// recurse through properties
		for prop, propVal := range inputMap {
			result, _ := RemovePreserve(ctx, propVal, opts)
			container := ctx.GetContainer(prop)
			resultList, isList := result.([]interface{})
			if opts.CompactArrays && isList && len(resultList) == 1 && container == "" {
				result = resultList[0]
			}
			inputMap[prop] = result
		}
	}

	return input, nil
}

// CompareValues compares two JSON-LD values for equality.
// Two JSON-LD values will be considered equal if:
//
// 1. They are both primitives of the same type and value.
// 2. They are both @values with the same @value, @type, and @language, OR
// 3. They both have @ids they are the same.
func CompareValues(v1 interface{}, v2 interface{}) bool {
	if v1 == v2 {
		return true
	}

	v1Map, isv1Map := v1.(map[string]interface{})
	v2Map, isv2Map := v2.(map[string]interface{})

	if IsValue(v1) && IsValue(v2) {
		if v1Map["@value"] == v2Map["@value"] &&
			v1Map["@type"] == v2Map["@type"] &&
			v1Map["@language"] == v2Map["@language"] &&
			v1Map["@index"] == v2Map["@index"] {
			return true
		}
	}

	id1, v1containsID := v1Map["@id"]
	id2, v2containsID := v2Map["@id"]
	if (isv1Map && v1containsID) && (isv2Map && v2containsID) && (id1 == id2) {
		return true
	}

	return false
}

// CloneDocument returns a cloned instance of the given document
func CloneDocument(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	m, isMap := value.(map[string]interface{})
	l, isList := value.([]interface{})

	if isMap {
		mClone := make(map[string]interface{}, len(m))
		for k, v := range m {
			mClone[k] = CloneDocument(v)
		}
		return mClone
	} else if isList {
		lClone := make([]interface{}, 0, len(l))
		for _, v := range l {
			lClone = append(lClone, CloneDocument(v))
		}
		return lClone
	} else {
		// This is a bit simplistic. Beware of string values, at least.
		return value
	}
}

// GetKeys returns all keys in the given object
func GetKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	return keys
}

// GetKeysString returns all keys in the given map[string]string
func GetKeysString(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	return keys
}

// GetOrderedKeys returns all keys in the given object as a sorted list
func GetOrderedKeys(m map[string]interface{}) []string {
	keys := GetKeys(m)
	sort.Strings(keys)

	return keys
}

// PrintDocument prints a JSON-LD document. This is useful for debugging.
func PrintDocument(msg string, doc interface{}) {
	b, _ := json.MarshalIndent(doc, "", "  ")
	if msg != "" {
		os.Stdout.WriteString(msg)
		os.Stdout.WriteString("\n")
	}
	os.Stdout.Write(b)
	os.Stdout.WriteString("\n")
}
