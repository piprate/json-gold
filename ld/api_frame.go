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

// EmbedNode represents embed meta info
type EmbedNode struct {
	parent   interface{}
	property string
}

type StackNode struct {
	subject map[string]interface{}
	graph   string
}

// FramingContext stores framing state
type FramingContext struct {
	embed        Embed
	embedded     bool
	explicit     bool
	requireAll   bool
	omitDefault  bool
	ordered      bool
	uniqueEmbeds map[string]map[string]*EmbedNode
	graphMap     map[string]interface{}
	subjects     map[string]interface{}
	graph        string
	graphStack   []string // TODO: is this field needed?
	subjectStack []*StackNode
	bnodeMap     map[string]interface{}
}

// NewFramingContext creates and returns as new framing context.
func NewFramingContext(opts *JsonLdOptions) *FramingContext {
	context := &FramingContext{
		embed:        EmbedOnce,
		embedded:     false,
		explicit:     false,
		requireAll:   false,
		omitDefault:  false,
		ordered:      false,
		uniqueEmbeds: make(map[string]map[string]*EmbedNode),
		graphMap: map[string]interface{}{
			"@default": make(map[string]interface{}),
		},
		graph:        "@default",
		graphStack:   make([]string, 0),
		subjectStack: make([]*StackNode, 0),
		bnodeMap:     make(map[string]interface{}),
	}

	if opts != nil {
		context.embed = opts.Embed
		context.explicit = opts.Explicit
		context.requireAll = opts.RequireAll
		context.omitDefault = opts.OmitDefault
		context.ordered = opts.Ordered
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
func (api *JsonLdApi) Frame(input interface{}, frame []interface{}, opts *JsonLdOptions, merged bool) ([]interface{}, []string, error) {

	// create framing state
	state := NewFramingContext(opts)

	// produce a map of all graphs and name each bnode
	issuer := NewIdentifierIssuer("_:b")
	if _, err := api.GenerateNodeMap(input, state.graphMap, "@default", issuer, "", "", nil); err != nil {
		return nil, nil, err
	}

	if merged {
		state.graphMap["@merged"] = api.mergeNodeMapGraphs(state.graphMap)
		state.graph = "@merged"
	}
	state.subjects = state.graphMap[state.graph].(map[string]interface{})

	// validate the frame
	if err := validateFrame(frame); err != nil {
		return nil, nil, err
	}

	// 1.
	// If frame is an array, set frame to the first member of the array.
	var frameParam map[string]interface{}
	if len(frame) > 0 {
		frameParam = frame[0].(map[string]interface{})
	} else {
		frameParam = make(map[string]interface{})
	}

	framed := make([]interface{}, 0)
	framedVal, err := api.matchFrame(state, GetOrderedKeys(state.subjects), frameParam, framed, "")
	if err != nil {
		return nil, nil, err
	}

	bnodesToClear := make([]string, 0)
	for id, val := range state.bnodeMap {
		if valArray, isArray := val.([]interface{}); isArray && len(valArray) == 1 {
			bnodesToClear = append(bnodesToClear, id)
		}
	}
	return framedVal.([]interface{}), bnodesToClear, nil
}

func createsCircularReference(id string, graph string, state *FramingContext) bool {
	for i := len(state.subjectStack) - 1; i >= 0; i-- {
		subject := state.subjectStack[i]
		if subject.graph == graph && subject.subject["@id"] == id {
			return true
		}
	}
	return false
}

func (api *JsonLdApi) mergeNodeMapGraphs(graphs map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})

	for _, name := range GetOrderedKeys(graphs) {
		graph := graphs[name].(map[string]interface{})
		for _, id := range GetOrderedKeys(graph) {
			var mergedNode map[string]interface{}
			mv, hasID := merged[id]
			if !hasID {
				mergedNode = map[string]interface{}{
					"@id": id,
				}
				merged[id] = mergedNode
			} else {
				mergedNode = mv.(map[string]interface{})
			}
			node := graph[id].(map[string]interface{})
			for _, property := range GetOrderedKeys(node) {
				if IsKeyword(property) {
					if property == "@type" {
						// merge @type arrays across graphs instead of overwriting
						for _, v := range Arrayify(node[property]) {
							AddValue(mergedNode, property, CloneDocument(v), true, false, false, false)
						}
					} else {
						// copy other keywords
						mergedNode[property] = CloneDocument(node[property])
					}
				} else {
					// merge objects
					for _, v := range node[property].([]interface{}) {
						AddValue(mergedNode, property, CloneDocument(v), true, false, false, false)
					}
				}
			}
		}
	}

	return merged
}

