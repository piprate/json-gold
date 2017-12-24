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

// Embed is an enum representing allowed Embed flag options as per Framing spec
type Embed int

const (
	Always Embed = 1 + iota
	Never
	Last
	Link
)

// EmbedNode represents embed meta info
type EmbedNode struct {
	parent   interface{}
	property string
}

// FramingContext stores framing state
type FramingContext struct {
	embed        Embed
	explicit     bool
	omitDefault  bool
	uniqueEmbeds map[string]*EmbedNode
	subjectStack []string
}

// NewFramingContext creates and returns as new framing context.
func NewFramingContext(opts *JsonLdOptions) *FramingContext {
	context := &FramingContext{
		embed:        Last,
		explicit:     false,
		omitDefault:  false,
		uniqueEmbeds: make(map[string]*EmbedNode),
		subjectStack: make([]string, 0),
	}

	if opts != nil {
		// TODO: make embed field a selector instead of a boolean, as per new spec.
		embedVal := Never
		if opts.Embed {
			embedVal = Last
		}
		context.embed = embedVal
		context.explicit = opts.Explicit
		context.omitDefault = opts.OmitDefault
	}

	return context
}

// Frame performs JSON-LD framing as defined in:
//
// http://json-ld.org/spec/latest/json-ld-framing/
//
// Frames the given input using the frame according to the steps in the Framing Algorithm.
// The input is used to build the framed output and is returned if there are no errors.
//
// Returns the framed output.
func (api *JsonLdApi) Frame(input interface{}, frame []interface{}, opts *JsonLdOptions) ([]interface{}, error) {
	issuer := NewIdentifierIssuer("_:b")

	// create framing state
	state := NewFramingContext(opts)

	nodes := make(map[string]interface{})
	api.GenerateNodeMap(input, nodes, "@default", nil, "", nil, issuer)
	nodeMap := nodes["@default"].(map[string]interface{})

	framed := make([]interface{}, 0)

	// NOTE: frame validation is done by the function not allowing anything
	// other than list to me passed
	// 1.
	// If frame is an array, set frame to the first member of the array, which MUST be a valid frame.
	var frameParam map[string]interface{}
	if frame != nil && len(frame) > 0 {
		frameParam = frame[0].(map[string]interface{})
	} else {
		frameParam = make(map[string]interface{})
	}
	framedVal, err := api.frame(state, nodeMap, nodeMap, frameParam, framed, "")
	if err != nil {
		return nil, err
	}
	return framedVal.([]interface{}), nil
}

func createsCircularReference(id string, state *FramingContext) bool {
	for _, i := range state.subjectStack {
		if i == id {
			return true
		}
	}
	return false
}

