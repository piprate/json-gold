package ld_test

// This map contains prefixes for test IDs that should be skipped
// when running the official test suites for JSON-LD, Framing and Normalisation.
//
// Structure: <relative path to manifest file> ==> list of test ID prefixes to skip
//
var skippedTests = map[string][]string{
	"testdata/html-manifest.jsonld": {
		"#t",
	},
}