// matchFrame frames subjects according to the given frame.
//
// state: the current framing state
// nodes:
// frame: the frame
// parent: the parent subject or top-level array
// property: the parent property, initialized to ""
func (api *JsonLdApi) matchFrame(state *FramingContext, subjects []string,
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
	requireAll := GetFrameFlag(frame, "@requireAll", state.requireAll)
	flags := map[string]interface{}{
		"@explicit":   []interface{}{explicitOn},
		"@requireAll": []interface{}{requireAll},
		"@embed":      []interface{}{embed},
	}

	// 3.
	// Create a list of matched subjects by filtering subjects against frame
	// using the Frame Matching algorithm with state, subjects, frame, and requireAll.
	matches, err := FilterSubjects(state, subjects, frame, requireAll)
	if err != nil {
		return nil, err
	}

	// 4.
	// For each id and associated node object node from the set of matched subjects, ordered
	// lexicographically by id if the optional ordered flag is true:
	for _, id := range GetOrderedKeys(matches) {

		// Note: In order to treat each top-level match as a
		// compartmentalized result, clear the unique embedded subjects map
		// when the property is None, which only occurs at the top-level.
		if property == "" {
			state.uniqueEmbeds = map[string]map[string]*EmbedNode{
				state.graph: make(map[string]*EmbedNode),
			}
		} else if _, found := state.uniqueEmbeds[state.graph]; !found {
			state.uniqueEmbeds[state.graph] = make(map[string]*EmbedNode)
		}

		// 4.1
		// Initialize output to a new dictionary with @id and id
		output := make(map[string]interface{})
		output["@id"] = id

		// keep track of objects having blank nodes
		if strings.HasPrefix(id, "_:") {
			AddValue(state.bnodeMap, id, output, true, false, true, false)
		}

		// 4.2
		// If the embedded flag in state is false and there is an existing embedded node in parent
		// associated with graph name and id in state, do not perform additional processing for
		// this node.
		if !state.embedded {
			if embeds, ok := state.uniqueEmbeds[state.graph]; ok {
				if _, exists := embeds[id]; exists {
					continue
				}
			}
		}

		// 4.3
		// Otherwise, if the embedded flag in state is true and either embed is @never
		// or if a circular reference would be created by an embed,
		// add output to parent and do not perform additional processing for this node.
		if state.embedded {
			if embed == EmbedNever || createsCircularReference(id, state.graph, state) {
				parent = addFrameOutput(parent, property, output)
				continue
			}
		}

		// 4.4
		// Otherwise, if the embedded flag in state is true, embed is @once, and there is an
		// existing embedded node in parent associated with graph name and id in state, add output
		// to parent and do not perform additional processing for this node.
		if state.embedded {
			if embed == EmbedOnce {
				if embeds, ok := state.uniqueEmbeds[state.graph]; ok {
					if _, exists := embeds[id]; exists {
						parent = addFrameOutput(parent, property, output)
						continue
					}
				}
			}
			// For @last: if we've seen this node before, remove the previous embed
			// (replacing it with a node reference), then fall through to embed this occurrence.
			if embed == EmbedLast {
				if _, containsID := state.uniqueEmbeds[state.graph][id]; containsID {
					removeEmbed(state, id)
				}
				state.uniqueEmbeds[state.graph][id] = &EmbedNode{
					parent:   parent,
					property: property,
				}
			}
		}

		// Otherwise, if embed is @once, remove any existing embedded node from parent associated
		// with graph name in state. Requires sorting of subjects.
		if embed == EmbedOnce {
			if _, containsID := state.uniqueEmbeds[state.graph][id]; containsID {
				removeEmbed(state, id)
			}
			state.uniqueEmbeds[state.graph][id] = &EmbedNode{
				parent:   parent,
				property: property,
			}
		}

		subject := matches[id].(map[string]interface{})

		state.subjectStack = append(state.subjectStack, &StackNode{
			subject: subject,
			graph:   state.graph,
		})

		// 4.5
		// If graph map in state has an entry for id:
		// subject is also the name of a graph
		if _, isAlsoGraph := state.graphMap[id]; isAlsoGraph {
			// 4.5.1
			// If frame does not have a @graph entry, set recurse to true,
			// unless graph name in state is @merged and set subframe to a new empty map.
			var recurse bool
			var subframe map[string]interface{}
			if _, hasGraph := frame["@graph"]; !hasGraph {
				recurse = state.graph != "@merged"
				subframe = make(map[string]interface{})
			} else {
				// 4.5.2
				// Otherwise, set subframe to the first entry for @graph in frame, or a new empty map,
				// if it does not exist, and set recurse to true, unless id is @merged or @default.
				if v, isMap := frame["@graph"].([]interface{})[0].(map[string]interface{}); isMap {
					subframe = v
				} else {
					subframe = make(map[string]interface{})
				}
				recurse = !(id == "@merged" || id == "@default")
			}

			// 4.5.3
			// If recurse is true:
			if recurse {
				state.graphStack = append(state.graphStack, state.graph)
				// 4.5.3.1
				// Set the value of graph name in state to id.
				state.graph = id

				// 4.5.3.2
				// Set the value of embedded flag in state to false.
				state_embedded := state.embedded
				state.embedded = false

				// 4.5.3.3
				// Invoke the algorithm using a copy of state with the value of graph name set to
				// id and the value of embedded flag set to false, the keys from the graph map in
				// state associated with id as subjects, subframe as frame, output as parent,
				// and @graph as active property.

				// recurse into graph
				subjects := GetOrderedKeys(state.graphMap[state.graph].(map[string]interface{}))
				if _, err = api.matchFrame(state, subjects, subframe, output, "@graph"); err != nil {
					return nil, err
				}
				// reset to current graph
				state.embedded = state_embedded // TODO: is this required?
				state.graph = state.graphStack[len(state.graphStack)-1]
				state.graphStack = state.graphStack[:len(state.graphStack)-1]
			}
		}

		// 4.6
		// If frame has an @included entry, invoke the algorithm using a copy of state with the
		// value of embedded flag set to false, subjects, frame, output as parent,
		// and @included as active property.
		// FIXME: The specification is syntactically incomplete.
		//        Below implementation reads as if "set to false, included as subjects"
		if _, hasIncluded := frame["@included"]; hasIncluded {
			var included []interface{}
			included = Arrayify(frame["@included"])

			for _, i := range included {
				state_embedded := state.embedded
				state.embedded = false
				includeMap, ok := i.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("expected map[string]interface{} in @included, got %T", i)
				}
				include_subjects := GetOrderedKeys(includeMap)
				if _, err = api.matchFrame(state, include_subjects, frame, output, "@included"); err != nil {
					return nil, err
				}
				// reset to current graph
				state.embedded = state_embedded
			}
		}

		// 4.7
		// For each property and objects in node, ordered lexicographically by property if the
		// optional ordered flag is true:
		for _, prop := range GetOrderedKeys(subject) {
			// 4.7.1
			// If property is a keyword, add property and objects to output.
			if IsKeyword(prop) {
				output[prop] = CloneDocument(subject[prop])

				if prop == "@type" {
					// count bnode values of @type
					for _, t := range subject[prop].([]interface{}) {
						if strings.HasPrefix(t.(string), "_:") {
							AddValue(state.bnodeMap, t.(string), output, true, false, true, false)
						}
					}
				}
				continue
			}

			// 4.7.2
			// Otherwise, if property is not in frame, and explicit is true,
			// processors MUST NOT add any values for property to output, and the following steps are skipped.
			framePropVal, containsProp := frame[prop]
			if explicitOn && !containsProp {
				continue
			}

			// add objects
			// 4.7.3 For each item in objects:
			for _, item := range subject[prop].([]interface{}) {
				itemMap, isMap := item.(map[string]interface{})
				listValue, hasList := itemMap["@list"]
				if isMap && hasList {
					// 4.7.3.1
					// If item is a map with the property @list, then each listitem in the list is
					// processed in sequence and added to a new list map in output:
					// add empty list
					list := map[string]interface{}{
						"@list": make([]interface{}, 0),
					}
					addFrameOutput(output, prop, list)

					// add list objects
					for _, listitem := range listValue.([]interface{}) {
						// 4.7.3.1.1
						// If listitem is a node reference, invoke the algorithm using a copy of
						// state with the value of embedded flag set to true, the value of @id from
						// listitem as the sole item in a new subjects array, the first value from
						// @list in frame as frame, list as parent, and @list as active property.
						//
						// If frame does not exist, create a new frame using a new map with
						// properties for @embed, @explicit and @requireAll taken from embed,
						// explicit and requireAll.
						if IsSubjectReference(listitem) {
							// recurse into subject reference
							itemid := listitem.(map[string]interface{})["@id"].(string)

							var subframe map[string]interface{}
							if containsProp && IsList(framePropVal.([]interface{})[0]) {
								subframe = framePropVal.([]interface{})[0].(map[string]interface{})["@list"].([]interface{})[0].(map[string]interface{})
							} else {
								subframe = flags
							}
							state_embedded := state.embedded
							state.embedded = true
							res, err := api.matchFrame(state, []string{itemid}, subframe, list, "@list")
							if err != nil {
								return nil, err
							}
							// reset to current graph
							state.embedded = state_embedded
							list = res.(map[string]interface{})
						} else {
							// 4.7.3.1.2
							// Otherwise, append a copy of listitem to @list in list.
							// include other values automatically (TODO:
							// may need Clone(n)
							addFrameOutput(list, "@list", listitem)
						}
					}
				} else {
					// 4.7.3.2
					// If item is a node reference, invoke the algorithm using a copy of state with
					// the value of embedded flag set to true, the value of @id from item as the
					// sole item in a new subjects array, the first value from property in frame as
					// frame, output as parent, and property as active property.
					//
					// If frame does not exist, create a new frame using a new map with properties
					// for @embed, @explicit and @requireAll taken from embed, explicit and requireAll.

					var subframe map[string]interface{}
					if containsProp {
						subframe = framePropVal.([]interface{})[0].(map[string]interface{})
					} else {
						subframe = flags
					}

					if IsSubjectReference(item) { // recurse into subject reference
						itemid := itemMap["@id"].(string)
						state_embedded := state.embedded
						state.embedded = true
						if _, err = api.matchFrame(state, []string{itemid}, subframe, output, prop); err != nil {
							return nil, err
						}
						// reset to current graph
						state.embedded = state_embedded

					} else if valueMatch(subframe, itemMap) {
						// 4.7.3.3
						// Otherwise, append a copy of item to active property in output.
						addFrameOutput(output, prop, CloneDocument(item))
					}
				}
			}

		}

		// 4.7.4
		// For each non-keyword property and objects in frame (other than `@type) that is not in output:
		// handle defaults
		for _, prop := range GetOrderedKeys(frame) {
			// skip keywords
			// TODO: Unless it's @type ?
			if IsKeyword(prop) {
				continue
			}

			// if omit default is off, then include default values for
			// properties that appear in the next frame but are not in
			// the matching subject
			var next map[string]interface{}
			if pf, found := frame[prop].([]interface{}); found && len(pf) > 0 {
				next = pf[0].(map[string]interface{})
			} else {
				next = make(map[string]interface{})
			}

			omitDefaultOn := GetFrameFlag(next, "@omitDefault", state.omitDefault)
			if _, hasProp := output[prop]; !omitDefaultOn && !hasProp {
				var preserve interface{} = "@null"
				if defaultVal, hasDefault := next["@default"]; hasDefault {
					preserve = CloneDocument(defaultVal)
				}
				preserve = Arrayify(preserve)
				output[prop] = []interface{}{
					map[string]interface{}{
						"@preserve": preserve,
					},
				}
			}
		}

		// 4.7.5
		// If frame has the property @reverse, then for each reverse property and sub frame that
		// are the values of @reverse in frame:

		// embed reverse values by finding nodes having this subject as a
		// value of the associated property
		if reverse, hasReverse := frame["@reverse"]; hasReverse {
			for _, reverseProp := range GetOrderedKeys(reverse.(map[string]interface{})) {
				for subject, subjectValue := range state.subjects {
					nodeValues := Arrayify(subjectValue.(map[string]interface{})[reverseProp])
					for _, v := range nodeValues {
						if v != nil && v.(map[string]interface{})["@id"] == id {
							// node has property referencing this subject, recurse
							outputReverse, hasReverse := output["@reverse"]
							if !hasReverse {
								outputReverse = make(map[string]interface{})
								output["@reverse"] = outputReverse
							}
							AddValue(output["@reverse"], reverseProp, []interface{}{}, true,
								false, true, false)
							var subframe map[string]interface{}
							sf := reverse.(map[string]interface{})[reverseProp]
							if sfArray, isArray := sf.([]interface{}); isArray {
								subframe = sfArray[0].(map[string]interface{})
							} else {
								subframe = sf.(map[string]interface{})
							}
							res, err := api.matchFrame(state, []string{subject}, subframe, outputReverse.(map[string]interface{})[reverseProp], property)
							if err != nil {
								return nil, err
							}
							outputReverse.(map[string]interface{})[reverseProp] = res
							break
						}
					}
				}
			}
		}

		// add output to parent
		parent = addFrameOutput(parent, property, output)

		// pop matching subject from circular ref-checking stack
		state.subjectStack = state.subjectStack[:len(state.subjectStack)-1]
	}

	return parent, nil
}

