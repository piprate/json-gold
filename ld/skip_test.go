package ld_test

// This map contains prefixes for test IDs that should be skipped
// when running the official test suites for JSON-LD, Framing and Normalisation.
//
// Structure: <relative path to manifest file> ==> list of test ID prefixes to skip
var skippedTests = map[string][]string{
	"testdata/compact-manifest.jsonld": {
		"#t0111", // TODO
		"#t0112", // TODO
		"#t0113", // TODO
		"#tin",   // TODO
		"#tm023", // TODO
		"#tp001", // TODO
	},
	"testdata/expand-manifest.jsonld": {
		"#t0122", // TODO
		"#t0123", // TODO
		"#tc032", // TODO
		"#tc033", // TODO
		"#tc037", // TODO: New test
		"#tc038", // TODO: New test
		"#tec02", // TODO
		"#ter52", // TODO
		"#ter54", // TODO: New test
		"#ter56", // TODO: New test
		"#tpr28", // TODO: New test
		"#tpr38", // TODO: New test
		"#tpr39", // TODO: New test
		"#tpr43", // TODO: New test
	},
	"testdata/flatten-manifest.jsonld": {},
	"testdata/fromRdf-manifest.jsonld": {
		"#t0027", // TODO: New test
		"#tdi05", // No support for i18n-datatype yet
		"#tdi06", // No support for i18n-datatype yet
		"#tdi11", // No support for compound-literal yet
		"#tdi12", // No support for compound-literal yet
		"#tjs",   // @json not yet supported
	},
	"testdata/remote-doc-manifest.jsonld": {
		"#t0013", // HTML documents aren't supported yet
	},
	"testdata/toRdf-manifest.jsonld": {
		"#tc032", // TODO
		"#tc033", // TODO
		"#tc037", // TODO: New test
		"#tc038", // TODO: New test
		"#tdi09", // No support for i18n-datatype yet
		"#tdi10", // No support for i18n-datatype yet
		"#tdi11", // No support for compound-literal yet
		"#tdi12", // No support for compound-literal yet
		"#te075", // No support for GeneralizedRdf
		"#te085", // test passes, bug in isomorphism check
		"#te086", // test passes, bug in isomorphism check
		"#te087", // test passes, bug in isomorphism check
		"#te111", // TODO
		"#te112", // TODO
		"#te123", // TODO: new test
		"#tec02", // TODO: new test
		"#ter52", // TODO: new test
		"#ter54", // TODO: new test
		"#ter56", // TODO: new test
		"#tjs03", // TODO numeric format
		"#tjs07",
		"#tjs08",
		"#tjs14",
		"#tjs15",
		"#tjs16",
		"#tjs17",
		"#tjs18",
		"#tjs21",
		"#tjs22",
		"#tjs23",
		"#tli11", // TODO: New test
		"#tli12", // TODO: New test
		"#tli14", // TODO: New test
		"#tpr28", // Skipped in Expand test suite
		"#tpr38", // TODO
		"#tpr39", // TODO
		"#tpr43", // TODO: New test
		"#ttn02", // TODO
	},
	"testdata/html-manifest.jsonld": {
		"#t", // HTML inputs not supported yet
	},
	"testdata/frame-manifest.jsonld": {
		// TODO: all tests below are skipped until we add support for JSON-LD Framing 1.1
		"#t0011",
		"#t0023",
		"#t0026",
		"#t0027",
		"#t0028",
		"#t0029",
		"#t0030",
		"#t0031",
		"#t0032",
		"#t0034",
		"#t0035",
		"#t0036",
		"#t0037",
		"#t0038",
		"#t0039",
		"#t0040",
		"#t0041",
		"#t0042",
		"#t0043",
		"#t0044",
		"#t0045",
		"#t0047",
		"#t0048",
		"#t0050",
		"#t0051",
		"#t0055",
		"#t0058",
		"#t006",
		"#teo01",
		"#tg002",
		"#tg003",
		"#tg004",
		"#tg006",
		"#tg007",
		"#tg008",
		"#tg009",
		"#tg010",
		"#tin",
		"#tp046",
		"#tp049",
		"#tp050",
		"#tra",
	},
	"testdata/normalization/manifest-urgna2012.jsonld": {
		"manifest-urgna2012#test060",
	},
	"testdata/normalization/manifest-urdna2015.jsonld": {
		"manifest-urdna2015#test060",
	},
}
