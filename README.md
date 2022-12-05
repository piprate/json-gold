# JSON-goLD

[![GoDoc](https://godoc.org/github.com/piprate/json-gold?status.svg)](https://pkg.go.dev/github.com/piprate/json-gold)
[![ci](https://github.com/piprate/json-gold/actions/workflows/ci.yml/badge.svg)](https://github.com/piprate/json-gold/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/piprate/json-gold/branch/master/graph/badge.svg?token=JvEEDMmppm)](https://codecov.io/gh/piprate/json-gold)

This library is an implementation of the [JSON-LD 1.1](http://json-ld.org/) specification in Go.
It supports both URDNA2015 and URGNA2012 RDF dataset normalisation algorithms.

## Conformance ##

This library aims to pass the official [test suite](https://json-ld.org/test-suite/) and conform with the following:

- [JSON-LD 1.0](http://www.w3.org/TR/2014/REC-json-ld-20140116/),
  W3C Recommendation,
  2014-01-16, and any [errata](http://www.w3.org/2014/json-ld-errata)
- [JSON-LD 1.0 Processing Algorithms and API](http://www.w3.org/TR/2014/REC-json-ld-api-20140116/),
  W3C Recommendation,
  2014-01-16, and any [errata](http://www.w3.org/2014/json-ld-errata)
- [JSON-LD 1.1](https://www.w3.org/TR/2019/CR-json-ld11-20191212/),
  W3C Candidate Recommendation,
  2019-12-12 or [newer JSON-LD latest](https://json-ld.org/spec/latest/json-ld/)
- [JSON-LD 1.1 Processing Algorithms and API](https://www.w3.org/TR/2019/CR-json-ld11-api-20191212/),
  W3C Candidate Recommendation,
  2019-12-12 or [newer JSON-LD Processing Algorithms and API latest](https://www.w3.org/TR/json-ld11-api/)
- [JSON-LD Framing 1.1](https://www.w3.org/TR/2019/CR-json-ld11-framing-20191212/)
  W3C Candidate Recommendation
  2019-12-12 or [newer](https://www.w3.org/TR/json-ld11-framing/)

### Current JSON-LD 1.1 Conformance Status

This library provides comprehensive support of JSON-LD 1.1 specification, except in the areas mentioned below:

#### Expansion

Good coverage.

#### Compaction

Good coverage, except:

- `@included` directive not supported

#### RDF Serialization/Deserialization

Good coverage, except:

- partial support for JSON literals (`@json`)
- `rdfDirection` option is not yet supported (including _i18n-datatype_ and _compound-literal_ forms)

#### HTML based processing

Not supported.

### Current JSON-LD 1.1 Framing Conformance Status

Not supported. The current implementation is still based on an earlier version of JSON-LD 1.1 Framing specification.

### Official 1.1 Test Suite

As of April 4th, 2020:

* 92.3% of tests from the [official JSON-LD test suite](https://github.com/w3c/json-ld-api/tree/master/tests) pass.
* all RDF Dataset Normalisation tests from the [current test suite](https://json-ld.github.io/normalization/tests/index.html) pass

## Examples ##

### Expand ###

See complete code in [examples/expand.go](examples/expand.go).

```go
proc := ld.NewJsonLdProcessor()
options := ld.NewJsonLdOptions("")

// expanding remote document

expanded, err := proc.Expand("http://json-ld.org/test-suite/tests/expand-0002-in.jsonld", options)
if err != nil {
	log.Println("Error when expanding JSON-LD document:", err)
	return
}

// expanding in-memory document

doc := map[string]interface{}{
	"@context": "http://schema.org/",
	"@type": "Person",
	"name": "Jane Doe",
	"jobTitle": "Professor",
	"telephone": "(425) 123-4567",
	"url": "http://www.janedoe.com",
}

expanded, err = proc.Expand(doc, options)
```

### Compact ###

See complete code in [examples/compact.go](examples/compact.go).

```go
proc := ld.NewJsonLdProcessor()
options := ld.NewJsonLdOptions("")

doc := map[string]interface{}{
	"@id": "http://example.org/test#book",
	"http://example.org/vocab#contains": map[string]interface{}{
		"@id": "http://example.org/test#chapter",
	},
	"http://purl.org/dc/elements/1.1/title": "Title",
}

context := map[string]interface{}{
	"@context": map[string]interface{}{
		"dc": "http://purl.org/dc/elements/1.1/",
		"ex": "http://example.org/vocab#",
		"ex:contains": map[string]interface{}{
			"@type": "@id",
		},
	},
}

compactedDoc, err := proc.Compact(doc, context, options)
```

### Flatten ###

See complete code in [examples/flatten.go](examples/flatten.go).

```go
proc := ld.NewJsonLdProcessor()
options := ld.NewJsonLdOptions("")

doc := map[string]interface{}{
	"@context": []interface{}{
		map[string]interface{}{
			"name": "http://xmlns.com/foaf/0.1/name",
			"homepage": map[string]interface{}{
				"@id": "http://xmlns.com/foaf/0.1/homepage",
				"@type": "@id",
			},
		},
		map[string]interface{}{
			"ical": "http://www.w3.org/2002/12/cal/ical#",
		},
	},
	"@id": "http://example.com/speakers#Alice",
	"name": "Alice",
	"homepage": "http://xkcd.com/177/",
	"ical:summary": "Alice Talk",
	"ical:location": "Lyon Convention Centre, Lyon, France",
}

flattenedDoc, err := proc.Flatten(doc, nil, options)
```

### Frame ###

See complete code in [examples/frame.go](examples/frame.go).

```go
proc := ld.NewJsonLdProcessor()
options := ld.NewJsonLdOptions("")

doc := map[string]interface{}{
	"@context": map[string]interface{}{
		"dc": "http://purl.org/dc/elements/1.1/",
		"ex": "http://example.org/vocab#",
		"ex:contains": map[string]interface{}{"@type": "@id"},
	},
	"@graph": []interface{}{
		map[string]interface{}{
			"@id": "http://example.org/test/#library",
			"@type": "ex:Library",
			"ex:contains": "http://example.org/test#book",
		},
		map[string]interface{}{
			"@id": "http://example.org/test#book",
			"@type": "ex:Book",
			"dc:contributor": "Writer",
			"dc:title": "My Book",
			"ex:contains": "http://example.org/test#chapter",
		},
		map[string]interface{}{
			"@id": "http://example.org/test#chapter",
			"@type": "ex:Chapter",
			"dc:description": "Fun",
			"dc:title": "Chapter One",
		},
	},
}

frame := map[string]interface{}{
	"@context": map[string]interface{}{
		"dc": "http://purl.org/dc/elements/1.1/",
		"ex": "http://example.org/vocab#",
	},
	"@type": "ex:Library",
	"ex:contains": map[string]interface{}{
		"@type": "ex:Book",
		"ex:contains": map[string]interface{}{
			"@type": "ex:Chapter",
		},
	},
}

framedDoc, err := proc.Frame(doc, frame, options)
```

### To RDF ###

See complete code in [examples/to_rdf.go](examples/to_rdf.go).

```go
proc := ld.NewJsonLdProcessor()
options := ld.NewJsonLdOptions("")
options.Format = "application/n-quads"

// this JSON-LD document was taken from http://json-ld.org/test-suite/tests/toRdf-0028-in.jsonld
doc := map[string]interface{}{
	"@context": map[string]interface{}{
		"sec":        "http://purl.org/security#",
		"xsd":        "http://www.w3.org/2001/XMLSchema#",
		"rdf":        "http://www.w3.org/1999/02/22-rdf-syntax-ns#",
		"dc":         "http://purl.org/dc/terms/",
		"sec:signer": map[string]interface{}{"@type": "@id"},
		"dc:created": map[string]interface{}{"@type": "xsd:dateTime"},
	},
	"@id":                "http://example.org/sig1",
	"@type":              []interface{}{"rdf:Graph", "sec:SignedGraph"},
	"dc:created":         "2011-09-23T20:21:34Z",
	"sec:signer":         "http://payswarm.example.com/i/john/keys/5",
	"sec:signatureValue": "OGQzNGVkMzVm4NTIyZTkZDYMmMzQzNmExMgoYzI43Q3ODIyOWM32NjI=",
	"@graph": map[string]interface{}{
		"@id":      "http://example.org/fact1",
		"dc:title": "Hello World!",
	},
}
triples, err := proc.ToRDF(doc, options)
```

### From RDF ###

See complete code in [examples/from_rdf.go](examples/from_rdf.go).

```go
proc := ld.NewJsonLdProcessor()
options := ld.NewJsonLdOptions("")

triples := `
	<http://example.com/Subj1> <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <http://example.com/Type> .
	<http://example.com/Subj1> <http://example.com/prop1> <http://example.com/Obj1> .
	<http://example.com/Subj1> <http://example.com/prop2> "Plain" .
	<http://example.com/Subj1> <http://example.com/prop2> "2012-05-12"^^<http://www.w3.org/2001/XMLSchema#date> .
	<http://example.com/Subj1> <http://example.com/prop2> "English"@en .
`

doc, err := proc.FromRDF(triples, options)
```

### Normalize ###

See complete code in [examples/normalize.go](examples/normalize.go).

```go
proc := ld.NewJsonLdProcessor()
options := ld.NewJsonLdOptions("")
options.Format = "application/n-quads"
options.Algorithm = "URDNA2015"

doc := map[string]interface{}{
	"@context": map[string]interface{}{
		"ex": "http://example.org/vocab#",
	},
	"@id": "http://example.org/test#example",
	"@type": "ex:Foo",
	"ex:embed": map[string]interface{}{
		"@type": "ex:Bar",
	},
}

normalizedTriples, err := proc.Normalize(doc, options)
```

## Inspiration ##

This implementation was influenced by [Ruby JSON-LD reader/writer](https://github.com/ruby-rdf/json-ld), [JSONLD-Java](https://github.com/jsonld-java/jsonld-java) with some techniques borrowed from [PyLD](https://github.com/digitalbazaar/pyld) and [gojsonld](https://github.com/linkeddata/gojsonld). Big thank you to the contributors of the aforementioned libraries for figuring out implementation details of the core algorithms.

## History ##

The [original library](https://github.com/kazarena/json-gold) was written by Stan Nazarenko
([@kazarena](https://github.com/kazarena)). See the full list of contributors
[here](https://github.com/piprate/json-gold/blob/master/CONTRIBUTORS.md).