// validateFrame validates a JSON-LD frame, returning an error if the frame is invalid.
func validateFrame(frame interface{}) error {

	valid := true
	if frameList, isList := frame.([]interface{}); isList {
		if len(frameList) > 1 {
			valid = false
		} else if len(frameList) == 1 {
			frame = frameList[0]
			if _, isMap := frame.(map[string]interface{}); !isMap {
				valid = false
			}
		} else {
			// TODO: other JSON-LD implementations don't cater for this case (frame==[]). Investigate.
			return nil
		}

	} else if _, isMap := frame.(map[string]interface{}); !isMap {
		valid = false
	}

	if !valid {
		return NewJsonLdError(InvalidFrame, "Invalid JSON-LD syntax; a JSON-LD frame must be a single object")
	}

	frameMap := frame.(map[string]interface{})

	if id, hasID := frameMap["@id"]; hasID {
		for _, idVal := range Arrayify(id) {
			if _, isMap := idVal.(map[string]interface{}); isMap {
				continue
			}
			if strings.HasPrefix(idVal.(string), "_:") {
				return NewJsonLdError(InvalidFrame,
					fmt.Sprintf("Invalid JSON-LD frame syntax; invalid value of @id: %v", id))
			}
		}
	}

	if t, hasType := frameMap["@type"]; hasType {
		for _, typeVal := range Arrayify(t) {
			if _, isMap := typeVal.(map[string]interface{}); isMap {
				continue
			}
			if strings.HasPrefix(typeVal.(string), "_:") {
				return NewJsonLdError(InvalidFrame,
					fmt.Sprintf("Invalid JSON-LD frame syntax; invalid value of @type: %v", t))
			}
		}
	}

	return nil
}

