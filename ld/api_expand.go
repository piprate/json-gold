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
	"fmt"
	"strings"
)

// Expand operation expands the given input according to the steps in the Expansion algorithm:
//
// http://www.w3.org/TR/json-ld-api/#expansion-algorithm
//
// Returns the expanded JSON-LD object.
// Returns an error if there was an error during expansion.
func (api *JsonLdApi) Expand(activeCtx *Context, activeProperty string, element interface{}, opts *JsonLdOptions) (interface{}, error) {
	frameExpansion := opts.ProcessingMode == JsonLd_1_1_Frame
	// 1)
	if element == nil {
		return nil, nil
	}

	// 3)
	switch elem := element.(type) {
	case []interface{}:
		// 3.1)
		var resultList = make([]interface{}, 0)
		// 3.2)
		for _, item := range elem {
			// 3.2.1)
			v, err := api.Expand(activeCtx, activeProperty, item, opts)
			if err != nil {
				return nil, err
			}
			// 3.2.2)
			if activeProperty == "@list" || activeCtx.GetContainer(activeProperty) == "@list" {
				_, isList := v.([]interface{})
				vMap, isMap := v.(map[string]interface{})
				_, mapContainsList := vMap["@list"]
				if isList || (isMap && mapContainsList) {
					return nil, NewJsonLdError(ListOfLists, "lists of lists are not permitted.")
				}
			}
			if v != nil {
				// 3.2.3)
				vList, isList := v.([]interface{})
				if isList {
					for _, vElement := range vList {
						resultList = append(resultList, vElement)
					}
				} else {
					resultList = append(resultList, v)
				}
			}
		}
		// 3.3)
		return resultList, nil

	case map[string]interface{}:

		// 4)
		// 5)
		if ctx, hasContext := elem["@context"]; hasContext {
			newCtx, err := activeCtx.Parse(ctx)
			if err != nil {
				return nil, err
			}
			activeCtx = newCtx
		}
		// 6)
		resultMap := make(map[string]interface{})
		// 7)
		for _, key := range GetOrderedKeys(elem) {
			value := elem[key]
			// 7.1)
			if key == "@context" {
				continue
			}
			// 7.2)
			expandedProperty, err := activeCtx.ExpandIri(key, false, true, nil, nil)
			if err != nil {
				return nil, err
			}
			var expandedValue interface{}
			// 7.3)
			if expandedProperty == "" || (!strings.Contains(expandedProperty, ":") && !IsKeyword(expandedProperty)) {
				continue
			}
			// 7.4)
			if IsKeyword(expandedProperty) {
				// 7.4.1)
				if activeProperty == "@reverse" {
					return nil, NewJsonLdError(InvalidReversePropertyMap,
						"a keyword cannot be used as a @reverse propery")
				}
				// 7.4.2)
				if _, containsKey := resultMap[expandedProperty]; containsKey {
					return nil, NewJsonLdError(CollidingKeywords, expandedProperty+" already exists in result")
				}
				// 7.4.3)
				if expandedProperty == "@id" {
					valueStr, isString := value.(string)
					if isString {
						expandedValue, err = activeCtx.ExpandIri(valueStr, true, false, nil, nil)
						if err != nil {
							return nil, err
						}
					} else if frameExpansion {
						if valueMap, isMap := value.(map[string]interface{}); isMap {
							if len(valueMap) != 0 {
								return nil, NewJsonLdError(InvalidIDValue, "@id value must be a an empty object for framing")
							}
							expandedValue = value
						} else if valueList, isList := value.([]interface{}); isList {
							expandedValue := make([]string, 0)
							for _, v := range valueList {
								vString, isString := v.(string)
								if !isString {
									return nil, NewJsonLdError(InvalidIDValue, "@id value must be a string, an array of strings or an empty dictionary")
								}
								v, err := activeCtx.ExpandIri(vString, true, true, nil, nil)
								if err != nil {
									return nil, err
								}
								expandedValue = append(expandedValue, v)
							}
						} else {
							return nil, NewJsonLdError(InvalidIDValue, "value of @id must be a string, an array of strings or an empty dictionary")
						}
					} else {
						return nil, NewJsonLdError(InvalidIDValue, "value of @id must be a string")
					}
				} else if expandedProperty == "@type" { // 7.4.4)
					switch v := value.(type) {
					case []interface{}:
						var expandedValueList []interface{}
						for _, listElem := range v {
							listElemStr, isString := listElem.(string)
							if !isString {
								return nil, NewJsonLdError(InvalidTypeValue,
									"@type value must be a string or array of strings")
							}
							newVal, err := activeCtx.ExpandIri(listElemStr, true, true, nil, nil)
							if err != nil {
								return nil, err
							}
							expandedValueList = append(expandedValueList, newVal)
						}
						expandedValue = expandedValueList
					case string:
						expandedValue, err = activeCtx.ExpandIri(v, true, true, nil, nil)
						if err != nil {
							return nil, err
						}
					case map[string]interface{}:
						if len(v) != 0 {
							return nil, NewJsonLdError(InvalidTypeValue,
								"@type value must be a an empty object for framing")
						}
						expandedValue = value
					default:
						return nil, NewJsonLdError(InvalidTypeValue, "@type value must be a string or array of strings")
					}
				} else if expandedProperty == "@graph" { // 7.4.5)
					expandedValue, _ = api.Expand(activeCtx, "@graph", value, opts)
				} else if expandedProperty == "@value" { // 7.4.6)
					_, isMap := value.(map[string]interface{})
					_, isList := value.([]interface{})
					if value != nil && (isMap || isList) {
						return nil, NewJsonLdError(InvalidValueObjectValue, "value of "+
							expandedProperty+" must be a scalar or null")
					}
					expandedValue = value
					if expandedValue == nil {
						resultMap["@value"] = nil
						continue
					}
				} else if expandedProperty == "@language" { // 7.4.7)
					valueStr, isString := value.(string)
					if !isString {
						return nil, NewJsonLdError(InvalidLanguageTaggedString, "Value of "+
							expandedProperty+" must be a string")
					}
					expandedValue = strings.ToLower(valueStr)
				} else if expandedProperty == "@index" { // 7.4.8)
					_, isString := value.(string)
					if !isString {
						return nil, NewJsonLdError(InvalidIndexValue, "Value of "+
							expandedProperty+" must be a string")
					}
					expandedValue = value
				} else if expandedProperty == "@list" { // 7.4.9)
					// 7.4.9.1)
					if activeProperty == "" || activeProperty == "@graph" {
						continue
					}
					// 7.4.9.2)
					expandedValue, _ = api.Expand(activeCtx, activeProperty, value, opts)

					// NOTE: step not in the spec yet
					expandedValueList, isList := expandedValue.([]interface{})
					if !isList {
						expandedValueList = []interface{}{expandedValue}
						expandedValue = expandedValueList
					}

					// 7.4.9.3)
					for _, o := range expandedValueList {
						oMap, isMap := o.(map[string]interface{})
						if _, containsList := oMap["@list"]; isMap && containsList {
							return nil, NewJsonLdError(ListOfLists, "A list may not contain another list")
						}
					}
				} else if expandedProperty == "@set" { // 7.4.10)
					expandedValue, _ = api.Expand(activeCtx, activeProperty, value, opts)
				} else if expandedProperty == "@reverse" { // 7.4.11)
					_, isMap := value.(map[string]interface{})
					if !isMap {
						return nil, NewJsonLdError(InvalidReverseValue, "@reverse value must be an object")
					}
					// 7.4.11.1)
					expandedValue, err = api.Expand(activeCtx, "@reverse", value, opts)
					if err != nil {
						return nil, err
					}

					// NOTE: algorithm assumes the result is a map
					// 7.4.11.2)
					reverseValue, containsReverse := expandedValue.(map[string]interface{})["@reverse"]
					if containsReverse {
						for property, item := range reverseValue.(map[string]interface{}) {
							// 7.4.11.2.1)
							var propertyList []interface{}
							if propertyValue, containsProperty := resultMap[property]; containsProperty {
								propertyList = propertyValue.([]interface{})
							} else {
								propertyList = make([]interface{}, 0)
								resultMap[property] = propertyList
							}
							// 7.4.11.2.2)
							if itemList, isList := item.([]interface{}); isList {
								propertyList = append(propertyList, itemList...)
							} else {
								propertyList = append(propertyList, item)
							}
							resultMap[property] = propertyList
						}
					}
					// 7.4.11.3)
					expandedValueMap := expandedValue.(map[string]interface{})
					var maxSize int
					if containsReverse {
						maxSize = 1
					} else {
						maxSize = 0
					}
					if len(expandedValueMap) > maxSize {
						var reverseMap map[string]interface{}
						if reverseValue, containsReverse := resultMap["@reverse"]; containsReverse {
							// 7.4.11.3.2)
							reverseMap = reverseValue.(map[string]interface{})
						} else {
							// 7.4.11.3.1)
							reverseMap = make(map[string]interface{})
							resultMap["@reverse"] = reverseMap
						}

						// 7.4.11.3.3)
						for property, propertyValue := range expandedValueMap {
							if property == "@reverse" {
								continue
							}
							// 7.4.11.3.3.1)
							items := propertyValue.([]interface{})
							for _, item := range items {
								// 7.4.11.3.3.1.1)
								itemMap := item.(map[string]interface{})
								_, containsValue := itemMap["@value"]
								_, containsList := itemMap["@list"]
								if containsValue || containsList {
									return nil, NewJsonLdError(InvalidReversePropertyValue, nil)
								}
								// 7.4.11.3.3.1.2)
								var propertyValueList []interface{}
								propertyValue, containsProperty := reverseMap[property]
								if containsProperty {
									propertyValueList = propertyValue.([]interface{})
								} else {
									propertyValueList = make([]interface{}, 0)
									reverseMap[property] = propertyValueList
								}
								// 7.4.11.3.3.1.3)
								reverseMap[property] = append(propertyValueList, item)
							}
						}
					}
					// 7.4.11.4)
					continue
				} else if expandedProperty == "@explicit" || // TODO: SPEC no mention of @explicit etc in spec
					expandedProperty == "@default" ||
					expandedProperty == "@embed" ||
					expandedProperty == "@embedChildren" ||
					expandedProperty == "@omitDefault" {
					expandedValue, _ = api.Expand(activeCtx, expandedProperty, value, opts)
				}
				// 7.4.12)
				if expandedValue != nil {
					resultMap[expandedProperty] = expandedValue
				}
				// 7.4.13)
				continue
			} else {
				valueMap, isMap := value.(map[string]interface{})
				// 7.5
				if activeCtx.GetContainer(key) == "@language" && isMap {
					// 7.5.1)
					var expandedValueList []interface{}
					// 7.5.2)
					for _, language := range GetOrderedKeys(valueMap) {
						languageValue := valueMap[language]
						// 7.5.2.1)
						languageList, isList := languageValue.([]interface{})
						if !isList {
							languageList = []interface{}{languageValue}
						}
						// 7.5.2.2)
						for _, item := range languageList {
							// 7.5.2.2.1)
							if _, isString := item.(string); !isString {
								return nil, NewJsonLdError(InvalidLanguageMapValue, "Expected "+
									fmt.Sprintf("%v", item)+" to be a string")
							}
							// 7.5.2.2.2)
							expandedValueList = append(expandedValueList, map[string]interface{}{
								"@value":    item,
								"@language": strings.ToLower(language),
							})
						}
					}
					expandedValue = expandedValueList
				} else if activeCtx.GetContainer(key) == "@index" && isMap { // 7.6)
					// 7.6.1)
					var expandedValueList []interface{}
					// 7.6.2)
					for _, index := range GetOrderedKeys(valueMap) {
						indexValue := valueMap[index]
						// 7.6.2.1)
						indexValueList, isList := indexValue.([]interface{})
						if !isList {
							indexValueList = []interface{}{indexValue}
						}
						// 7.6.2.2)
						indexValue, _ = api.Expand(activeCtx, key, indexValueList, opts)
						// 7.6.2.3)
						for _, itemValue := range indexValue.([]interface{}) {
							item := itemValue.(map[string]interface{})
							// 7.6.2.3.1)
							if _, containsKey := item["@index"]; !containsKey {
								item["@index"] = index
							}
							// 7.6.2.3.2)
							expandedValueList = append(expandedValueList, item)
						}
					}
					expandedValue = expandedValueList
				} else {
					// 7.7)
					expandedValue, err = api.Expand(activeCtx, key, value, opts)
					if err != nil {
						return nil, err
					}
				}
			}

			// 7.8)
			if expandedValue == nil {
				continue
			}
			// 7.9)
			if activeCtx.GetContainer(key) == "@list" {
				expandedValueMap, isMap := expandedValue.(map[string]interface{})
				_, containsList := expandedValueMap["@list"]
				if !isMap || !containsList {
					newExpandedValue := make(map[string]interface{}, 1)
					_, isList := expandedValue.([]interface{})
					if !isList {
						newExpandedValue["@list"] = []interface{}{expandedValue}
					} else {
						newExpandedValue["@list"] = expandedValue
					}
					expandedValue = newExpandedValue
				}
			}
			// 7.10)
			if activeCtx.IsReverseProperty(key) {
				var reverseMap map[string]interface{}
				if reverseValue, containsReverse := resultMap["@reverse"]; containsReverse {
					// 7.10.2)
					reverseMap = reverseValue.(map[string]interface{})
				} else {
					// 7.10.1)
					reverseMap = make(map[string]interface{})
					resultMap["@reverse"] = reverseMap
				}

				// 7.10.3)
				expandedValueList, isList := expandedValue.([]interface{})
				if !isList {
					expandedValueList = []interface{}{expandedValue}
					expandedValue = expandedValueList
				}
				// 7.10.4)
				for _, item := range expandedValueList {

					// 7.10.4.2)
					var expandedPropertyList []interface{}
					expandedPropertyValue, containsExpandedProperty := reverseMap[expandedProperty]
					if containsExpandedProperty {
						expandedPropertyList = expandedPropertyValue.([]interface{})
					} else {
						expandedPropertyList = make([]interface{}, 0)
					}

					switch v := item.(type) {
					case map[string]interface{}:
						// 7.10.4.1)
						_, containsValue := v["@value"]
						_, containsList := v["@list"]
						if containsValue || containsList {
							return nil, NewJsonLdError(InvalidReversePropertyValue, nil)
						}
						expandedPropertyList = append(expandedPropertyList, v)
					case []interface{}:
						// 7.10.4.3)
						expandedPropertyList = append(expandedPropertyList, v...)
					default:
						expandedPropertyList = append(expandedPropertyList, v)
					}
					reverseMap[expandedProperty] = expandedPropertyList
				}
			} else { // 7.11)
				// 7.11.1)
				var expandedPropertyList []interface{}
				expandedPropertyValue, containsExpandedProperty := resultMap[expandedProperty]
				if containsExpandedProperty {
					expandedPropertyList = expandedPropertyValue.([]interface{})
				} else {
					expandedPropertyList = make([]interface{}, 0)
					resultMap[expandedProperty] = expandedPropertyList
				}
				// 7.11.2)
				if expandedValueList, isList := expandedValue.([]interface{}); isList {
					expandedPropertyList = append(expandedPropertyList, expandedValueList...)
				} else {
					expandedPropertyList = append(expandedPropertyList, expandedValue)
				}
				resultMap[expandedProperty] = expandedPropertyList
			}
		}
		// 8)
		if rval, hasValue := resultMap["@value"]; hasValue {
			// 8.1)
			allowedKeys := map[string]interface{}{
				"@value":    nil,
				"@index":    nil,
				"@language": nil,
				"@type":     nil,
			}
			hasDisallowedKeys := false
			for key := range resultMap {
				if _, containsKey := allowedKeys[key]; !containsKey {
					hasDisallowedKeys = true
					break
				}
			}
			_, hasLanguage := resultMap["@language"]
			typeValue, hasType := resultMap["@type"]
			if hasDisallowedKeys || (hasLanguage && hasType) {
				return nil, NewJsonLdError(InvalidValueObject, "value object has unknown keys")
			}
			// 8.2)
			if rval == nil {
				// nothing else is possible with result if we set it to
				// null, so simply return it
				return nil, nil
			}
			// 8.3)
			if _, isString := rval.(string); !isString && hasLanguage {
				return nil, NewJsonLdError(InvalidLanguageTaggedValue,
					"when @language is used, @value must be a string")
			} else if hasType { // 8.4)
				// TODO: is this enough for "is an IRI"
				typeStr, isString := typeValue.(string)
				if !isString || strings.HasPrefix(typeStr, "_:") || !strings.Contains(typeStr, ":") {
					return nil, NewJsonLdError(InvalidTypedValue, "value of @type must be an IRI")
				}
			}
		} else if rtype, hasType := resultMap["@type"]; hasType { // 9)
			if _, isList := rtype.([]interface{}); !isList {
				resultMap["@type"] = []interface{}{rtype}
			}
		} else {
			// 10)
			rset, hasSet := resultMap["@set"]
			_, hasList := resultMap["@list"]
			if hasSet || hasList {
				// 10.1)
				maxSize := 1
				if _, hasIndex := resultMap["@index"]; hasIndex {
					maxSize = 2
				}
				if len(resultMap) > maxSize {
					return nil, NewJsonLdError(InvalidSetOrListObject,
						"@set or @list may only contain @index")
				}
				// 10.2)
				if hasSet {
					// result becomes an array here, thus the remaining checks
					// will never be true from here on
					// so simply return the value rather than have to make
					// result an object and cast it with every
					// other use in the function.
					return rset, nil
				}
			}
		}
		var result interface{} = resultMap
		// 11)
		if _, hasLanguage := resultMap["@language"]; hasLanguage && len(resultMap) == 1 {
			resultMap = nil
			result = nil
		}
		// 12)
		if activeProperty == "" || activeProperty == "@graph" {
			// 12.1)
			_, hasValue := resultMap["@value"]
			_, hasList := resultMap["@list"]
			_, hasID := resultMap["@id"]
			if resultMap != nil && (len(resultMap) == 0 || hasValue || hasList) {
				resultMap = nil
				result = nil
			} else if resultMap != nil && !frameExpansion && hasID && len(resultMap) == 1 { // 12.2)
				resultMap = nil
				result = nil
			}
		}
		// 13)
		return result, nil
	default:
		// 2) If element is a scalar
		// 2.1)
		if activeProperty == "" || activeProperty == "@graph" {
			return nil, nil
		}
		return activeCtx.ExpandValue(activeProperty, element)
	}
}
