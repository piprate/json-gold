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
	"regexp"
	"sort"
	"strings"
)

// Context represents a JSON-LD context and provides easy access to specific
// keys and operations.
type Context struct {
	values          map[string]interface{}
	options         *JsonLdOptions
	termDefinitions map[string]interface{}
	inverse         map[string]interface{}
}

// NewContext creates and returns a new Context object.
func NewContext(values map[string]interface{}, options *JsonLdOptions) *Context {
	if options == nil {
		options = NewJsonLdOptions("")
	}

	context := &Context{
		values:          make(map[string]interface{}),
		options:         options,
		termDefinitions: make(map[string]interface{}),
	}

	if values != nil {
		for k, v := range values {
			context.values[k] = v
		}
	}

	context.values["@base"] = options.Base
	context.values["processingMode"] = options.ProcessingMode

	return context
}

// CopyContext creates a full copy of the given context.
func CopyContext(ctx *Context) *Context {
	context := NewContext(ctx.values, ctx.options)

	for k, v := range ctx.termDefinitions {
		context.termDefinitions[k] = v
	}

	// do not copy c.inverse, because it will be regenerated

	return context
}

// Parse processes a local context, retrieving any URLs as necessary, and
// returns a new active context.
// Refer to http://www.w3.org/TR/json-ld-api/#context-processing-algorithms for details
// TODO pyLD is doing a fair bit more in process_context(self, active_ctx, local_ctx, options)
// than just parsing the context. In particular, we need to check if additional logic is required
// to load remote scoped contexts.
func (c *Context) Parse(localContext interface{}) (*Context, error) {
	return c.parse(localContext, make([]string, 0), false)
}

