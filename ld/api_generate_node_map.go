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
func (api *JsonLdApi) GenerateNodeMap(input interface{}, graphs map[string]interface{}, activeGraph string,
	issuer *IdentifierIssuer, name string, list []interface{}) ([]interface{}, error) {

	// recurse through array
	if elementList, isList := input.([]interface{}); isList {
		// 1.1)
		for _, item := range elementList {
			var err error
			list, err = api.GenerateNodeMap(item, graphs, activeGraph, issuer, "", list)
			if err != nil {
				return nil, err
			}
		}
		return list, nil
	}

	// add non-object to list
	elem, isMap := input.(map[string]interface{})
	if !isMap {
		if list != nil {
			list = append(list, input)
		}
		return list, nil
	}

	// add values to list
	if IsValue(input) {
		if typeVal, hasType := elem["@type"]; hasType {
			// relabel @type blank node
			typeStr := typeVal.(string)
			if strings.HasPrefix(typeStr, "_:") {
				typeStr = issuer.GetId(typeStr)
				elem["@type"] = typeStr
			}
		}
		if list != nil {
			list = append(list, input)
		}
		return list, nil
	}

	// Note: At this point, input must be a subject.

	// spec requires @type to be labeled first, so assign identifiers early
	if typeVal, hasType := elem["@type"]; hasType {
		for _, t := range typeVal.([]interface{}) {
			typeStr := t.(string)
			if strings.HasPrefix(typeStr, "_:") {
				issuer.GetId(typeStr)
			}
		}
	}

	// get identifier for subject
	if name == "" {
		if id, hasID := elem["@id"]; hasID {
			name = id.(string)
		}
		if IsBlankNodeValue(elem) {
			name = issuer.GetId(name)
		}
	}

	// add subject reference to list
	if list != nil {
		list = append(list, map[string]interface{}{
			"@id": name,
		})
	}

	// create new subject or merge into existing one
	subject := setDefault(
		setDefault(
			graphs,
			activeGraph,
			make(map[string]interface{}),
		).(map[string]interface{}),
		name,
		map[string]interface{}{
			"@id": name,
		},
	).(map[string]interface{})
	for _, property := range GetOrderedKeys(elem) {
		// skip @id
		if property == "@id" {
			continue
		}

		// handle reverse properties
		if property == "@reverse" {
			referencedNode := map[string]interface{}{
				"@id": name,
			}
			reverseMap := elem["@reverse"].(map[string]interface{})
			for reverseProperty, items := range reverseMap {
				for _, item := range items.([]interface{}) {
					var itemName string
					if idVal, hasID := item.(map[string]interface{})["@id"]; hasID {
						itemName = idVal.(string)
					}
					if IsBlankNodeValue(item) {
						itemName = issuer.GetId(itemName)
					}
					_, err := api.GenerateNodeMap(item, graphs, activeGraph, issuer, itemName, nil)
					if err != nil {
						return nil, err
					}
					AddValue(graphs[activeGraph].(map[string]interface{})[itemName], reverseProperty, referencedNode, true, false)
				}
			}

			continue
		}

		objects := elem[property]

		// recurse into graph
		if property == "@graph" {
			// add graph subjects map entry
			if _, hasName := graphs[name]; !hasName {
				graphs[name] = make(map[string]interface{})
			}
			g := name
			if activeGraph == "@merged" {
				g = "@merged"
			}
			_, err := api.GenerateNodeMap(objects, graphs, g, issuer, "", nil)
			if err != nil {
				return nil, err
			}

			continue
		}

		// copy non-@type keywords
		if property != "@type" && IsKeyword(property) {
			if subjIndex, hasIndex := subject["@index"]; hasIndex && property == "@index" && (subjIndex != elem["@index"] || subject["@index"].(map[string]interface{})["@id"] != elem["@index"].(map[string]interface{})["@id"]) {
				return nil, NewJsonLdError(ConflictingIndexes, "conflicting @index property detected")
			}
			subject[property] = elem[property]

			continue
		}

		// if property is a bnode, assign it a new id
		if strings.HasPrefix(property, "_:") {
			property = issuer.GetId(property)
		}

		// ensure property is added for empty arrays
		if len(objects.([]interface{})) == 0 {
			AddValue(subject, property, []interface{}{}, true, true)
		}

		for _, o := range objects.([]interface{}) {
			if property == "@type" {
				// rename @type blank nodes
				oStr := o.(string)
				if strings.HasPrefix(oStr, "_:") {
					o = issuer.GetId(oStr)
				}
			}

			// handle embedded subject or subject reference
			if IsSubject(o) || IsSubjectReference(o) {
				// rename blank node @id
				var id string
				if idVal, hasID := o.(map[string]interface{})["@id"]; hasID {
					id = idVal.(string)
				}
				if IsBlankNodeValue(o) {
					id = issuer.GetId(id)
				}

				// add reference and recurse
				AddValue(subject, property, map[string]interface{}{
					"@id": id,
				}, true, false)
				if _, err := api.GenerateNodeMap(o, graphs, activeGraph, issuer, id, nil); err != nil {
					return nil, err
				}
			} else if IsList(o) {
				// handle @list
				oList := make([]interface{}, 0)
				var err error
				if oList, err = api.GenerateNodeMap(o.(map[string]interface{})["@list"], graphs, activeGraph, issuer, name, oList); err != nil {
					return nil, err
				}
				newO := map[string]interface{}{
					"@list": oList,
				}
				AddValue(subject, property, newO, true, false)
			} else {
				// handle @value
				if _, err := api.GenerateNodeMap(o, graphs, activeGraph, issuer, name, nil); err != nil {
					return nil, err
				}
				AddValue(subject, property, o, true, false)
			}
		}
	}

	return list, nil
}

func setDefault(m map[string]interface{}, key string, val interface{}) interface{} {
	if v, ok := m[key]; ok {
		return v
	} else {
		m[key] = val
		return val
	}
}