func getFrameValue(frame map[string]interface{}, name string) interface{} {
	value := frame[name]
	switch v := value.(type) {
	case []interface{}:
		if len(v) > 0 {
			value = v[0]
		}
	case map[string]interface{}:
		if valueVal, containsValue := v["@value"]; containsValue {
			value = valueVal
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
	} else if value == "true" {
		return true
	} else if value == "false" {
		return false
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
			return EmbedOnce, nil
		} else {
			return EmbedNever, nil
		}
	}
	if embedVal, isEmbed := value.(Embed); isEmbed {
		return embedVal, nil
	}
	if stringVal, isString := value.(string); isString {
		switch stringVal {
		case "@always":
			return EmbedAlways, nil
		case "@last":
			return EmbedLast, nil
		case "@never":
			return EmbedNever, nil
		case "@once":
			return EmbedOnce, nil
		default:
			return EmbedOnce, NewJsonLdError(InvalidEmbedValue,
				fmt.Sprintf("Invalid JSON-LD frame syntax; invalid value of @embed: %s", stringVal))
		}
	}
	return EmbedOnce, NewJsonLdError(InvalidEmbedValue, "Invalid JSON-LD frame syntax; invalid value of @embed")
}

// removeEmbed removes an existing embed with the given id.
func removeEmbed(state *FramingContext, id string) {
	// get existing embed
	links := state.uniqueEmbeds[state.graph]
	embed := links[id]
	parent := embed.parent
	property := embed.property

	// create reference to replace embed
	subject := map[string]interface{}{
		"@id": id,
	}

	// remove existing embed
	if _, isArray := parent.([]interface{}); isArray {
		// replace subject with reference
		newVals := make([]interface{}, 0)
		parentMap := parent.(map[string]interface{})
		oldvals := parentMap[property].([]interface{})
		for _, v := range oldvals {
			vMap, isMap := v.(map[string]interface{})
			if isMap && vMap["@id"] == id {
				newVals = append(newVals, subject)
			} else {
				newVals = append(newVals, v)
			}
		}
		parentMap[property] = newVals
	} else {
		// replace subject with reference
		parentMap := parent.(map[string]interface{})
		_, useArray := parentMap[property]
		RemoveValue(parentMap, property, subject, useArray)
		AddValue(parentMap, property, subject, useArray, false, true, false)
	}
	// recursively remove dependent dangling embeds
	removeDependents(links, id)
}