// parse processes a local context, retrieving any URLs as necessary, and
// returns a new active context.
//
// If parsingARemoteContext is true, localContext represents a remote context
// that has been parsed and sent into this method. This must be set to know
// whether to propagate the @base key from the context to the result.
func (c *Context) parse(localContext interface{}, remoteContexts []string, parsingARemoteContext bool) (*Context, error) {
	// 1. Initialize result to the result of cloning active context.
	result := CopyContext(c)

	// 3)
	for _, context := range Arrayify(localContext) {
		// 3.1)
		if context == nil {
			result = NewContext(nil, c.options)
			continue
		}

		var contextMap map[string]interface{}

		switch ctx := context.(type) {
		case *Context:
			result = ctx
		// 3.2)
		case string:
			uri := Resolve(result.values["@base"].(string), ctx)
			// 3.2.2
			for _, remoteCtx := range remoteContexts {
				if remoteCtx == uri {
					return nil, NewJsonLdError(RecursiveContextInclusion, uri)
				}
			}
			remoteContexts = append(remoteContexts, uri)

			// 3.2.3: Dereference context
			rd, err := c.options.DocumentLoader.LoadDocument(uri)
			if err != nil {
				return nil, NewJsonLdError(LoadingRemoteContextFailed,
					fmt.Sprintf("Dereferencing a URL did not result in a valid JSON-LD context: %s", uri))
			}
			remoteContextMap, isMap := rd.Document.(map[string]interface{})
			context, hasContextKey := remoteContextMap["@context"]
			if !isMap || !hasContextKey {
				// If the dereferenced document has no top-level JSON object
				// with an @context member
				return nil, NewJsonLdError(InvalidRemoteContext, context)
			}

			// 3.2.4
			resultRef, err := result.parse(context, remoteContexts, true)
			if err != nil {
				return nil, err
			}
			result = resultRef
			// 3.2.5
			continue
		case map[string]interface{}:
			contextMap = ctx
		default:
			// 3.3
			return nil, NewJsonLdError(InvalidLocalContext, context)
		}

		pm, hasProcessingMode := c.values["processingMode"]

		if versionValue, versionPresent := contextMap["@version"]; versionPresent {
			if versionValue != 1.1 {
				return nil, NewJsonLdError(InvalidVersionValue, fmt.Sprintf("unsupported JSON-LD version: %s", versionValue))
			}
			if hasProcessingMode {
				if pm.(string) == JsonLd_1_0 {
					return nil, NewJsonLdError(ProcessingModeConflict, fmt.Sprintf("@version: %v not compatible with %s", versionValue, pm))
				}
			}
			result.values["processingMode"] = JsonLd_1_1
			result.values["@version"] = versionValue
		} else if !hasProcessingMode {
			// if not set explicitly, set processingMode to "json-ld-1.0"
			result.values["processingMode"] = JsonLd_1_0
		} else {
			result.values["processingMode"] = pm
		}

		// 3.4
		baseValue, basePresent := contextMap["@base"]
		if !parsingARemoteContext && basePresent {
			if baseValue == nil {
				delete(result.values, "@base")
			} else if baseString, isString := baseValue.(string); isString {
				if IsAbsoluteIri(baseString) {
					result.values["@base"] = baseValue
				} else {
					baseURI := result.values["@base"].(string)
					if !IsAbsoluteIri(baseURI) {
						return nil, NewJsonLdError(InvalidBaseIRI, baseURI)
					}
					result.values["@base"] = Resolve(baseURI, baseString)
				}
			} else {
				return nil, NewJsonLdError(InvalidBaseIRI, "the value of @base in a @context must be a string or null")
			}
		}

		// 3.5
		if vocabValue, vocabPresent := contextMap["@vocab"]; vocabPresent {
			if vocabValue == nil {
				delete(result.values, "@vocab")
			} else if vocabString, isString := vocabValue.(string); isString {
				if IsAbsoluteIri(vocabString) {
					result.values["@vocab"] = vocabValue
				} else if vocabString == "" {
					if baseVal, hasBase := result.values["@base"]; hasBase {
						result.values["@vocab"] = baseVal
					} else {
						return nil, NewJsonLdError(InvalidVocabMapping, "@vocab is empty but @base is not specified")
					}
				} else {
					return nil, NewJsonLdError(InvalidVocabMapping, "@vocab must be an absolute IRI")
				}
			} else {
				return nil, NewJsonLdError(InvalidVocabMapping, "@vocab must be a string or null")
			}
		}

		// 3.6
		if languageValue, languagePresent := contextMap["@language"]; languagePresent {
			if languageValue == nil {
				delete(result.values, "@language")
			} else if languageString, isString := languageValue.(string); isString {
				result.values["@language"] = strings.ToLower(languageString)
			} else {
				return nil, NewJsonLdError(InvalidDefaultLanguage, languageValue)
			}
		}

		// 3.7
		defined := make(map[string]bool)

		for key := range contextMap {
			if key == "@base" || key == "@vocab" || key == "@language" || key == "@version" {
				continue
			}
			if err := result.createTermDefinition(contextMap, key, defined); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

// CompactValue performs value compaction on an object with @value or @id as the only property.
// See http://www.w3.org/TR/json-ld-api/#value-compaction
func (c *Context) CompactValue(activeProperty string, value map[string]interface{}) interface{} {
	propType, _ := c.GetTermDefinition(activeProperty)["@type"]

	if IsValue(value) {
		language := c.GetLanguageMapping(activeProperty)
		isIndexContainer := c.HasContainerMapping(activeProperty, "@index")

		// whether or not the value has an @index that must be preserved
		_, hasIndex := value["@index"]
		typeVal, hasType := value["@type"]
		languageVal, hasLanguage := value["@language"]

		preserveIndex := hasIndex && !isIndexContainer

		// if there's no @index to preserve
		if !preserveIndex {
			// matching @type or @language specified in context, compact
			if (hasType && typeVal == propType) || (hasLanguage && languageVal == language) {
				return value["@value"]
			}
		}

		// return just the value of @value if all are true:
		// 1. @value is the only key or @index isn't being preserved
		// 2. there is no default language or @value is not a string or
		// the key has a mapping with a null @language
		keyCount := len(value)
		isValueOnlyKey := keyCount == 1 || (keyCount == 2 && hasIndex && !preserveIndex)
		_, hasDefaultLanguage := c.values["@language"]
		_, isValueString := value["@value"].(string)
		langEntry, hasLanguageEntry := c.GetTermDefinition(activeProperty)["@language"]
		hasNullMapping := c.GetTermDefinition(activeProperty) != nil && hasLanguageEntry && langEntry == nil
		if isValueOnlyKey && (!hasDefaultLanguage || !isValueString || hasNullMapping) {
			return value["@value"]
		}

		rval := make(map[string]interface{})

		// preserve @index
		if preserveIndex {
			indexAlias := c.CompactIri("@index", nil, false, false)
			rval[indexAlias] = value["@index"]
		}

		// compact @type IRI
		if hasType {
			typeAlias := c.CompactIri("@type", nil, false, false)
			rval[typeAlias] = c.CompactIri(typeVal.(string), nil, true, false)
		} else if hasLanguage {
			// alias @language
			languageAlias := c.CompactIri("@language", nil, false, false)
			rval[languageAlias] = languageVal
		}

		// alias @value
		valueAlias := c.CompactIri("@value", nil, false, false)
		rval[valueAlias] = value["@value"]

		return rval
	} else {
		// value is a subject reference
		expandedProperty, err := c.ExpandIri(activeProperty, false, true, nil, nil)
		if err != nil {
			return err
		}
		compacted := c.CompactIri(value["@id"].(string), nil, propType == "@vocab", false)

		// compact to scalar
		if propType == "@id" || propType == "@vocab" || expandedProperty == "@graph" {
			return compacted
		}

		return map[string]interface{}{
			c.CompactIri("@id", nil, false, false): compacted,
		}
	}
}

// processingMode returns true if the given version is compatible with the current processing mode
func (c *Context) processingMode(version float64) bool {
	mode, hasMode := c.values["processingMode"]
	if version >= 1.1 {
		if hasMode {
			return mode.(string) >= fmt.Sprintf("json-ld-%v", version)
		} else {
			return false
		}
	} else {
		if hasMode {
			return mode.(string) == JsonLd_1_0
		} else {
			return true
		}
	}
}

// createTermDefinition creates a term definition in the active context
// for a term being processed in a local context as described in
// http://www.w3.org/TR/json-ld-api/#create-term-definition
func (c *Context) createTermDefinition(context map[string]interface{}, term string,
	defined map[string]bool) error {
	if definedValue, inDefined := defined[term]; inDefined {
		if definedValue {
			return nil
		}
		return NewJsonLdError(CyclicIRIMapping, term)
	}

	defined[term] = false

	if IsKeyword(term) {
		return NewJsonLdError(KeywordRedefinition, term)
	}

	delete(c.termDefinitions, term)
	value := context[term]
	mapValue, isMap := value.(map[string]interface{})
	idValue, hasID := mapValue["@id"]
	if value == nil || (isMap && hasID && idValue == nil) {
		c.termDefinitions[term] = nil
		defined[term] = true
		return nil
	}

	simpleTerm := false
	if _, isString := value.(string); isString {
		mapValue = map[string]interface{}{"@id": value}
		simpleTerm = true
		isMap = true
	}

	if !isMap {
		return NewJsonLdError(InvalidTermDefinition, value)
	}

	// casting the value so it doesn't have to be done below everytime
	val := mapValue

	// 9) create a new term definition
	var definition = make(map[string]interface{})

	// make sure term definition only has expected keywords
	validKeys := map[string]bool{
		"@container": true,
		"@id":        true,
		"@language":  true,
		"@reverse":   true,
		"@type":      true,
	}
	if c.processingMode(1.1) {
		validKeys["@context"] = true
		validKeys["@nest"] = true
		validKeys["@prefix"] = true
	}
	for k := range val {
		if _, isValid := validKeys[k]; !isValid {
			return NewJsonLdError(InvalidTermDefinition, fmt.Sprintf("a term definition must not contain %s", k))
		}
	}

	// always compute whether term has a colon as an optimization for _compact_iri
	termHasColon := strings.Contains(term, ":")

	definition["@reverse"] = false

	// 11)
	if reverseValue, present := val["@reverse"]; present {
		if _, idPresent := val["@id"]; idPresent {
			return NewJsonLdError(InvalidReverseProperty, "an @reverse term definition must not contain @id.")
		}
		if _, nestPresent := val["@nest"]; nestPresent {
			return NewJsonLdError(InvalidReverseProperty, "an @reverse term definition must not contain @nest.")
		}
		reverseStr, isString := reverseValue.(string)
		if !isString {
			return NewJsonLdError(InvalidIRIMapping,
				fmt.Sprintf("expected string for @reverse value. got %v", reverseValue))
		}
		id, err := c.ExpandIri(reverseStr, false, true, context, defined)
		if err != nil {
			return err
		}
		if !IsAbsoluteIri(id) {
			return NewJsonLdError(InvalidIRIMapping, fmt.Sprintf(
				"@context @reverse value must be an absolute IRI or a blank node identifier, got %s", id))
		}
		definition["@id"] = id
		definition["@reverse"] = true
	} else if idValue, hasID := val["@id"]; hasID { // 13)
		idStr, isString := idValue.(string)
		if !isString {
			return NewJsonLdError(InvalidIRIMapping, "expected value of @id to be a string")
		}

		if term != idStr {
			res, err := c.ExpandIri(idStr, false, true, context, defined)
			if err != nil {
				return err
			}
			if IsKeyword(res) || IsAbsoluteIri(res) {
				if res == "@context" {
					return NewJsonLdError(InvalidKeywordAlias, "cannot alias @context")
				}
				definition["@id"] = res

				var regexExp = regexp.MustCompile(".*[:/\\?#\\[\\]@]$")
				// NOTE: definition["_prefix"] is implemented in Python and JS libraries as follows:
				//
				// definition["_prefix"] = !termHasColon && regexExp.Match([]byte(res)) && (simpleTerm || c.processingMode(1.0))
				//
				// but the test https://json-ld.org/test-suite/tests/compact-manifest.jsonld#t0038 fails. TODO investigate
				definition["_prefix"] = !termHasColon && (regexExp.Match([]byte(res)) && simpleTerm || c.processingMode(1.0))
			} else {
				return NewJsonLdError(InvalidIRIMapping,
					"resulting IRI mapping should be a keyword, absolute IRI or blank node")
			}
		}
		// 14)
	}

	if _, hasID := definition["@id"]; !hasID {
		if colIndex := strings.Index(term, ":"); colIndex >= 0 {
			prefix := term[0:colIndex]
			if _, containsPrefix := context[prefix]; containsPrefix {
				if err := c.createTermDefinition(context, prefix, defined); err != nil {
					return err
				}
			}
			if termDef, hasTermDef := c.termDefinitions[prefix]; hasTermDef {
				termDefMap, _ := termDef.(map[string]interface{})
				suffix := term[colIndex+1:]
				definition["@id"] = termDefMap["@id"].(string) + suffix
			} else {
				definition["@id"] = term
			}
			// 15)
		} else if vocabValue, containsVocab := c.values["@vocab"]; containsVocab {
			definition["@id"] = vocabValue.(string) + term
		} else {
			return NewJsonLdError(InvalidIRIMapping, "relative term definition without vocab mapping")
		}
	}

	defined[term] = true

	// 10)
	if typeValue, present := val["@type"]; present {
		typeStr, isString := typeValue.(string)
		if !isString {
			return NewJsonLdError(InvalidTypeMapping, typeValue)
		}
		if typeStr != "@id" && typeStr != "@vocab" {
			// expand @type to full IRI
			var err error
			typeStr, err = c.ExpandIri(typeStr, false, true, context, defined)
			if err != nil {
				if err.(*JsonLdError).Code != InvalidIRIMapping {
					return err
				}
				return NewJsonLdError(InvalidTypeMapping, typeStr)
			}
			if !IsAbsoluteIri(typeStr) {
				return NewJsonLdError(InvalidTypeMapping, "an @context @type value must be an absolute IRI")
			}
			if strings.HasPrefix(typeStr, "_:") {
				return NewJsonLdError(InvalidTypeMapping, "an @context @type values must be an IRI, not a blank node identifier")
			}
		}

		// add @type to mapping
		definition["@type"] = typeStr
	}

	// 16)
	if containerVal, hasContainer := val["@container"]; hasContainer {
		containerArray, isArray := containerVal.([]interface{})
		var container []string
		containerValueMap := make(map[string]bool)
		if isArray {
			container = make([]string, 0)
			for _, c := range containerArray {
				container = append(container, c.(string))
				containerValueMap[c.(string)] = true
			}
		} else {
			container = []string{containerVal.(string)}
			containerValueMap[containerVal.(string)] = true
		}

		validContainers := map[string]bool{
			"@list":     true,
			"@set":      true,
			"@index":    true,
			"@language": true,
		}
		if c.processingMode(1.1) {
			validContainers["@graph"] = true
			validContainers["@id"] = true
			validContainers["@type"] = true

			// check container length

			if _, hasList := containerValueMap["@list"]; hasList && len(container) != 1 {
				return NewJsonLdError(InvalidContainerMapping,
					"@context @container with @graph must have no other values other than @id, @index, and @set")
			}

			if _, hasGraph := containerValueMap["@graph"]; hasGraph {
				validKeys := map[string]bool{
					"@graph": true,
					"@id":    true,
					"@index": true,
					"@set":   true,
				}
				for key := range containerValueMap {
					if _, found := validKeys[key]; !found {
						return NewJsonLdError(InvalidContainerMapping,
							"@context @container with @list must have no other values.")
					}
				}
			} else {
				maxLen := 1
				if _, hasSet := containerValueMap["@set"]; hasSet {
					maxLen = 2
				}
				if len(container) > maxLen {
					return NewJsonLdError(InvalidContainerMapping, "@set can only be combined with one more type")
				}
			}
		} else {
			// json-ld-1.0
			if _, isString := containerVal.(string); !isString {
				return NewJsonLdError(InvalidContainerMapping, "@container must be a string")
			}
		}

		// check against valid containers
		for _, v := range container {
			if _, isValidContainer := validContainers[v]; !isValidContainer {
				allowedValues := make([]string, 0)
				for k := range validContainers {
					allowedValues = append(allowedValues, k)
				}
				return NewJsonLdError(InvalidContainerMapping, fmt.Sprintf(
					"@context @container value must be one of the following: %q", allowedValues))
			}
		}

		// @set not allowed with @list
		_, hasSet := containerValueMap["@set"]
		_, hasList := containerValueMap["@list"]
		if hasSet && hasList {
			return NewJsonLdError(InvalidContainerMapping, "@set not allowed with @list")
		}

		if reverseVal, hasReverse := definition["@reverse"]; hasReverse && reverseVal.(bool) {

			for key := range containerValueMap {
				if key != "@index" && key != "@set" {
					return NewJsonLdError(InvalidReverseProperty,
						"@context @container value for an @reverse type definition must be @index or @set")
				}
			}
		}

		definition["@container"] = container
	}

	// scoped contexts
	if ctxVal, hasCtx := val["@context"]; hasCtx {
		definition["@context"] = ctxVal
	}

	// 17)
	_, hasType := val["@type"]
	if languageVal, hasLanguage := val["@language"]; hasLanguage && !hasType {
		if language, isString := languageVal.(string); isString {
			definition["@language"] = strings.ToLower(language)
		} else if languageVal == nil {
			definition["@language"] = nil
		} else {
			return NewJsonLdError(InvalidLanguageMapping, "@language must be a string or null")
		}
	}

	// term may be used as prefix
	if prefixVal, hasPrefix := val["@prefix"]; hasPrefix {
		if termHasColon {
			return NewJsonLdError(InvalidTermDefinition, "@context @prefix used on a compact IRI term")
		}
		prefix, isBool := prefixVal.(bool)
		if !isBool {
			return NewJsonLdError(InvalidPrefixValue, "@context value for @prefix must be boolean")
		}
		definition["_prefix"] = prefix
	}

	// nesting
	if nestVal, hasNest := val["@nest"]; hasNest {
		nest, isString := nestVal.(string)
		if !isString || (nest != "@nest" && nest[0] == '@') {
			return NewJsonLdError(InvalidNestValue,
				"@context @nest value must be a string which is not a keyword other than @nest")
		}
		definition["@nest"] = nest
	}

	// disallow aliasing @context and @preserve
	id := definition["@id"]
	if id == "@context" || id == "@preserve" {
		return NewJsonLdError(InvalidKeywordAlias, "@context and @preserve cannot be aliased")
	}

	// 18)
	c.termDefinitions[term] = definition

	return nil
}

// ExpandIri expands a string value to a full IRI.
//
// The string may be a term, a prefix, a relative IRI, or an absolute IRI.
// The associated absolute IRI will be returned.
//
// value: the string value to expand.
// relative: true to resolve IRIs against the base IRI, false not to.
// vocab: true to concatenate after @vocab, false not to.
// context: the local context being processed (only given if called during context processing).
// defined: a map for tracking cycles in context definitions (only given if called during context processing).
func (c *Context) ExpandIri(value string, relative bool, vocab bool, context map[string]interface{},
	defined map[string]bool) (string, error) {
	// 1)
	if IsKeyword(value) {
		return value, nil
	}
	// 2)
	if context != nil {
		if _, containsKey := context[value]; containsKey && !defined[value] {
			if err := c.createTermDefinition(context, value, defined); err != nil {
				return "", err
			}
		}
	}
	// 3)
	if termDef, hasTermDef := c.termDefinitions[value]; vocab && hasTermDef {
		termDefMap, isMap := termDef.(map[string]interface{})
		if isMap && termDefMap != nil {
			return termDefMap["@id"].(string), nil
		}

		return "", nil
	}
	// 4)
	colIndex := strings.Index(value, ":")
	if colIndex >= 0 {
		// 4.1)
		prefix := value[0:colIndex]
		suffix := value[colIndex+1:]
		// 4.2)
		if prefix == "_" || strings.HasPrefix(suffix, "//") {
			return value, nil
		}
		// 4.3)
		if context != nil {
			if _, containsPrefix := context[prefix]; containsPrefix && !defined[prefix] {
				if err := c.createTermDefinition(context, prefix, defined); err != nil {
					return "", err
				}
			}
		}
		// 4.4)
		if termDef, hasPrefix := c.termDefinitions[prefix]; hasPrefix {
			termDefMap := termDef.(map[string]interface{})
			return termDefMap["@id"].(string) + suffix, nil
		}
		// 4.5)
		return value, nil
	}
	// 5)
	if vocabValue, containsVocab := c.values["@vocab"]; vocab && containsVocab {
		return vocabValue.(string) + value, nil
	} else if relative {
		// 6)
		baseValue, hasBase := c.values["@base"]
		var base string
		if hasBase {
			base = baseValue.(string)
		} else {
			base = ""
		}
		return Resolve(base, value), nil
	} else if context != nil && IsRelativeIri(value) {
		return "", NewJsonLdError(InvalidIRIMapping, "not an absolute IRI: "+value)
	}
	// 7)
	return value, nil
}

// CompactIri compacts an IRI or keyword into a term or CURIE if it can be.
// If the IRI has an associated value it may be passed.
//
// iri: the IRI to compact.
// value: the value to check or None.
// relativeToVocab: true to compact using @vocab if available, false not to.
// reverse: true if a reverse property is being compacted, false if not.
//
// Returns the compacted term, prefix, keyword alias, or original IRI.
func (c *Context) CompactIri(iri string, value interface{}, relativeToVocab bool, reverse bool) string {
	// 1)
	if iri == "" {
		return ""
	}

	inverseCtx := c.GetInverse()

	// term is a keyword, force relativeToVocab to True
	if IsKeyword(iri) {
		// look for an alias
		if v, found := inverseCtx[iri]; found {
			if v, found = v.(map[string]interface{})["@none"]; found {
				if v, found = v.(map[string]interface{})["@type"]; found {
					if v, found = v.(map[string]interface{})["@none"]; found {
						return v.(string)
					}
				}
			}
		}
		relativeToVocab = true
	}

	// 2)
	if relativeToVocab {
		if _, containsIRI := inverseCtx[iri]; containsIRI {
			// 2.1)
			// TODO see pyLD, defaultLanguage is never used. It looks like a bug in their implementation.
			//defaultLanguage := "@none"
			//langVal, hasLang := c.values["@language"]
			//if hasLang {
			//	defaultLanguage = langVal.(string)
			//}

			// 2.2)

			// prefer @index if available in value
			containers := make([]string, 0)

			valueMap, isObject := value.(map[string]interface{})
			if isObject {

				_, hasIndex := valueMap["@index"]
				_, hasGraph := valueMap["@graph"]
				if hasIndex && !hasGraph {
					containers = append(containers, "@index", "@index@set")
				}

				// if value is a preserve object, use its value
				if pv, hasPreserve := valueMap["@preserve"]; hasPreserve {
					value = pv.([]interface{})[0]
					valueMap, isObject = value.(map[string]interface{})
				}
			}

			// prefer most specific container including @graph
			if IsGraph(value) {

				_, hasIndex := valueMap["@index"]
				_, hasID := valueMap["@id"]

				if hasIndex {
					containers = append(containers, "@graph@index", "@graph@index@set", "@index", "@index@set")
				}
				if hasID {
					containers = append(containers, "@graph@id", "@graph@id@set")
				}
				containers = append(containers, "@graph", "@graph@set", "@set")
				if !hasIndex {
					containers = append(containers, "@graph@index", "@graph@index@set", "@index", "@index@set")
				}
				if !hasID {
					containers = append(containers, "@graph@id", "@graph@id@set")
				}
			} else if isObject && !IsValue(value) {
				containers = append(containers, "@id", "@id@set", "@type", "@set@type")
			}

			// 2.3)

			// defaults for term selection based on type/language
			typeLanguage := "@language"
			typeLanguageValue := "@null"

			// 2.5)
			if reverse {
				typeLanguage = "@type"
				typeLanguageValue = "@reverse"
				containers = append(containers, "@set")
			} else if valueList, containsList := valueMap["@list"]; containsList {
				// 2.6)
				// 2.6.1)
				if _, containsIndex := valueMap["@index"]; !containsIndex {
					containers = append(containers, "@list")
				}
				// 2.6.2)
				list := valueList.([]interface{})
				// 2.6.3)
				if len(list) == 0 {
					//commonLanguage = defaultLanguage
					typeLanguage = "@any"
					typeLanguageValue = "@none"
				} else {
					commonLanguage := ""
					commonType := ""
					if len(list) == 0 {
						commonType = "@id"
					}
					// 2.6.4)
					for _, item := range list {
						// 2.6.4.1)
						itemLanguage := "@none"
						itemType := "@none"
						// 2.6.4.2)
						if IsValue(item) {
							// 2.6.4.2.1)
							itemMap := item.(map[string]interface{})
							if langVal, hasLang := itemMap["@language"]; hasLang {
								itemLanguage = langVal.(string)
							} else if typeVal, hasType := itemMap["@type"]; hasType {
								// 2.6.4.2.2)
								itemType = typeVal.(string)
							} else {
								// 2.6.4.2.3)
								itemLanguage = "@null"
							}
						} else {
							// 2.6.4.3)
							itemType = "@id"
						}
						// 2.6.4.4)
						if commonLanguage == "" {
							commonLanguage = itemLanguage
						} else if commonLanguage != itemLanguage && IsValue(item) {
							// 2.6.4.5)
							commonLanguage = "@none"
						}
						// 2.6.4.6)
						if commonType == "" {
							commonType = itemType
						} else if commonType != itemType {
							// 2.6.4.7)
							commonType = "@none"
						}
						// 2.6.4.8)
						if commonLanguage == "@none" && commonType == "@none" {
							break
						}
					}
					// 2.6.5)
					if commonLanguage == "" {
						commonLanguage = "@none"
					}
					// 2.6.6)
					if commonType == "" {
						commonType = "@none"
					}
					// 2.6.7)
					if commonType != "@none" {
						typeLanguage = "@type"
						typeLanguageValue = commonType
					} else {
						// 2.6.8)
						typeLanguageValue = commonLanguage
					}
				}
			} else {
				// 2.7)
				// 2.7.1)
				if IsValue(value) {

					// 2.7.1.1)
					langVal, hasLang := valueMap["@language"]
					_, hasIndex := valueMap["@index"]
					if hasLang && !hasIndex {
						containers = append(containers, "@language", "@language@set")
						typeLanguageValue = langVal.(string)
					} else if typeVal, hasType := valueMap["@type"]; hasType {
						// 2.7.1.2)
						typeLanguage = "@type"
						typeLanguageValue = typeVal.(string)
					}
				} else {
					// 2.7.2)
					typeLanguage = "@type"
					typeLanguageValue = "@id"
				}
				// 2.7.3)
				containers = append(containers, "@set")
			}
			// 2.8)
			containers = append(containers, "@none")

			// an index map can be used to index values using @none, so add as
			// a low priority
			if isObject {
				if _, hasIndex := valueMap["@index"]; !hasIndex {
					containers = append(containers, "@index", "@index@set")
				}
			}

			// values without type or language can use @language map
			if IsValue(value) && len(value.(map[string]interface{})) == 1 {
				containers = append(containers, "@language", "@language@set")
			}

			// 2.9)
			if typeLanguageValue == "" {
				typeLanguageValue = "@null"
			}
			// 2.10)
			preferredValues := make([]string, 0)
			// 2.11)

			// 2.12)
			if (typeLanguageValue == "@reverse" || typeLanguageValue == "@id") && IsSubjectReference(value) {
				idVal := valueMap["@id"]

				if typeLanguageValue == "@reverse" {
					preferredValues = append(preferredValues, "@reverse")
				}

				// 2.12.1)
				result := c.CompactIri(idVal.(string), nil, true, false)
				resultVal, hasResult := c.termDefinitions[result]
				check := false
				if hasResult {
					resultIDVal, hasResultID := resultVal.(map[string]interface{})["@id"]
					check = hasResultID && idVal == resultIDVal
				}
				if check {
					preferredValues = append(preferredValues, "@vocab")
					preferredValues = append(preferredValues, "@id")
				} else {
					// 2.12.2)
					preferredValues = append(preferredValues, "@id")
					preferredValues = append(preferredValues, "@vocab")
				}
			} else {
				// 2.13)
				preferredValues = append(preferredValues, typeLanguageValue)
			}
			preferredValues = append(preferredValues, "@none")

			// 2.14)
			term := c.SelectTerm(iri, containers, typeLanguage, preferredValues)
			// 2.15)
			if term != "" {
				return term
			}
		}

		// 3)
		if vocabVal, containsVocab := c.values["@vocab"]; containsVocab {
			// determine if vocab is a prefix of the iri
			vocab := vocabVal.(string)
			// 3.1)
			if strings.HasPrefix(iri, vocab) && iri != vocab {
				// use suffix as relative iri if it is not a term in the
				// active context
				suffix := iri[len(vocab):]
				if _, hasSuffix := c.termDefinitions[suffix]; !hasSuffix {
					return suffix
				}
			}
		}
	}

	// 4)
	compactIRI := ""

	// 5)
	for term, termDefinitionVal := range c.termDefinitions {
		if termDefinitionVal == nil {
			continue
		}

		// 5.1)
		if strings.Contains(term, ":") {
			continue
		}

		// 5.2)
		termDefinition := termDefinitionVal.(map[string]interface{})
		idStr := termDefinition["@id"].(string)
		if iri == idStr || !strings.HasPrefix(iri, idStr) {
			continue
		}

		// 5.3)
		candidate := term + ":" + iri[len(idStr):]
		// 5.4)
		candidateVal, containsCandidate := c.termDefinitions[candidate]
		prefix, hasPrefix := termDefinition["_prefix"]
		if (compactIRI == "" || CompareShortestLeast(candidate, compactIRI)) && hasPrefix && prefix.(bool) &&
			(!containsCandidate ||
				(iri == candidateVal.(map[string]interface{})["@id"] && value == nil)) {
			compactIRI = candidate
		}
	}

	// 6)
	if compactIRI != "" {
		return compactIRI
	}

	// 7)
	if !relativeToVocab {
		return RemoveBase(c.values["@base"], iri)
	}

	// 8)
	return iri
}

// GetPrefixes returns a map of potential RDF prefixes based on the JSON-LD Term Definitions
// in this context. No guarantees of the prefixes are given, beyond that it will not contain ":".
//
// onlyCommonPrefixes: If true, the result will not include "not so useful" prefixes, such as
// "term1": "http://example.com/term1", e.g. all IRIs will end with "/" or "#".
// If false, all potential prefixes are returned.
//
// Returns a map from prefix string to IRI string
func (c *Context) GetPrefixes(onlyCommonPrefixes bool) map[string]string {
	prefixes := make(map[string]string)

	for term, termDefinition := range c.termDefinitions {
		if strings.Contains(term, ":") {
			continue
		}
		if termDefinition == nil {
			continue
		}
		termDefinitionMap := termDefinition.(map[string]interface{})
		id := termDefinitionMap["@id"].(string)
		if id == "" {
			continue
		}
		if strings.HasPrefix(term, "@") || strings.HasPrefix(id, "@") {
			continue
		}
		if !onlyCommonPrefixes || strings.HasSuffix(id, "/") || strings.HasSuffix(id, "#") {
			prefixes[term] = id
		}
	}

	return prefixes
}

// GetInverse generates an inverse context for use in the compaction algorithm,
// if not already generated for the given active context.
// See http://www.w3.org/TR/json-ld-api/#inverse-context-creation for further details.
func (c *Context) GetInverse() map[string]interface{} {

	// lazily create inverse
	if c.inverse != nil {
		return c.inverse
	}

	// 1)
	c.inverse = make(map[string]interface{})

	// 2)
	defaultLanguage := "@none"
	langVal, hasLang := c.values["@language"]
	if hasLang {
		defaultLanguage = langVal.(string)
	}

	// create term selections for each mapping in the context, ordered by
	// shortest and then lexicographically least
	terms := GetKeys(c.termDefinitions)
	sort.Sort(ShortestLeast(terms))

	for _, term := range terms {
		definitionVal := c.termDefinitions[term]
		// 3.1)
		if definitionVal == nil {
			continue
		}
		definition := definitionVal.(map[string]interface{})

		// 3.2)
		var containerJoin string // this implementation was adapted from pyLD
		containerVal, present := definition["@container"]
		if !present {
			containerJoin = "@none"
		} else {
			container := containerVal.([]string)
			sort.Strings(container)
			containerJoin = strings.Join(container, "")
		}

		// 3.3)
		iri := definition["@id"].(string)

		// 3.4 + 3.5)
		var containerMap map[string]interface{}
		containerMapVal, present := c.inverse[iri]
		if !present {
			containerMap = make(map[string]interface{})
			c.inverse[iri] = containerMap
		} else {
			containerMap = containerMapVal.(map[string]interface{})
		}

		// 3.6 + 3.7)
		var typeLanguageMap map[string]interface{}
		typeLanguageMapVal, present := containerMap[containerJoin]
		if !present {
			typeLanguageMap = make(map[string]interface{})
			typeLanguageMap["@language"] = make(map[string]interface{})
			typeLanguageMap["@type"] = make(map[string]interface{})
			typeLanguageMap["@any"] = map[string]interface{}{
				"@none": term,
			}
			containerMap[containerJoin] = typeLanguageMap
		} else {
			typeLanguageMap = typeLanguageMapVal.(map[string]interface{})
		}

		// 3.8)
		if reverseVal, hasValue := definition["@reverse"]; hasValue && reverseVal.(bool) {
			typeMap := typeLanguageMap["@type"].(map[string]interface{})
			if _, hasValue := typeMap["@reverse"]; !hasValue {
				typeMap["@reverse"] = term
			}
			// 3.9)
		} else if typeVal, hasValue := definition["@type"]; hasValue {
			typeMap := typeLanguageMap["@type"].(map[string]interface{})
			if _, hasValue := typeMap["@type"]; !hasValue {
				typeMap[typeVal.(string)] = term
			}
			// 3.10)
		} else if langVal, hasValue := definition["@language"]; hasValue {
			languageMap := typeLanguageMap["@language"].(map[string]interface{})
			language := "@null"
			if langVal != nil {
				language = langVal.(string)
			}
			if _, hasLang := languageMap[language]; !hasLang {
				languageMap[language] = term
			}
			// 3.11)
		} else {
			// 3.11.1)
			languageMap := typeLanguageMap["@language"].(map[string]interface{})
			// 3.11.2)
			if _, hasLang := languageMap[defaultLanguage]; !hasLang {
				languageMap[defaultLanguage] = term
			}
			// 3.11.3)
			if _, hasNone := languageMap["@none"]; !hasNone {
				languageMap["@none"] = term
			}
			// 3.11.4)
			typeMap := typeLanguageMap["@type"].(map[string]interface{})
			// 3.11.5)
			if _, hasNone := typeMap["@none"]; !hasNone {
				typeMap["@none"] = term
			}
		}
	}

	// 4)
	return c.inverse
}

// SelectTerm picks the preferred compaction term from the inverse context entry.
// See http://www.w3.org/TR/json-ld-api/#term-selection
//
// This algorithm, invoked via the IRI Compaction algorithm, makes use of an
// active context's inverse context to find the term that is best used to
// compact an IRI. Other information about a value associated with the IRI
// is given, including which container mappings and which type mapping or
// language mapping would be best used to express the value.
func (c *Context) SelectTerm(iri string, containers []string, typeLanguage string, preferredValues []string) string {
	inv := c.GetInverse()
	// 1)
	containerMap := inv[iri].(map[string]interface{})
	// 2)
	for _, container := range containers {
		// 2.1)
		containerVal, hasContainer := containerMap[container]
		if !hasContainer {
			continue
		}
		// 2.2)
		typeLanguageMap := containerVal.(map[string]interface{})
		// 2.3)
		valueMap := typeLanguageMap[typeLanguage].(map[string]interface{})

		// 2.4 )
		for _, item := range preferredValues {
			// 2.4.1
			itemVal, containsItem := valueMap[item]
			if !containsItem {
				continue
			}
			// 2.4.2
			return itemVal.(string)
		}
	}
	// 3)
	return ""
}

// GetContainer retrieves container mapping for the given property.
func (c *Context) GetContainer(property string) []string {
	propertyMap, isMap := c.termDefinitions[property].(map[string]interface{})
	if isMap {
		if container, hasContainer := propertyMap["@container"]; hasContainer {
			return container.([]string)
		}
	}

	return []string{}
}

// GetContainer retrieves container mapping for the given property.
func (c *Context) HasContainerMapping(property string, val string) bool {
	propertyMap, isMap := c.termDefinitions[property].(map[string]interface{})
	if isMap {
		if container, hasContainer := propertyMap["@container"]; hasContainer {
			for _, container := range container.([]string) {
				if container == val {
					return true
				}
			}
		}
	}

	return false
}

// IsReverseProperty returns true if the given property is a reverse property
func (c *Context) IsReverseProperty(property string) bool {
	td := c.GetTermDefinition(property)
	if td == nil {
		return false
	}
	reverse, containsReverse := td["@reverse"]
	return containsReverse && reverse.(bool)
}

// GetTypeMapping returns type mapping for the given property
func (c *Context) GetTypeMapping(property string) string {
	rval := ""
	if defaultLang, hasDefault := c.values["@type"]; hasDefault {
		rval = defaultLang.(string)
	}

	td := c.GetTermDefinition(property)
	if td != nil {
		if val, contains := td["@type"]; contains && val != nil {
			return val.(string)
		}
	}

	return rval
}

// GetLanguageMapping returns language mapping for the given property
func (c *Context) GetLanguageMapping(property string) string {
	rval := ""
	if defaultLang, hasDefault := c.values["@language"]; hasDefault {
		rval = defaultLang.(string)
	}

	td := c.GetTermDefinition(property)
	if td != nil {
		if val, contains := td["@language"]; contains && val != nil {
			return val.(string)
		}
	}

	return rval
}

// GetTermDefinition returns a term definition for the given key
func (c *Context) GetTermDefinition(key string) map[string]interface{} {
	value, _ := c.termDefinitions[key].(map[string]interface{})
	return value
}

// ExpandValue expands the given value by using the coercion and keyword rules in the context.
func (c *Context) ExpandValue(activeProperty string, value interface{}) (interface{}, error) {
	var rval = make(map[string]interface{})
	td := c.GetTermDefinition(activeProperty)
	// 1)
	if td != nil && td["@type"] == "@id" {
		if strVal, isString := value.(string); isString {
			var err error
			rval["@id"], err = c.ExpandIri(strVal, true, false, nil, nil)
			if err != nil {
				return nil, err
			}
		} else {
			rval["@value"] = value
		}
		return rval, nil
	}
	// 2)
	if td != nil && td["@type"] == "@vocab" {
		if strVal, isString := value.(string); isString {
			var err error
			rval["@id"], err = c.ExpandIri(strVal, true, true, nil, nil)
			if err != nil {
				return nil, err
			}
		} else {
			rval["@value"] = value
		}
		return rval, nil
	}
	// 3)
	rval["@value"] = value
	// 4)
	if typeVal, containsType := td["@type"]; td != nil && containsType && typeVal != "@id" && typeVal != "@vocab" {
		rval["@type"] = typeVal
	} else if _, isString := value.(string); isString { // 5)
		// 5.1)
		langVal, containsLang := td["@language"]
		if td != nil && containsLang { // TODO: is "td != nil" necessary?
			if langVal != nil {
				rval["@language"] = langVal.(string)
			}
		} else if langVal := c.values["@language"]; langVal != nil {
			// 5.2)
			rval["@language"] = langVal
		}
	}
	return rval, nil
}

// Serialize transforms the context back into JSON form.
func (c *Context) Serialize() map[string]interface{} {
	ctx := make(map[string]interface{})

	baseVal, hasBase := c.values["@base"]
	if hasBase && baseVal != c.options.Base {
		ctx["@base"] = baseVal
	}
	if langVal, hasLang := c.values["@language"]; hasLang {
		ctx["@language"] = langVal
	}
	if vocabVal, hasVocab := c.values["@vocab"]; hasVocab {
		ctx["@vocab"] = vocabVal
	}
	for term, definitionVal := range c.termDefinitions {
		// Note: definitionVal may be nil for terms which are set to be ignored
		// (see the definition for null value in JSON-LD spec)
		definition, _ := definitionVal.(map[string]interface{})
		langVal, hasLang := definition["@language"]
		containerVal, hasContainer := definition["@container"]
		typeMappingVal, hasType := definition["@type"]
		reverseVal, hasReverse := definition["@reverse"]
		if !hasLang && !hasContainer && !hasType && (!hasReverse || reverseVal == false) {
			var cid interface{}
			id, hasId := definition["@id"]
			if !hasId {
				cid = nil
				ctx[term] = cid
			} else if IsKeyword(id) {
				ctx[term] = id
			} else {
				cid = c.CompactIri(id.(string), nil, false, false)
				if term == cid {
					ctx[term] = id
				} else {
					ctx[term] = cid
				}
				ctx[term] = cid
			}
		} else {
			defn := make(map[string]interface{})
			cid := c.CompactIri(definition["@id"].(string), nil, false, false)
			reverseProperty := reverseVal.(bool)
			if !(term == cid && !reverseProperty) {
				if reverseProperty {
					defn["@reverse"] = cid
				} else {
					defn["@id"] = cid
				}
			}
			if hasType {
				typeMapping := typeMappingVal.(string)
				if IsKeyword(typeMapping) {
					defn["@type"] = typeMapping
				} else {
					defn["@type"] = c.CompactIri(typeMapping, nil, true, false)
				}
			}
			if hasContainer {
				if av, isArray := containerVal.([]string); isArray && len(av) == 1 {
					defn["@container"] = av[0]
				} else {
					defn["@container"] = containerVal
				}
			}
			if hasLang {
				if langVal == false {
					defn["@language"] = nil
				} else {
					defn["@language"] = langVal
				}
			}
			ctx[term] = defn
		}
	}

	rval := make(map[string]interface{})
	if !(ctx == nil || len(ctx) == 0) {
		rval["@context"] = ctx
	}
	return rval
}
