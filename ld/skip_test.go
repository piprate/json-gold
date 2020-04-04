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
		"#tdi07", // No support for i18n-datatype yet
		"#tdi08", // No support for i18n-datatype yet
		"#tdi09", // No support for compound-literal yet
		"#tdi10", // No support for compound-literal yet
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

		"#tdi09", // No support for i18n-datatype yet
		"#tdi10", // No support for i18n-datatype yet
		"#tdi11", // No support for compound-literal yet
		"#tdi12", // No support for compound-literal yet

		"#te111", // Unclear reasons for failure. TODO
		"#te112", // Unclear reasons for failure. TODO
		"#te123", // TODO

		"#tjs", // @json not yet supported

		"#tpr28", // Skipped in Expand test suite
		"#tpr38", // TODO
		"#tpr39", // TODO
		"#ttn02", // Unclear what the correct behaviour should be: two values get collapsed into one
	},
	"testdata/html-manifest.jsonld": {
		"#t", // HTML inputs not supported yet
	},
	"testdata/normalization/manifest-urgna2012.jsonld": {
		"manifest-urgna2012#test060", // TODO
	},
	"testdata/normalization/manifest-urdna2015.jsonld": {
		"manifest-urdna2015#test060", // TODO
	},
}