// removeDependents recursively removes dependent dangling embeds.
func removeDependents(embeds map[string]*EmbedNode, id string) {
	// get embed keys as a separate array to enable deleting keys in map
	for idDep, e := range embeds {
		if e.parent == nil {
			continue
		}
		p, isMap := e.parent.(map[string]interface{})
		if !isMap {
			continue
		}
		if pid, hasID := p["@id"]; hasID {
			if id == pid.(string) {
				delete(embeds, idDep)
				removeDependents(embeds, idDep)
			}
		}
	}
}

// FilterSubjects returns a map of all of the nodes that match a parsed frame.
func FilterSubjects(state *FramingContext, subjects []string, frame map[string]interface{}, requireAll bool) (map[string]interface{}, error) {
	rval := make(map[string]interface{})
	for _, id := range subjects {
		// id, elementVal
		elementVal := state.graphMap[state.graph].(map[string]interface{})[id]
		element, _ := elementVal.(map[string]interface{})
		if element != nil {
			res, err := FilterSubject(state, element, frame, requireAll)
			if res {
				if err != nil {
					return nil, err
				}
				rval[id] = element
			}
		}
	}
	return rval, nil
}

// FilterSubject returns true if the given node matches the given frame.
//
// Matches either based on explicit type inclusion where the node has any
// type listed in the frame. If the frame has empty types defined matches
// nodes not having a @type. If the frame has a type of {} defined matches
// nodes having any type defined.
//
// Otherwise, does duck typing, where the node must have all of the
// properties defined in the frame.
//
// Algorithm steps from: https://www.w3.org/TR/json-ld-framing/#frame-matching
func FilterSubject(state *FramingContext, subject map[string]interface{}, frame map[string]interface{}, requireAll bool) (bool, error) {
	// check ducktype
	wildcard := true
	matchesSome := false
	matchThis := false

	// 1
	// node matches if frame has no properties.
	if isEmptyObject(frame) {
		return true, nil
	}

	// 2
	// For the values of each property from frame in node:
	for _, k := range GetOrderedKeys(frame) {
		matchThis = false
		v := frame[k]

		var nodeValues []interface{}
		if kVal, found := subject[k]; found {
			nodeValues = Arrayify(kVal)
		} else {
			nodeValues = make([]interface{}, 0)
		}

		vList, _ := v.([]interface{})
		vMap, _ := v.(map[string]interface{})
		isEmpty := (len(vList) + len(vMap)) == 0

		if IsKeyword(k) {
			// skip non-@id and non-@type
			if k != "@id" && k != "@type" {
				continue
			}
			wildcard = true

			// 2.1
			// If property is @id:
			// check @id for a specific @id value
			if k == "@id" {
				// 2.1.1
				// property matches if the @id property in frame includes any IRI in values.
				//
				// if @id is not a wildcard and is not empty, then match
				// or not on specific value
				frameID := Arrayify(frame["@id"])
				if len(frameID) > 0 {
					_, isString := frameID[0].(string)
					if !isEmptyObject(frameID[0]) || isString {
						idMatches := inArray(nodeValues[0], frameID)
						// requireAll needs to be set twice to prevent digressions in
						// tests #t0012, #tg005, #tg006, #tg008.
						if requireAll && frame["@requireAll"] != nil {
							if !idMatches {
								return false, nil
							}
							matchesSome = true
							continue
						}
						return idMatches, nil
					}
				}
				// @id: {} wildcard — counts as a match but must not skip matchesSome
				// so that other properties with @default still produce a match overall
				matchesSome = true
				matchThis = true
				continue
			}

			// 2.2
			// Otherwise, if property is @type:

			// check @type (object value means 'any' type, fall through to
			// ducktyping)
			if k == "@type" {
				if isEmpty {
					if len(nodeValues) > 0 {
						// don't match on no @type
						return false, nil
					}
					matchThis = true
				} else {
					frameType := frame["@type"].([]interface{})
					if isEmptyObject(frameType[0]) {
						matchThis = len(nodeValues) > 0
					} else {
						// match on a specific @type
						r := make([]interface{}, 0)
						for _, tv := range nodeValues {
							for _, tf := range frameType {
								if tv == tf {
									r = append(r, tv)
									// break early, as we just need one element to succeed
									break
								}
							}
						}
						typeMatches := len(r) > 0
						if requireAll && frame["@requireAll"] != nil {
							if !typeMatches {
								return false, nil
							}
							matchesSome = true
							continue
						}
						return typeMatches, nil
					}
				}
			}

		}
		// force a copy of this frame entry so it can be manipulated
		var thisFrame interface{}
		if x := Arrayify(frame[k]); len(x) > 0 {
			thisFrame = x[0]
		}
		hasDefault := false
		if thisFrame != nil {
			if err := validateFrame(thisFrame); err != nil {
				return false, err
			}
			_, hasDefault = thisFrame.(map[string]interface{})["@default"]
		}

		// no longer a wildcard pattern if frame has any non-keyword
		// properties
		wildcard = false

		// 2.5 ?
		// property matches if values is empty, or non existent,
		// the value of property in frame is a map containing only the @default entry with any value,
		// and any other property in node has a non-default match.

		// skip, but allow match if node has no value for property, and
		// frame has a default value
		if len(nodeValues) == 0 && hasDefault {
			continue
		}

		// if frame value is empty, don't match if subject has any value
		if len(nodeValues) > 0 && isEmpty {
			return false, nil
		}

		if thisFrame == nil {
			// 2.6
			// node does not match if values is not empty and the value of
			// property in frame is match none.
			if len(nodeValues) > 0 {
				return false, nil
			}
			matchThis = true
		} else if IsValue(thisFrame) {
			// 2.8
			// Otherwise, if the value of property in frame is a value pattern (value pattern):
			// property matching is determined using the Value matching algorithm.
			// 2.8 as value pattern: must be checked before the generic isMap wildcard,
			// because value objects are also maps.
			for _, nv := range nodeValues {
				if valueMatch(thisFrame.(map[string]interface{}), nv.(map[string]interface{})) {
					matchThis = true
					break
				}
			}
		} else if IsList(thisFrame) {
			// 2.8 as list pattern: must be checked before the generic isMap wildcard,
			// because list objects are also maps.
			listValue := thisFrame.(map[string]interface{})["@list"].([]interface{})[0]
			if len(nodeValues) > 0 && IsList(nodeValues[0]) {
				nodeListValues := nodeValues[0].(map[string]interface{})["@list"]

				if IsValue(listValue) {
					for _, lv := range nodeListValues.([]interface{}) {
						if valueMatch(listValue.(map[string]interface{}), lv.(map[string]interface{})) {
							matchThis = true
							break
						}
					}
				} else if IsSubject(listValue) || IsSubjectReference(listValue) {
					for _, lv := range nodeListValues.([]interface{}) {
						if nodeMatch(state, listValue.(map[string]interface{}), lv.(map[string]interface{}), requireAll) {
							matchThis = true
							break
						}
					}
				}
			}
		} else if IsSubjectReference(thisFrame) {
			// specific node reference match: frame value is {"@id": "some-iri"},
			// so only match nodeValues whose @id equals that IRI.
			frameRefID := thisFrame.(map[string]interface{})["@id"]
			for _, nv := range nodeValues {
				if nvMap, ok := nv.(map[string]interface{}); ok {
					if nvMap["@id"] == frameRefID {
						matchThis = true
						break
					}
				}
			}
		} else if _, isMap := thisFrame.(map[string]interface{}); isMap {
			// 2.7
			// node matches if values is not empty and the value of
			// property in frame is wildcard
			matchThis = len(nodeValues) > 0
		}

		// 2
		// If requireAll is true, node matches if all properties (property) in frame match any of the following conditions.
		// Or, if requireAll is false, if any of the properties (property) in frame match any of the following conditions.
		if !matchThis && requireAll {
			return false, nil
		}

		matchesSome = matchesSome || matchThis
	}

	return wildcard || matchesSome, nil
}

