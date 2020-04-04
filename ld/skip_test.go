package ld_test

// This map contains prefixes for test IDs that should be skipped
// when running the official test suites for JSON-LD, Framing and Normalisation.
//
// Structure: <relative path to manifest file> ==> list of test ID prefixes to skip
//
var skippedTests = map[string][]string{
	"testdata/compact-manifest.jsonld": {
		"#tin",   // TODO
		"#tp001", // TODO
	},
	"testdata/expand-manifest.jsonld": {
		"#tpr28", // TODO
		"#tpr38", // TODO
		"#tpr39", // TODO
		"#t0122", // TODO
		"#t0123", // TODO
		"#tc032", // TODO
		"#tc033", // TODO
		"#tec02", // TODO
		"#ter52", // TODO
	},
	"testdata/flatten-manifest.jsonld": {},
	"testdata/fromRdf-manifest.jsonld": {
		"#tdi05", // No support for i18n-datatype yet
		"#tdi06", // No support for i18n-datatype yet
		"#tdi11", // No support for compound-literal yet
		"#tdi12", // No support for compound-literal yet
		"#tjs",   // @json not yet supported
	},
	"testdata/remote-doc-manifest.jsonld": {
		"#t0013", // HTML documents aren't supported yet
		"#tla01", // HTML documents aren't supported yet
		"#tla05", // HTML documents aren't supported yet
	},
	"testdata/toRdf-manifest.jsonld": {
		"#tc032", // TODO
		"#tc033", // TODO
		"#tec02", // TODO
		"#ter52", // TODO

		"#te123", // TODO

		"#tpr28", // Skipped in Expand test suite
		"#tpr38", // TODO
		"#tpr39", // TODO
	},
	"testdata/html-manifest.jsonld": {
		"#t", // HTML inputs not supported yet
	},
	"testdata/frame-manifest.jsonld": {},
	"testdata/normalization/manifest-urgna2012.jsonld": {},
	"testdata/normalization/manifest-urdna2015.jsonld": {},
}
