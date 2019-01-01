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

import "sort"

// Compact operation compacts the given input using the context
// according to the steps in the Compaction Algorithm:
//
// http://www.w3.org/TR/json-ld-api/#compaction-algorithm
//
// Returns the compacted JSON-LD object.
// Returns an error if there was an error during compaction.
func (api *JsonLdApi) Compact(activeCtx *Context, activeProperty string, element interface{},
	compactArrays bool) (interface{}, error) {

	if elementList, isList := element.([]interface{}); isList {
		result := make([]interface{}, 0)
		for _, item := range elementList {
			compactedItem, err := api.Compact(activeCtx, activeProperty, item, compactArrays)
			if err != nil {
				return nil, err
			}
			if compactedItem != nil {
				result = append(result, compactedItem)
			}
		}

		if compactArrays && len(result) == 1 && len(activeCtx.GetContainer(activeProperty)) == 0 {
			return result[0], nil
		}

		return result, nil
	}

	// use any scoped context on active_property
	td := activeCtx.GetTermDefinition(activeProperty)
	if ctx, hasCtx := td["@context"]; hasCtx {
		newCtx, err := activeCtx.Parse(ctx)
		if err != nil {
			return nil, err
		}
		activeCtx = newCtx
	}

	if elem, isMap := element.(map[string]interface{}); isMap {

		// do value compaction on @values and subject references
		if IsValue(elem) || IsSubjectReference(elem) {
			compactedValue := activeCtx.CompactValue(activeProperty, elem)
			return compactedValue, nil
		}

		insideReverse := activeProperty == "@reverse"

		result := make(map[string]interface{})

		// apply any context defined on an alias of @type
		// if key is @type and any compacted value is a term having a local
		// context, overlay that context
		if typeVal, hasType := elem["@type"]; hasType {
			// set scoped contexts from @type
			types := make([]string, 0)
			for _, t := range Arrayify(typeVal) {
				if typeStr, isString := t.(string); isString {
					compactedType := activeCtx.CompactIri(typeStr, nil, true, false)
					types = append(types, compactedType)
				}
			}
			// process in lexicographical order, see https://github.com/json-ld/json-ld.org/issues/616
			sort.Strings(types)
			for _, tt := range types {
				td := activeCtx.GetTermDefinition(tt)
				if ctx, hasCtx := td["@context"]; hasCtx {
					newCtx, err := activeCtx.Parse(ctx)
					if err != nil {
						return nil, err
					}
					activeCtx = newCtx
				}
			}
		}

		// recursively process element keys in order
		for _, expandedProperty := range GetOrderedKeys(elem) {
			expandedValue := elem[expandedProperty]

			if expandedProperty == "@id" || expandedProperty == "@type" {
				var compactedValue interface{}

				compactedValues := make([]interface{}, 0)

				for _, v := range Arrayify(expandedValue) {
					cv := activeCtx.CompactIri(v.(string), nil, expandedProperty == "@type", false)
					compactedValues = append(compactedValues, cv)
				}

				if len(compactedValues) == 1 {
					compactedValue = compactedValues[0]
				} else {
					compactedValue = compactedValues
				}

				alias := activeCtx.CompactIri(expandedProperty, nil, true, false)
				compValArray, isArray := compactedValue.([]interface{})
				AddValue(result, alias, compactedValue, isArray && len(compValArray) == 0, true)

				continue
			}

			if expandedProperty == "@reverse" {

				compactedObject, _ := api.Compact(activeCtx, "@reverse", expandedValue, compactArrays)
				compactedValue := compactedObject.(map[string]interface{})

				for _, property := range GetKeys(compactedValue) {
					value := compactedValue[property]

					if activeCtx.IsReverseProperty(property) {
						useArray := activeCtx.HasContainerMapping(property, "@set") || !compactArrays

						AddValue(result, property, value, useArray, true)

						delete(compactedValue, property)
					}

				}

				if len(compactedValue) > 0 {
					alias := activeCtx.CompactIri("@reverse", nil, false, false)
					AddValue(result, alias, compactedValue, false, true)
				}

				continue
			}

			if expandedProperty == "@preserve" {
				// compact using activeProperty
				compactedValue, _ := api.Compact(activeCtx, activeProperty, expandedValue, compactArrays)
				if cva, isArray := compactedValue.([]interface{}); !(isArray && len(cva) == 0) {
					AddValue(result, expandedProperty, compactedValue, false, true)
				}
				continue
			}

			if expandedProperty == "@index" && activeCtx.HasContainerMapping(activeProperty, "@index") {
				continue
			} else if expandedProperty == "@index" || expandedProperty == "@value" || expandedProperty == "@language" {
				alias := activeCtx.CompactIri(expandedProperty, nil, false, false)
				AddValue(result, alias, expandedValue, false, true)
				continue
			}

			// skip array processing for keywords that aren't @graph or @list
			if expandedProperty != "@graph" && expandedProperty != "@list" && IsKeyword(expandedProperty) {
				alias := activeCtx.CompactIri(expandedProperty, nil, false, false)
				AddValue(result, alias, expandedValue, false, true)
				continue
			}

			// NOTE: expanded value must be an array due to expansion algorithm.

			expandedValueList, isList := expandedValue.([]interface{})
			if isList && len(expandedValueList) == 0 {

				itemActiveProperty := activeCtx.CompactIri(expandedProperty, expandedValue, true, insideReverse)

				nestResult := result
				nestProperty, hasNest := activeCtx.GetTermDefinition(itemActiveProperty)["@nest"]
				if hasNest {
					if err := api.checkNestProperty(activeCtx, nestProperty.(string)); err != nil {
						return nil, err
					}
					if _, isMap := result[nestProperty.(string)].(map[string]interface{}); !isMap {
						result[nestProperty.(string)] = make(map[string]interface{})
					}
					nestResult = result[nestProperty.(string)].(map[string]interface{})
				}

				AddValue(nestResult, itemActiveProperty, make([]interface{}, 0), true, true)
			}

			for _, expandedItem := range expandedValueList {
				itemActiveProperty := activeCtx.CompactIri(expandedProperty, expandedItem, true, insideReverse)

				isListContainer := activeCtx.HasContainerMapping(itemActiveProperty, "@list")
				isGraphContainer := activeCtx.HasContainerMapping(itemActiveProperty, "@graph")
				isSetContainer := activeCtx.HasContainerMapping(itemActiveProperty, "@set")
				isLanguageContainer := activeCtx.HasContainerMapping(itemActiveProperty, "@language")
				isIndexContainer := activeCtx.HasContainerMapping(itemActiveProperty, "@index")
				isIdContainer := activeCtx.HasContainerMapping(itemActiveProperty, "@id")
				isTypeContainer := activeCtx.HasContainerMapping(itemActiveProperty, "@type")

				// if itemActiveProperty is a @nest property, add values to nestResult, otherwise result
				nestResult := result
				nestProperty, hasNest := activeCtx.GetTermDefinition(itemActiveProperty)["@nest"]
				if hasNest {
					if err := api.checkNestProperty(activeCtx, nestProperty.(string)); err != nil {
						return nil, err
					}
					if _, isMap := result[nestProperty.(string)].(map[string]interface{}); !isMap {
						result[nestProperty.(string)] = make(map[string]interface{})
					}
					nestResult = result[nestProperty.(string)].(map[string]interface{})
				}

				// get @list value if appropriate
				expandedItemMap, isMap := expandedItem.(map[string]interface{})
				isGraph := IsGraph(expandedItemMap)
				list, containsList := expandedItemMap["@list"]
				isList := isMap && containsList
				var inner interface{}

				if isList {
					inner = list
				} else if isGraph {
					inner = expandedItemMap["@graph"]
				}

				var elementToCompact interface{}
				if isList || isGraph {
					elementToCompact = inner
				} else {
					elementToCompact = expandedItem
				}

				// recursively compact expanded item
				compactedItem, err := api.Compact(activeCtx, itemActiveProperty, elementToCompact, compactArrays)
				if err != nil {
					return nil, err
				}

				if isList {
					compactedItem = Arrayify(compactedItem)

					if !isListContainer {

						listAlias := activeCtx.CompactIri("@list", nil, false, false)
						wrapper := map[string]interface{}{
							listAlias: compactedItem,
						}
						compactedItem = wrapper

						if indexVal, containsIndex := expandedItemMap["@index"]; containsIndex {
							indexAlias := activeCtx.CompactIri("@index", nil, false, false)
							wrapper[indexAlias] = indexVal
						}
					} else if _, present := nestResult[itemActiveProperty]; present { // 7.6.4.3)
						return nil, NewJsonLdError(CompactionToListOfLists,
							"There cannot be two list objects associated with an active property that has a container mapping")
					}
				}

				// graph object compaction
				if isGraph {
					asArray := !compactArrays || isSetContainer
					if isGraphContainer && (isIdContainer || isIndexContainer && IsSimpleGraph(expandedItemMap)) {
						var mapObject map[string]interface{}
						if v, present := nestResult[itemActiveProperty]; present {
							mapObject = v.(map[string]interface{})
						} else {
							mapObject = make(map[string]interface{})
							nestResult[itemActiveProperty] = mapObject
						}

						// index on @id or @index or alias of @none
						k := "@index"
						if isIdContainer {
							k = "@id"
						}
						mapKey := ""
						if v, found := expandedItemMap[k]; found {
							mapKey = v.(string)
						} else {
							mapKey = activeCtx.CompactIri("@none", nil, false, false)
						}

						// add compactedItem to map, using value of "@id" or a new blank node identifier
						AddValue(mapObject, mapKey, compactedItem, asArray, true)
					} else if isGraphContainer && IsSimpleGraph(expandedItemMap) {
						AddValue(nestResult, itemActiveProperty, compactedItem, asArray, true)
					} else {
						// wrap using @graph alias, remove array if only one item and compactArrays not set
						compactedItemArray, isArray := compactedItem.([]interface{})
						if isArray && len(compactedItemArray) == 1 && compactArrays {
							compactedItem = compactedItemArray[0]
						}
						graphAlias := activeCtx.CompactIri("@graph", nil, false, false)
						compactedItemMap := map[string]interface{}{
							graphAlias: compactedItem,
						}
						compactedItem = compactedItemMap

						// include @id from expanded graph, if any
						if val, hasID := expandedItemMap["@id"]; hasID {
							idAlias := activeCtx.CompactIri("@id", nil, false, false)
							compactedItemMap[idAlias] = val
						}

						// include @index from expanded graph, if any
						if val, hasIndex := expandedItemMap["@index"]; hasIndex {
							indexAlias := activeCtx.CompactIri("@index", nil, false, false)
							compactedItemMap[indexAlias] = val
						}

						AddValue(nestResult, itemActiveProperty, compactedItem, asArray, true)
					}
				} else if isLanguageContainer || isIndexContainer || isIdContainer || isTypeContainer {

					var mapObject map[string]interface{}
					if v, present := nestResult[itemActiveProperty]; present {
						mapObject = v.(map[string]interface{})
					} else {
						mapObject = make(map[string]interface{})
						nestResult[itemActiveProperty] = mapObject
					}

					var mapKey string

					if isLanguageContainer {
						compactedItemMap, isMap := compactedItem.(map[string]interface{})
						compactedItemValue, containsValue := compactedItemMap["@value"]
						if isLanguageContainer && isMap && containsValue {
							compactedItem = compactedItemValue
						}
						if v, found := expandedItemMap["@language"]; found {
							mapKey = v.(string)
						}
					} else if isIndexContainer {
						if v, found := expandedItemMap["@index"]; found {
							mapKey = v.(string)
						}
					} else if isIdContainer {
						idKey := activeCtx.CompactIri("@id", nil, false, false)
						compactedItemMap := compactedItem.(map[string]interface{})
						if compactedItemValue, containsValue := compactedItemMap[idKey]; containsValue {
							mapKey = compactedItemValue.(string)
							delete(compactedItemMap, idKey)
						} else {
							mapKey = ""
						}
					} else if isTypeContainer {
						typeKey := activeCtx.CompactIri("@type", nil, false, false)

						compactedItemMap := compactedItem.(map[string]interface{})
						var types []interface{}
						if compactedItemValue, containsValue := compactedItemMap[typeKey]; containsValue {
							var isArray bool
							types, isArray = compactedItemValue.([]interface{})
							if !isArray {
								types = []interface{}{compactedItemValue}
							}

							delete(compactedItemMap, typeKey)
							if len(types) > 0 {
								mapKey = types[0].(string)
								types = types[1:]
							}
						} else {
							types = make([]interface{}, 0)
						}

						if len(types) > 0 {
							AddValue(compactedItemMap, typeKey, types, false, false)
						}
					}

					if mapKey == "" {
						mapKey = activeCtx.CompactIri("@none", nil, false, false)
					}

					AddValue(mapObject, mapKey, compactedItem, isSetContainer, true)
				} else {
					compactedItemArray, isArray := compactedItem.([]interface{})

					asArray := !compactArrays || isSetContainer || isListContainer ||
						(isArray && len(compactedItemArray) == 0) || expandedProperty == "@list" ||
						expandedProperty == "@graph"
					AddValue(nestResult, itemActiveProperty, compactedItem, asArray, true)
				}
			}
		}

		return result, nil
	}

	return element, nil
}

// checkNestProperty ensures that the value of `@nest` in the term definition must
// either be "@nest", or a term which resolves to "@nest".
func (api *JsonLdApi) checkNestProperty(activeCtx *Context, nestProperty string) error {
	if v, _ := activeCtx.ExpandIri(nestProperty, false, true, nil, nil); v != "@nest" {
		return NewJsonLdError(InvalidNestValue, "nested property must have an @nest value resolving to @nest")
	}
	return nil
}
