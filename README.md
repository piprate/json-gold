# JSON-goLD

This library is an implementation of the [JSON-LD 1.0](http://json-ld.org/) specification in Go.
It supports both URDNA2015 and URGNA2012 RDF dataset normalisation algorithms.

## MAINTAINER CHANGE NOTIFICATION ##

We are planning to add new features to this library, including support for Linked Data Signatures, content-addressable JSON-LD, etc. In order to facilitate this development, the maintainer of JSON-goLD (@kazarena) is going to copy this repository to the GitHub organisation of his company (@piprate). See the details and further discussion [here](https://github.com/kazarena/json-gold/issues/20). Feedback welcome.

## Testing & Compliance ##

As of December 1, 2017:

* all JSON-LD 1.0 tests from the [official JSON-LD test suite](https://github.com/json-ld/json-ld.org/tree/master/test-suite) pass
* all RDF Dataset Normalisation tests from the [current test suite](https://json-ld.github.io/normalization/tests/index.html) pass
* JSON-LD 1.1 spec is not currenty supported

## Inspiration ##

This implementation was heavily influenced by [JSONLD-Java](https://github.com/jsonld-java/jsonld-java) with some techniques borrowed from [PyLD](https://github.com/digitalbazaar/pyld) and [gojsonld](https://github.com/linkeddata/gojsonld). Big thank you to the contributors of the forementioned libraries for figuring out implementation details of the core algorithms.

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
options.Format = "application/nquads"

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
options.Format = "application/nquads"
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