// frame subjects according to the given frame.
// state: the current framing state
// nodes:
// nodeMap: node map
// frame: the frame
// parent: the parent subject or top-level array
// property: the parent property, initialized to nil
func (api *JsonLdApi) frame(state *FramingContext, nodes map[string]interface{}, nodeMap map[string]interface{},
	frame map[string]interface{}, parent interface{}, property string) (interface{}, error) {
	// https://json-ld.org/spec/latest/json-ld-framing/#framing-algorithm

	// 2.
	// Initialize flags embed, explicit, and requireAll from object embed flag,
	// explicit inclusion flag, and require all flag in state overriding from
	// any property values for @embed, @explicit, and @requireAll in frame.
	// TODO: handle @requireAll
	embed, err := getFrameEmbed(frame, state.embed)
	if err != nil {
		return nil, err
	}
	explicitOn := GetFrameFlag(frame, "@explicit", state.explicit)
	flags := make(map[string]interface{})
	flags["@explicit"] = explicitOn
	flags["@embed"] = embed

	// 3.
	// Create a list of matched subjects by filtering subjects against frame
	// using the Frame Matching algorithm with state, subjects, frame, and requireAll.
	matches, err := FilterNodes(nodes, frame)
	if err != nil {
		return nil, err
	}

	// 4.
	// Set link the the value of link in state associated with graph name in state,
	// creating a new empty dictionary, if necessary. TODO
	//link := state.uniqueEmbeds;

	// 5.
	// For each id and associated node object node from the set of matched subjects, ordered by id:
	for _, id := range GetOrderedKeys(matches) {
		// 5.1
		// Initialize output to a new dictionary with @id and id and add output to link associated with id.
		output := make(map[string]interface{})
		output["@id"] = id

		// 5.2
		// If embed is @link and id is in link, node already exists in results.
		// Add the associated node object from link to parent and do not perform
		// additional processing for this node.
		if embed == Link {
			if idVal, containsId := state.uniqueEmbeds[id]; containsId {
				parent = addFrameOutput(parent, property, idVal)
				continue
			}
		}

		// Occurs only at top level, compartmentalize each top-level match
		if property == "" {
			state.uniqueEmbeds = make(map[string]*EmbedNode)
		}

		// 5.3
		// Otherwise, if embed is @never or if a circular reference would be created by an embed,
		// add output to parent and do not perform additional processing for this node.
		if embed == Never || createsCircularReference(id, state) {
			parent = addFrameOutput(parent, property, output)
			continue
		}

		// 5.4
		// Otherwise, if embed is @last, remove any existing embedded node from parent associated
		// with graph name in state. Requires sorting of subjects.
		if embed == Last {
			if _, containsId := state.uniqueEmbeds[id]; containsId {
				removeEmbed(state, id)
			}
			state.uniqueEmbeds[id] = &EmbedNode{
				parent:   parent,
				property: property,
			}
		}

		state.subjectStack = append(state.subjectStack, id)

		// 5.5 If embed is @last or @always

		// Skip 5.5.1

		// 5.5.2 For each property and objects in node, ordered by property:
		element := matches[id].(map[string]interface{})
		for _, prop := range GetOrderedKeys(element) {

			// 5.5.2.1 If property is a keyword, add property and objects to output.
			if IsKeyword(prop) {
				output[prop] = CloneDocument(element[prop])
				continue
			}

			// 5.5.2.2 Otherwise, if property is not in frame, and explicit is true, processors
			// MUST NOT add any values for property to output, and the following steps are skipped.
			framePropVal, containsProp := frame[prop]
			if explicitOn && !containsProp {
				continue
			}

			// add objects
			value := element[prop].([]interface{})

			// 5.5.2.3 For each item in objects:
			for _, item := range value {
				itemMap, isMap := item.(map[string]interface{})
				listValue, hasList := itemMap["@list"]
				if isMap && hasList {
					// add empty list
					list := make(map[string]interface{})
					list["@list"] = make([]interface{}, 0)
					addFrameOutput(output, prop, list)

					// add list objects
					for _, listitem := range listValue.([]interface{}) {
						// 5.5.2.3.1.1 recurse into subject reference
						if IsNodeReference(listitem) {
							tmp := make(map[string]interface{})
							itemid := listitem.(map[string]interface{})["@id"].(string)
							// TODO: nodes may need to be node_map,
							// which is global
							tmp[itemid] = nodeMap[itemid]

							subframe := make(map[string]interface{})
							if containsProp {
								subframe = framePropVal.([]map[string]interface{})[0]
							} else {
								subframe = flags
							}
							api.frame(state, tmp, nodeMap, subframe, list, "@list")
						} else {
							// include other values automatically (TODO:
							// may need Clone(n)
							addFrameOutput(list, "@list", listitem)
						}
					}
				} else if IsNodeReference(item) { // recurse into subject reference
					tmp := make(map[string]interface{})
					itemid := item.(map[string]interface{})["@id"].(string)
					// TODO: nodes may need to be node_map, which is
					// global
					tmp[itemid] = nodeMap[itemid]
					subframe := make(map[string]interface{})
					if containsProp {
						subframe = framePropVal.([]interface{})[0].(map[string]interface{})
					} else {
						subframe = flags
					}
					api.frame(state, tmp, nodeMap, subframe, output, prop)
				} else {
					// include other values automatically (TODO: may
					// need JsonLdUtils.clone(o))
					addFrameOutput(output, prop, item)
				}
			}

		}

		// handle defaults
		for _, prop := range GetOrderedKeys(frame) {
			// skip keywords
			if IsKeyword(prop) {
				continue
			}

			pf := frame[prop].([]interface{})
			var propertyFrame map[string]interface{}
			if len(pf) > 0 {
				propertyFrame = pf[0].(map[string]interface{})
			}

			if propertyFrame == nil {
				propertyFrame = make(map[string]interface{})
			}

			omitDefaultOn := GetFrameFlag(propertyFrame, "@omitDefault", state.omitDefault)
			if _, hasProp := output[prop]; !omitDefaultOn && !hasProp {
				var def interface{} = "@null"
				if defaultVal, hasDefault := propertyFrame["@default"]; hasDefault {
					def = CloneDocument(defaultVal)
				}
				if _, isList := def.([]interface{}); !isList {
					def = []interface{}{def}
				}
				output[prop] = []interface{}{
					map[string]interface{}{
						"@preserve": def,
					},
				}
			}
		}

		// add output to parent
		parent = addFrameOutput(parent, property, output)

		state.subjectStack = state.subjectStack[:len(state.subjectStack)-1]
	}

	return parent, nil
}

