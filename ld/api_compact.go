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

// Compact operation compacts the given input using the context
// according to the steps in the Compaction Algorithm:
//
// http://www.w3.org/TR/json-ld-api/#compaction-algorithm
//
// Returns the compacted JSON-LD object.
// Returns an error if there was an error during compaction.
func (api *JsonLdApi) Compact(activeCtx *Context, activeProperty string, element interface{},
	compactArrays bool) (interface{}, error) {
	// 2)
	if elementList, isList := element.([]interface{}); isList {
		// 2.1)
		result := make([]interface{}, 0)
		// 2.2)
		for _, item := range elementList {
			// 2.2.1)
			compactedItem, err := api.Compact(activeCtx, activeProperty, item, compactArrays)
			if err != nil {
				return nil, err
			}
			// 2.2.2)
			if compactedItem != nil {
				result = append(result, compactedItem)
			}
		}
		// 2.3)
		if compactArrays && len(result) == 1 && activeCtx.GetContainer(activeProperty) == "" {
			return result[0], nil
		}
		// 2.4)
		return result, nil
	}

	// 3)
	if elem, isMap := element.(map[string]interface{}); isMap {
		// 4
		_, containsValue := elem["@value"]
		_, containsID := elem["@id"]
		if containsValue || containsID {
			compactedValue := activeCtx.CompactValue(activeProperty, elem)
			_, isMap := compactedValue.(map[string]interface{})
			_, isList := compactedValue.([]interface{})
			if !(isMap || isList) {
				return compactedValue, nil
			}
		}
		// 5)
		insideReverse := activeProperty == "@reverse"

		// 6)
		result := make(map[string]interface{})
		// 7)
		for _, expandedProperty := range GetOrderedKeys(elem) {
			expandedValue := elem[expandedProperty]

			// 7.1)
			if expandedProperty == "@id" || expandedProperty == "@type" {
				var compactedValue interface{}

				// 7.1.1)
				if expandedValueStr, isString := expandedValue.(string); isString {
					compactedValue = activeCtx.CompactIri(expandedValueStr, nil, expandedProperty == "@type", false)
				} else { // 7.1.2)
					types := make([]interface{}, 0)
					// 7.1.2.2)
					for _, expandedTypeVal := range expandedValue.([]interface{}) {
						expandedType := expandedTypeVal.(string)
						types = append(types, activeCtx.CompactIri(expandedType, nil, true, false))
					}
					// 7.1.2.3)
					if len(types) == 1 {
						compactedValue = types[0]
					} else {
						compactedValue = types
					}
				}

				// 7.1.3)
				alias := activeCtx.CompactIri(expandedProperty, nil, true, false)
				// 7.1.4)
				result[alias] = compactedValue
				continue
			}

			// 7.2)
			if expandedProperty == "@reverse" {
				// 7.2.1)
				compactedObject, _ := api.Compact(activeCtx, "@reverse", expandedValue, compactArrays)
				compactedValue := compactedObject.(map[string]interface{})
				// 7.2.2)
				for _, property := range GetKeys(compactedValue) {
					value := compactedValue[property]
					// 7.2.2.1)
					if activeCtx.IsReverseProperty(property) {
						// 7.2.2.1.1)
						valueList, isList := value.([]interface{})
						if (activeCtx.GetContainer(property) == "@set" || !compactArrays) && !isList {
							result[property] = []interface{}{value}
						}
						// 7.2.2.1.2)
						if _, present := result[property]; !present {
							result[property] = value
						} else { // 7.2.2.1.3)
							propertyValueList, isPropertyList := result[property].([]interface{})
							if !isPropertyList {
								propertyValueList = []interface{}{result[property]}
							}
							if isList {
								propertyValueList = append(propertyValueList, valueList...)
							} else {
								propertyValueList = append(propertyValueList, value)
							}
							result[property] = propertyValueList
						}
						// 7.2.2.1.4)
						delete(compactedValue, property)
					}

				}
				// 7.2.3)
				if len(compactedValue) > 0 {
					// 7.2.3.1)
					alias := activeCtx.CompactIri("@reverse", nil, true, false)
					// 7.2.3.2)
					result[alias] = compactedValue
				}
				// 7.2.4)
				continue
			}
			// 7.3)
			if expandedProperty == "@index" && activeCtx.GetContainer(activeProperty) == "@index" {
				continue
			} else if expandedProperty == "@index" || expandedProperty == "@value" ||
				expandedProperty == "@language" { // 7.4)
				// 7.4.1)
				alias := activeCtx.CompactIri(expandedProperty, nil, true, false)
				// 7.4.2)
				result[alias] = expandedValue
				continue
			}

			// NOTE: expanded value must be an array due to expansion
			// algorithm.

			// 7.5)
			expandedValueList, isList := expandedValue.([]interface{})
			if isList && len(expandedValueList) == 0 {
				// 7.5.1)
				itemActiveProperty := activeCtx.CompactIri(expandedProperty, expandedValue, true, insideReverse)
				// 7.5.2)
				itemActivePropertyVal, present := result[itemActiveProperty]
				if !present {
					result[itemActiveProperty] = make([]interface{}, 0)
				} else {
					if _, isList := itemActivePropertyVal.([]interface{}); !isList {
						result[itemActiveProperty] = []interface{}{itemActivePropertyVal}
					}
				}
			}

			// 7.6)
			for _, expandedItem := range expandedValueList {
				// 7.6.1)
				itemActiveProperty := activeCtx.CompactIri(expandedProperty, expandedItem, true, insideReverse)
				// 7.6.2)
				container := activeCtx.GetContainer(itemActiveProperty)

				// get @list value if appropriate
				expandedItemMap, isMap := expandedItem.(map[string]interface{})
				list, containsList := expandedItemMap["@list"]
				isList := isMap && containsList

				// 7.6.3)
				var elementToCompact interface{}
				if isList {
					elementToCompact = list
				} else {
					elementToCompact = expandedItem
				}
				compactedItem, _ := api.Compact(activeCtx, itemActiveProperty, elementToCompact, compactArrays)

				// 7.6.4)
				if isList {
					// 7.6.4.1)

					if _, isCompactedList := compactedItem.([]interface{}); !isCompactedList {
						compactedItem = []interface{}{compactedItem}
					}
					// 7.6.4.2)
					if container != "@list" {
						// 7.6.4.2.1)
						wrapper := make(map[string]interface{})
						// TODO: SPEC: no mention of vocab = true
						wrapper[activeCtx.CompactIri("@list", nil, true, false)] = compactedItem
						compactedItem = wrapper

						// 7.6.4.2.2)
						if indexVal, containsIndex := expandedItemMap["@index"]; containsIndex {
							// TODO: SPEC: no mention of vocab = true
							wrapper[activeCtx.CompactIri("@index", nil, true, false)] = indexVal
						}
					} else if _, present := result[itemActiveProperty]; present { // 7.6.4.3)
						return nil, NewJsonLdError(CompactionToListOfLists,
							"There cannot be two list objects associated with an active property that has a container mapping")
					}
				}
				// 7.6.5)
				if container == "@language" || container == "@index" {
					// 7.6.5.1)

					var mapObject map[string]interface{}
					if v, present := result[itemActiveProperty]; present {
						mapObject = v.(map[string]interface{})
					} else {
						mapObject = make(map[string]interface{})
						result[itemActiveProperty] = mapObject
					}

					// 7.6.5.2)
					compactedItemMap, isMap := compactedItem.(map[string]interface{})
					compactedItemValue, containsValue := compactedItemMap["@value"]
					if container == "@language" && isMap && containsValue {
						compactedItem = compactedItemValue
					}

					// 7.6.5.3)
					mapKey := expandedItemMap[container].(string)
					// 7.6.5.4)
					mapValue, hasMapKey := mapObject[mapKey]
					if !hasMapKey {
						mapObject[mapKey] = compactedItem
					} else {
						mapValueList, isList := mapValue.([]interface{})
						var tmp []interface{}
						if !isList {
							tmp = []interface{}{mapValue}
						} else {
							tmp = mapValueList
						}
						tmp = append(tmp, compactedItem)
						mapObject[mapKey] = tmp
					}
				} else { // 7.6.6)
					// 7.6.6.1)
					_, isList := compactedItem.([]interface{})
					check := (!compactArrays || container == "@set" || container == "@list" ||
						expandedProperty == "@list" || expandedProperty == "@graph") && !isList
					if check {
						compactedItem = []interface{}{compactedItem}
					}
					// 7.6.6.2)
					itemActivePropertyVal, present := result[itemActiveProperty]
					if !present {
						result[itemActiveProperty] = compactedItem
					} else {
						itemActivePropertyValueList, isList := itemActivePropertyVal.([]interface{})
						if !isList {
							itemActivePropertyValueList = []interface{}{itemActivePropertyVal}
							result[itemActiveProperty] = itemActivePropertyValueList
						}
						compactedItemList, isList := compactedItem.([]interface{})
						if isList {
							itemActivePropertyValueList = append(itemActivePropertyValueList, compactedItemList...)
						} else {
							itemActivePropertyValueList = append(itemActivePropertyValueList, compactedItem)
						}
						result[itemActiveProperty] = itemActivePropertyValueList
					}
				}
			}
		}
		// 8)
		return result, nil
	}
	// 2)
	return element, nil
}
