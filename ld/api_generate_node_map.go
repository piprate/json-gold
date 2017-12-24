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
	"strings"
)

// GenerateNodeMap recursively flattens the subjects in the given JSON-LD expanded
// input into a node map.
func (api *JsonLdApi) GenerateNodeMap(element interface{}, nodeMap map[string]interface{}, activeGraph string,
	activeSubject interface{}, activeProperty string, list map[string]interface{},
	issuer *IdentifierIssuer) error {
	// 1)
	if elementList, isList := element.([]interface{}); isList {
		// 1.1)
		for _, item := range elementList {
			if err := api.GenerateNodeMap(item, nodeMap, activeGraph, activeSubject, activeProperty, list, issuer); err != nil {
				return err
			}
		}
		return nil
	}

	// for convenience
	elem := element.(map[string]interface{})

	// 2)
	if _, present := nodeMap[activeGraph]; !present {
		nodeMap[activeGraph] = make(map[string]interface{})
	}

	graph := nodeMap[activeGraph].(map[string]interface{})
	var node map[string]interface{}
	if activeSubjectStr, isString := activeSubject.(string); activeSubject != nil && isString {
		node = graph[activeSubjectStr].(map[string]interface{})
	}

	// 3)
	if typeVal, hasType := elem["@type"]; hasType {
		// 3.1)
		oldTypes := make([]string, 0)
		newTypes := make([]string, 0)
		typeList, isList := typeVal.([]interface{})
		if isList {
			for _, v := range typeList {
				oldTypes = append(oldTypes, v.(string))
			}
		} else {
			oldTypes = append(oldTypes, typeVal.(string))
		}
		for _, item := range oldTypes {
			if strings.HasPrefix(item, "_:") {
				newTypes = append(newTypes, issuer.GetId(item))
			} else {
				newTypes = append(newTypes, item)
			}
		}
		if isList {
			elem["@type"] = newTypes
		} else {
			elem["@type"] = newTypes[0]
		}
	}

	// 4)
	if _, hasValue := elem["@value"]; hasValue {
		// 4.1)
		if list == nil {
			MergeValue(node, activeProperty, elem)
		} else {
			// 4.2)
			MergeValue(list, "@list", elem)
		}
	} else if listVal, hasList := elem["@list"]; hasList { // 5)
		// 5.1)
		result := make(map[string]interface{})
		result["@list"] = make([]interface{}, 0)
		// 5.2)
		api.GenerateNodeMap(listVal, nodeMap, activeGraph, activeSubject, activeProperty, result, issuer)
		// 5.3)
		MergeValue(node, activeProperty, result)
	} else { // 6)
		// 6.1)
		idVal, hasID := elem["@id"]
		id, _ := idVal.(string)
		delete(elem, "@id")

		if hasID {
			if strings.HasPrefix(id, "_:") {
				id = issuer.GetId(id)
			}
		} else {
			// 6.2)
			id = issuer.GetId("")
		}
		// 6.3)
		if _, hasID := graph[id]; !hasID {
			graph[id] = map[string]interface{}{"@id": id}
		}

		// 6.4) TODO: SPEC this line is asked for by the spec, but it breaks
		// various tests
		// node = graph[id].(map[string]interface{})
		// 6.5)
		if _, isMap := activeSubject.(map[string]interface{}); isMap {
			// 6.5.1)
			MergeValue(graph[id].(map[string]interface{}), activeProperty, activeSubject)
		} else if activeProperty != "" { // 6.6)
			reference := make(map[string]interface{})
			reference["@id"] = id

			// 6.6.2)
			if list == nil {
				// 6.6.2.1+2)
				MergeValue(node, activeProperty, reference)
			} else {
				// 6.6.3) TODO: SPEC says to add ELEMENT to @list member, should
				// be REFERENCE
				MergeValue(list, "@list", reference)
			}
		}

		// TODO: SPEC this is removed in the spec now, but it's still needed
		// (see 6.4)
		node = graph[id].(map[string]interface{})
		// 6.7)
		if typeListVal, hasType := elem["@type"]; hasType {
			typeList := typeListVal.([]string)
			delete(elem, "@type")
			for _, typeVal := range typeList {
				MergeValue(node, "@type", typeVal)
			}
		}

		// 6.8)
		if elemIndex, hasIndex := elem["@index"]; hasIndex {
			delete(elem, "@index")
			if indexVal, nodeHasIndex := node["@index"]; nodeHasIndex {
				if !DeepCompare(indexVal, elemIndex, false) {
					return NewJsonLdError(ConflictingIndexes, nil)
				}
			} else {
				node["@index"] = elemIndex
			}
		}

		// 6.9)
		if reverseVal, hasReverse := elem["@reverse"]; hasReverse {
			// 6.9.1)
			referencedNode := make(map[string]interface{})
			referencedNode["@id"] = id
			// 6.9.2+6.9.4)
			reverseMap := reverseVal.(map[string]interface{})
			delete(elem, "@reverse")

			// 6.9.3)
			for _, property := range GetKeys(reverseMap) {
				values := reverseMap[property].([]interface{})
				// 6.9.3.1)
				for _, value := range values {
					// 6.9.3.1.1)
					api.GenerateNodeMap(value, nodeMap, activeGraph, referencedNode, property, nil, issuer)
				}
			}
		}

		// 6.10)
		if graphVal, hasGraph := elem["@graph"]; hasGraph {
			delete(elem, "@graph")
			api.GenerateNodeMap(graphVal, nodeMap, id, nil, "", nil, issuer)
		}

		// 6.11)
		for _, property := range GetOrderedKeys(elem) {
			value := elem[property]
			// 6.11.1)
			if strings.HasPrefix(property, "_:") {
				property = issuer.GetId(property)
			}
			// 6.11.2)
			if _, hasProperty := node[property]; !hasProperty {
				node[property] = make([]interface{}, 0)
			}
			// 6.11.3)
			api.GenerateNodeMap(value, nodeMap, activeGraph, id, property, nil, issuer)
		}
	}

	return nil
}