func getFrameValue(frame map[string]interface{}, name string) interface{} {
	value := frame[name]
	if valueList, isList := value.([]interface{}); isList {
		if len(valueList) > 0 {
			value = valueList[0]
		}
	} else if valueMap, isMap := value.(map[string]interface{}); isMap {
		if v, containsValue := valueMap["@value"]; containsValue {
			value = v
		}
	}
	return value
}

// GetFrameFlag gets the frame flag value for the given flag name.
// If boolean value is not found, returns theDefault
func GetFrameFlag(frame map[string]interface{}, name string, theDefault bool) bool {
	value := frame[name]
	switch v := value.(type) {
	case []interface{}:
		if len(v) > 0 {
			value = v[0]
		}
	case map[string]interface{}:
		if valueVal, present := v["@value"]; present {
			value = valueVal
		}
	case bool:
		return v
	}

	if valueBool, isBool := value.(bool); isBool {
		return valueBool
	}

	return theDefault
}

func getFrameEmbed(frame map[string]interface{}, theDefault Embed) (Embed, error) {

	value := getFrameValue(frame, "@embed")
	if value == nil {
		return theDefault, nil
	}
	if boolVal, isBoolean := value.(bool); isBoolean {
		if boolVal {
			return Last, nil
		} else {
			return Never, nil
		}
	}
	if embedVal, isEmbed := value.(Embed); isEmbed {
		return embedVal, nil
	}
	if stringVal, isString := value.(string); isString {
		switch stringVal {
		case "@always":
			return Always, nil
		case "@never":
			return Never, nil
		case "@last":
			return Last, nil
		case "@link":
			return Link, nil
		default:
			return Last, NewJsonLdError(SyntaxError, "invalid @embed value")
		}
	}
	return Last, NewJsonLdError(SyntaxError, "invalid @embed value")
}

// removeEmbed removes an existing embed with the given id.
func removeEmbed(state *FramingContext, id string) {
	// get existing embed
	links := state.uniqueEmbeds
	embed := links[id]
	parent := embed.parent
	property := embed.property

	// create reference to replace embed
	node := make(map[string]interface{})
	node["@id"] = id

	// remove existing embed
	if IsNode(parent) {
		// replace subject with reference
		newVals := make([]interface{}, 0)
		parentMap := parent.(map[string]interface{})
		oldvals := parentMap[property].([]interface{})
		for _, v := range oldvals {
			vMap, isMap := v.(map[string]interface{})
			if isMap && vMap["@id"] == id {
				newVals = append(newVals, node)
			} else {
				newVals = append(newVals, v)
			}
		}
		parentMap[property] = newVals
	}
	// recursively remove dependent dangling embeds
	removeDependents(links, id)
}

// removeDependents recursively removes dependent dangling embeds.
func removeDependents(embeds map[string]*EmbedNode, id string) {
	// get embed keys as a separate array to enable deleting keys in map
	for idDep, e := range embeds {
		var p map[string]interface{}
		if e.parent != nil {
			var isMap bool
			p, isMap = e.parent.(map[string]interface{})
			if !isMap {
				continue
			}
		} else {
			p = make(map[string]interface{})
		}

		pid := p["@id"].(string)
		if id == pid {
			delete(embeds, idDep)
			removeDependents(embeds, idDep)
		}
	}
}