// addFrameOutput adds framing output to the given parent.
// parent: the parent to add to.
// property: the parent property.
// output: the output to add.
func addFrameOutput(parent interface{}, property string, output interface{}) interface{} {
	if parentMap, isMap := parent.(map[string]interface{}); isMap {
		AddValue(parentMap, property, output, true, false, true, false)
		return parentMap
	}

	return append(parent.([]interface{}), output)
}

func nodeMatch(state *FramingContext, pattern, value map[string]interface{}, requireAll bool) bool {
	id, hasID := value["@id"]
	if !hasID {
		return false
	}
	nodeObject, found := state.subjects[id.(string)]
	if !found {
		return false
	}
	ok, _ := FilterSubject(state, nodeObject.(map[string]interface{}), pattern, requireAll)
	return ok
}

// valueMatch returns true if it is a value and matches the value pattern
//
//   - `pattern` is empty
//   - @values are the same, or `pattern[@value]` is a wildcard,
//   - @types are the same or `value[@type]` is not None
//     and `pattern[@type]` is `{}` or `value[@type]` is None
//     and `pattern[@type]` is None or `[]`, and
//   - @languages are the same or `value[@language]` is not None
//     and `pattern[@language]` is `{}`, or `value[@language]` is None
//     and `pattern[@language]` is None or `[]`
//
// https://www.w3.org/TR/json-ld-framing/#value-matching
func valueMatch(pattern, value map[string]interface{}) bool {
	// 2.
	// Let v2, t2, and l2 be the values of @value, @type, and @language in value pattern, or null
	// if none exists, where string values of @language are normalized to lower case..

	v2v := pattern["@value"]
	t2v := pattern["@type"]
	l2v := pattern["@language"]

	// 3.
	// Value matches pattern when pattern is wildcard, or
	if isEmptyObject(pattern) || (v2v == nil && t2v == nil && l2v == nil) {
		return true
	}

	var v2 []interface{}
	if v2v != nil {
		v2 = Arrayify(v2v)
	}
	var t2 []interface{}
	if t2v != nil {
		t2 = Arrayify(t2v)
	}
	var l2 []interface{}
	if l2v != nil {
		l2 = Arrayify(l2v)
	}

	// 1.
	// Let v1, t1, and l1 be the values of @value, @type, and @language in value, or null if none
	// exists, where values of @language are normalized to lower case..
	v1 := value["@value"]
	t1 := value["@type"]
	l1 := value["@language"]

	// FIXME: replace isEmptyObject with isWildcard

	// 3.1.
	// v1 is in v2, or v1 is not null and v2 is wildcard, and
	if !(inArray(v1, v2) || (len(v2) > 0 && isEmptyObject(v2[0]))) {
		return false
	}

	// 3.2
	// t1 is in t2, or t1 is not null and t2 is wildcard, or null, or t1 is null and t2 is null or match none, and
	if !((t1 == nil && len(t2) == 0) || (inArray(t1, t2)) || (t1 != nil && len(t2) > 0 && isEmptyObject(t2[0]))) {
		return false
	}

	// 3.3
	// l1 is in l2, or l1 is not null and l2 is wildcard, or null, or l1 is null and l2 is null or match none.
	if !((l1 == nil && len(l2) == 0) || (inArray(l1, l2)) || (l1 != nil && len(l2) > 0 && isEmptyObject(l2[0]))) {
		return false
	}
	return true
}