// FilterNodes returns a map of all of the nodes that match a parsed frame.
func FilterNodes(nodes map[string]interface{}, frame map[string]interface{}) (map[string]interface{}, error) {
	rval := make(map[string]interface{})
	for id, elementVal := range nodes {
		element, _ := elementVal.(map[string]interface{})
		if element != nil {
			if res, err := FilterNode(element, frame); res {
				if err != nil {
					return nil, err
				}
				rval[id] = element
			}
		}
	}
	return rval, nil
}

// FilterNode returns true if the given node matches the given frame.
func FilterNode(node map[string]interface{}, frame map[string]interface{}) (bool, error) {
	types, _ := frame["@type"]
	frameIds, _ := frame["@id"]

	// https://json-ld.org/spec/latest/json-ld-framing/#frame-matching
	//
	// 1. Node matches if it has an @id property including any IRI or
	// blank node in the @id property in frame.
	if frameIds != nil {
		if _, isString := frameIds.(string); isString {
			nodeId, _ := node["@id"]
			if nodeId == nil {
				return false, nil
			}
			if DeepCompare(nodeId, frameIds, false) {
				return true, nil
			}
		} else {
			frameIdList, isList := frameIds.([]interface{})
			if !isList {
				return false, NewJsonLdError(SyntaxError, "frame @id must be an array")
			} else {
				nodeId, _ := node["@id"]
				if nodeId == nil {
					return false, nil
				}
				for _, j := range frameIdList {
					if DeepCompare(nodeId, j, false) {
						return true, nil
					}
				}
			}
		}
		return false, nil
	}
	// 2. Node matches if frame has no non-keyword properties.TODO
	// 3.1 If property is @type:
	if types != nil {
		typesList, isList := types.([]interface{})
		if !isList {
			return false, NewJsonLdError(SyntaxError, "frame @type must be an array")
		}
		nodeTypesVal, nodeHasType := node["@type"]
		var nodeTypes []interface{}
		if !nodeHasType {
			nodeTypes = make([]interface{}, 0)
		} else if nodeTypes, isList = nodeTypesVal.([]interface{}); !isList {
			return false, NewJsonLdError(SyntaxError, "node @type must be an array")
		}
		// 3.1.1 Property matches if the @type property in frame includes any IRI in values.
		for _, i := range nodeTypes {
			for _, j := range typesList {
				if DeepCompare(i, j, false) {
					return true, nil
				}
			}
		}
		// TODO: 3.1.2
		// 3.1.3 Otherwise, property matches if values is empty and the @type property in frame is match none.
		if len(typesList) == 1 {
			vMap, isMap := typesList[0].(map[string]interface{})
			if isMap && len(vMap) == 0 {
				return len(nodeTypes) > 0, nil
			}
		}
		// 3.1.4 Otherwise, property does not match.
		return false, nil
	}

	// 3.2
	for _, key := range GetKeys(frame) {
		_, nodeContainsKey := node[key]
		if !IsKeyword(key) && !nodeContainsKey {
			frameObject := frame[key]
			if oList, isList := frameObject.([]interface{}); isList {
				_default := false
				for _, obj := range oList {
					if oMap, isMap := obj.(map[string]interface{}); isMap {
						if _, containsKey := oMap["@default"]; containsKey {
							_default = true
						}
					}
				}
				if _default {
					continue
				}
			}

			return false, nil
		}
	}

	return true, nil
}

// addFrameOutput adds framing output to the given parent.
// parent: the parent to add to.
// property: the parent property.
// output: the output to add.
func addFrameOutput(parent interface{}, property string, output interface{}) interface{} {
	if parentMap, isMap := parent.(map[string]interface{}); isMap {
		propVal, hasProperty := parentMap[property]
		if hasProperty {
			parentMap[property] = append(propVal.([]interface{}), output)
		} else {
			parentMap[property] = []interface{}{output}
		}
		return parentMap
	}

	return append(parent.([]interface{}), output)
}
