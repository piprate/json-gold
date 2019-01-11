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

package ld_test

import (
	"encoding/json"
	. "github.com/piprate/json-gold/ld"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// RewriteHostTransport is an http.RoundTripper that rewrites requests
// using the provided Host. The Opaque field is untouched.
// If Transport is nil, http.DefaultTransport is used
type RewriteHostTransport struct {
	Transport http.RoundTripper
	Host      string
}

func (t RewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// save the original host
	origHost := req.URL.Host
	// rewrite the host
	req.URL.Host = t.Host

	rt := t.Transport
	if rt == nil {
		rt = http.DefaultTransport
	}
	res, err := rt.RoundTrip(req)

	if err == nil {
		// restore the original host to ensure the client doesn't know the response
		// came from a MockServer instance
		res.Request.URL.Host = origHost
	}
	return res, err
}

// MockServer uses httptest package to mock live HTTP calls.
type MockServer struct {
	Base       string
	TestFolder string

	ContentType string
	HttpLink    []string
	HttpStatus  int
	RedirectTo  string

	server *httptest.Server

	DocumentLoader DocumentLoader
}

// NewMockServer creates a new instance of MockServer.
func NewMockServer(base string, testFolder string) *MockServer {
	mockServer := &MockServer{
		Base:       base,
		TestFolder: testFolder,
	}

	var ts *httptest.Server
	mockFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if mockServer.HttpStatus != 0 {
			// must be a redirect
			w.Header().Set("Location", mockServer.Base+mockServer.RedirectTo)
			w.WriteHeader(mockServer.HttpStatus)
		} else {
			u := r.URL.String()

			if strings.HasPrefix(u, mockServer.Base) {
				contentType := mockServer.ContentType
				if contentType == "" {
					if strings.HasSuffix(u, ".jsonld") {
						contentType = "application/ld+json"
					} else {
						contentType = "application/json"
					}
				}

				fileName := filepath.Join(mockServer.TestFolder, u[len(mockServer.Base):])
				inputBytes, err := ioutil.ReadFile(fileName)
				if err == nil {
					w.Header().Set("Content-Type", contentType)
					if mockServer.HttpLink != nil {
						w.Header().Set("Link", strings.Join(mockServer.HttpLink, ", "))
					}
					w.WriteHeader(http.StatusOK)
					w.Write(inputBytes)
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}

		}

		// reset the context for the second call so that it succeeds.
		// currently there are no tests where it needs to work in a different way
		mockServer.HttpStatus = 0
		mockServer.HttpLink = nil
	})

	if strings.HasPrefix(base, "https") {
		ts = httptest.NewTLSServer(mockFunc)
	} else {
		ts = httptest.NewServer(mockFunc)
	}

	// get httptest.Server's URL

	tsUrl, err := url.Parse(ts.URL)
	if err != nil {
		log.Fatalln("failed to parse httptest.Server URL:", err)
	}

	// update base URL with httptest.Server's host

	baseUrl, err := url.Parse(base)
	if err != nil {
		log.Fatalln("failed to parse base URL:", err)
	}
	baseUrl.Host = tsUrl.Host
	mockServer.Base = baseUrl.Path

	client := ts.Client()

	client.Transport = RewriteHostTransport{
		Transport: client.Transport,
		Host:      tsUrl.Host,
	}

	mockServer.server = ts
	mockServer.DocumentLoader = NewDefaultDocumentLoader(client)

	return mockServer
}

func (ms *MockServer) SetExpectedBehaviour(contentType string, httpLink []string, httpStatus int, redirectTo string) {
	ms.ContentType = contentType
	ms.HttpLink = httpLink
	ms.HttpStatus = httpStatus
	ms.RedirectTo = redirectTo
}

func (ms *MockServer) Close() {
	if ms.server != nil {
		ms.server.Close()
	}
}

type TestDefinition struct {
	Id               string
	Name             string
	Type             string
	EvaluationType   string
	InputURL         string
	InputFileName    string
	ExpectedFileName string
	Option           map[string]interface{}
	Raw              map[string]interface{}
	Skip             bool
}

func TestSuite(t *testing.T) {
	testDir := "testdata"
	fileInfoList, err := ioutil.ReadDir(testDir)
	assert.NoError(t, err)

	// read all manifests
	manifestMap := make(map[string]map[string]interface{})
	for _, fileInfo := range fileInfoList {
		if strings.HasSuffix(fileInfo.Name(), "-manifest.jsonld") {
			inputBytes, err := ioutil.ReadFile(filepath.Join(testDir, fileInfo.Name()))
			assert.NoError(t, err)

			var manifest map[string]interface{}
			err = json.Unmarshal(inputBytes, &manifest)
			assert.NoError(t, err)

			manifestMap[fileInfo.Name()] = manifest
		}
	}

	dl := NewDefaultDocumentLoader(nil)
	proc := NewJsonLdProcessor()
	earlReport := NewEarlReport()

	for manifestName, manifest := range manifestMap {
		baseIri := ""
		testListKey := "entries"
		if baseValue, hasBase := manifest["baseIri"]; hasBase {
			baseIri = baseValue.(string)
			// it must be a JSON-LD test manifest
			testListKey = "sequence"
		}
		manifestURI := baseIri + manifestName

		// start a mock HTTP server
		mockServer := NewMockServer(baseIri, testDir)
		defer mockServer.Close()

		testList := make([]*TestDefinition, 0)
		for _, testData := range manifest[testListKey].([]interface{}) {
			testMap := testData.(map[string]interface{})
			testId := ""
			testType := ""
			testEvaluationType := "jld:PositiveEvaluationTest"
			inputURL := ""
			inputFileName := ""
			expectedFileName := ""
			if baseIri != "" {
				// JSON-LD test manifest
				testId = testMap["@id"].(string)
				testType = testMap["@type"].([]interface{})[1].(string)
				testEvaluationType = testMap["@type"].([]interface{})[0].(string)
				inputURL = baseIri + testMap["input"].(string)
				inputFileName = testMap["input"].(string)
				expectedFileName = testMap["expect"].(string)
			} else {
				// Normalisation test manifest
				testId = testMap["id"].(string)
				testType = testMap["type"].(string)
				inputFileName = testMap["action"].(string)
				expectedFileName = testMap["result"].(string)
			}

			skip := false
			if skipVal, hasSkip := testMap["skip"]; hasSkip {
				skip = skipVal.(bool)
			}

			testName := testId
			if strings.HasPrefix(testName, "#") {
				testName = manifestURI + testName
			}

			td := &TestDefinition{
				Id:               testId,
				Name:             testName,
				Type:             testType,
				EvaluationType:   testEvaluationType,
				InputURL:         inputURL,
				InputFileName:    filepath.Join(testDir, inputFileName),
				ExpectedFileName: filepath.Join(testDir, expectedFileName),
				Raw:              testMap,
				Skip:             skip,
			}
			if optionVal, optionsPresent := testMap["option"]; optionsPresent {
				td.Option = optionVal.(map[string]interface{})
			}
			testList = append(testList, td)
		}

	SequenceLoop:
		for _, td := range testList {
			// ToRDF tests with a reference to RFC3986 don't agree with Go implementation of RFC 3986
			// (see url.URL.ResolveReference(). Skipping for now, as other JSON-LD implementations do.
			purpose := td.Raw["purpose"]
			if purpose != nil && strings.Contains(purpose.(string), "RFC3986") {
				log.Println("Skipping RFC3986 test", td.Id, ":", td.Name)
				continue
			}

			if td.Skip {
				log.Println("Test marked as skipped:", td.Id, ":", td.Name)
				continue
			}

			// read 'option' section and initialise JsonLdOptions and expected HTTP server responses

			options := NewJsonLdOptions("")

			var returnContentType string
			var returnHttpStatus int
			var returnRedirectTo string
			var returnHttpLink []string

			if td.Option != nil {
				testOpts := td.Option

				if value, hasValue := testOpts["specVersion"]; hasValue {
					if value == JsonLd_1_0 {
						log.Println("Skipping JSON-LD 1.0 test:", td.Id, ":", td.Name)
						continue
					}
				}

				if value, hasValue := testOpts["processingMode"]; hasValue {
					options.ProcessingMode = value.(string)
				}

				if value, hasValue := testOpts["base"]; hasValue {
					options.Base = value.(string)
				}
				if value, hasValue := testOpts["expandContext"]; hasValue {
					contextDoc, err := dl.LoadDocument(filepath.Join(testDir, value.(string)))
					assert.NoError(t, err)
					options.ExpandContext = contextDoc.Document
				}
				if value, hasValue := testOpts["compactArrays"]; hasValue {
					options.CompactArrays = value.(bool)
				}
				if value, hasValue := testOpts["omitGraph"]; hasValue {
					options.OmitGraph = value.(bool)
				}
				if value, hasValue := testOpts["useNativeTypes"]; hasValue {
					options.UseNativeTypes = value.(bool)
				}
				if value, hasValue := testOpts["useRdfType"]; hasValue {
					options.UseRdfType = value.(bool)
				}
				if value, hasValue := testOpts["produceGeneralizedRdf"]; hasValue {
					options.ProduceGeneralizedRdf = value.(bool)
				}

				if value, hasValue := testOpts["contentType"]; hasValue {
					returnContentType = value.(string)
				}
				if value, hasValue := testOpts["httpStatus"]; hasValue {
					returnHttpStatus = int(value.(float64))
				}
				if value, hasValue := testOpts["redirectTo"]; hasValue {
					returnRedirectTo = value.(string)
				}
				if value, hasValue := testOpts["httpLink"]; hasValue {
					returnHttpLink = make([]string, 0)
					if valueList, isList := value.([]interface{}); isList {
						for _, link := range valueList {
							returnHttpLink = append(returnHttpLink, link.(string))
						}
					} else {
						returnHttpLink = append(returnHttpLink, value.(string))
					}
				}
			}

			mockServer.SetExpectedBehaviour(returnContentType, returnHttpLink, returnHttpStatus, returnRedirectTo)

			options.DocumentLoader = mockServer.DocumentLoader

			var result interface{}
			var opError error

			switch td.Type {
			case "jld:ExpandTest":
				log.Println("Running Expand test", td.Id, ":", td.Name)
				result, opError = proc.Expand(td.InputURL, options)
			case "jld:CompactTest":
				log.Println("Running Compact test", td.Id, ":", td.Name)

				contextFilename := td.Raw["context"].(string)
				contextDoc, err := dl.LoadDocument(filepath.Join(testDir, contextFilename))
				assert.NoError(t, err)

				result, opError = proc.Compact(td.InputURL, contextDoc.Document, options)
			case "jld:FlattenTest":
				log.Println("Running Flatten test", td.Id, ":", td.Name)

				var ctxDoc interface{}
				if ctxVal, hasContext := td.Raw["context"]; hasContext {
					contextFilename := ctxVal.(string)
					contextDoc, err := dl.LoadDocument(filepath.Join(testDir, contextFilename))
					assert.NoError(t, err)
					ctxDoc = contextDoc.Document
				}

				result, opError = proc.Flatten(td.InputURL, ctxDoc, options)
			case "jld:FrameTest":
				log.Println("Running Frame test", td.Id, ":", td.Name)

				frameFilename := td.Raw["frame"].(string)
				frameDoc, err := dl.LoadDocument(filepath.Join(testDir, frameFilename))
				assert.NoError(t, err)

				result, opError = proc.Frame(td.InputURL, frameDoc.Document, options)
			case "jld:FromRDFTest":
				log.Println("Running FromRDF test", td.Id, ":", td.Name)

				inputBytes, err := ioutil.ReadFile(td.InputFileName)
				assert.NoError(t, err)
				input := string(inputBytes)

				result, err = proc.FromRDF(input, options)
			case "jld:ToRDFTest":
				log.Println("Running ToRDF test", td.Id, ":", td.Name)

				options.Format = "application/n-quads"
				result, opError = proc.ToRDF(td.InputURL, options)
			case "rdfn:Urgna2012EvalTest":
				log.Println("Running URGNA2012 test", td.Id, ":", td.Name)

				inputBytes, err := ioutil.ReadFile(td.InputFileName)
				assert.NoError(t, err)
				input := string(inputBytes)
				options.InputFormat = "application/n-quads"
				options.Format = "application/n-quads"
				options.Algorithm = "URGNA2012"
				result, opError = proc.Normalize(input, options)
			case "rdfn:Urdna2015EvalTest":
				log.Println("Running URDNA2015 test", td.Name)

				inputBytes, err := ioutil.ReadFile(td.InputFileName)
				assert.NoError(t, err)
				input := string(inputBytes)
				options.InputFormat = "application/n-quads"
				options.Format = "application/n-quads"
				options.Algorithm = "URDNA2015"
				result, opError = proc.Normalize(input, options)
			default:
				break SequenceLoop
			}

			var expected interface{}
			var expectedType string
			if td.EvaluationType == "jld:PositiveEvaluationTest" {
				// we don't expect any errors here
				if !assert.NoError(t, opError) {
					earlReport.addAssertion(td.Name, false)
				}

				// load expected document
				expectedType = filepath.Ext(td.ExpectedFileName)
				if expectedType == ".jsonld" || expectedType == ".json" {
					// load as JSON-LD/JSON
					rdOut, err := dl.LoadDocument(td.ExpectedFileName)
					assert.NoError(t, err)
					expected = rdOut.Document
				} else if expectedType == ".nq" {
					// load as N-Quads
					expectedBytes, err := ioutil.ReadFile(td.ExpectedFileName)
					assert.NoError(t, err)
					expected = string(expectedBytes)
				}

				// marshal/unmarshal the result to avoid any differences due to formatting & key sequences
				resultBytes, _ := json.MarshalIndent(result, "", "  ")
				err = json.Unmarshal(resultBytes, &result)
			} else if td.EvaluationType == "jld:NegativeEvaluationTest" {
				expected = td.Raw["expect"].(string)

				if opError != nil {
					result = string(opError.(*JsonLdError).Code)
				} else {
					result = ""
				}
			}

			if !assert.True(t, DeepCompare(expected, result, true)) {
				// print out expected vs. actual results in a human readable form
				if expectedType == ".jsonld" || expectedType == ".json" {
					log.Println("==== ACTUAL ====")
					b, _ := json.MarshalIndent(result, "", "  ")
					_, _ = os.Stdout.Write(b)
					_, _ = os.Stdout.WriteString("\n")
					log.Println("==== EXPECTED ====")
					b, _ = json.MarshalIndent(expected, "", "  ")
					_, _ = os.Stdout.Write(b)
					_, _ = os.Stdout.WriteString("\n")

				} else if expectedType == ".nq" {
					log.Println("==== ACTUAL ====")
					_, _ = os.Stdout.WriteString(result.(string))
					log.Println("==== EXPECTED ====")
					_, _ = os.Stdout.WriteString(expected.(string))
				} else {
					log.Println("==== ACTUAL ====")
					_, _ = os.Stdout.WriteString(result.(string))
					_, _ = os.Stdout.WriteString("\n")
					log.Println("==== EXPECTED ====")
					_, _ = os.Stdout.WriteString(expected.(string))
					_, _ = os.Stdout.WriteString("\n")
				}
				log.Println("Error when running", td.Id, "for", td.Type)
				earlReport.addAssertion(td.Name, false)
				return
			} else {
				earlReport.addAssertion(td.Name, true)
			}
		}
	}
	earlReport.write("earl.jsonld")
}

const (
	assertor     = "https://github.com/kazarena"
	assertorName = "Stan Nazarenko"
)

// EarlReport generates an EARL report.
type EarlReport struct {
	report map[string]interface{}
}

func NewEarlReport() *EarlReport {
	rval := &EarlReport{
		report: map[string]interface{}{
			"@context": map[string]interface{}{
				"doap":            "http://usefulinc.com/ns/doap#",
				"foaf":            "http://xmlns.com/foaf/0.1/",
				"dc":              "http://purl.org/dc/terms/",
				"earl":            "http://www.w3.org/ns/earl#",
				"xsd":             "http://www.w3.org/2001/XMLSchema#",
				"doap:homepage":   map[string]interface{}{"@type": "@id"},
				"doap:license":    map[string]interface{}{"@type": "@id"},
				"dc:creator":      map[string]interface{}{"@type": "@id"},
				"foaf:homepage":   map[string]interface{}{"@type": "@id"},
				"subjectOf":       map[string]interface{}{"@reverse": "earl:subject"},
				"earl:assertedBy": map[string]interface{}{"@type": "@id"},
				"earl:mode":       map[string]interface{}{"@type": "@id"},
				"earl:test":       map[string]interface{}{"@type": "@id"},
				"earl:outcome":    map[string]interface{}{"@type": "@id"},
				"dc:date":         map[string]interface{}{"@type": "xsd:date"},
			},
			"@id": "https://github.com/piprate/json-gold",
			"@type": []interface{}{
				"doap:Project",
				"earl:TestSubject",
				"earl:Software",
			},
			"doap:name":                 "JSON-goLD",
			"dc:title":                  "JSON-goLD",
			"doap:homepage":             "https://github.com/piprate/json-gold",
			"doap:license":              "https://github.com/piprate/json-gold/blob/master/LICENSE",
			"doap:description":          "A JSON-LD processor for Go",
			"doap:programming-language": "Go",
			"dc:creator":                assertor,
			"doap:developer": map[string]interface{}{
				"@id": assertor,
				"@type": []interface{}{
					"foaf:Person",
					"earl:Assertor",
				},
				"foaf:name":     assertorName,
				"foaf:homepage": assertor,
			},
			"dc:date": map[string]interface{}{
				"@value": time.Now().Format("2006-01-02"),
				"@type":  "xsd:date",
			},
			"subjectOf": make([]interface{}, 0),
		},
	}

	return rval
}

func (er *EarlReport) addAssertion(testName string, success bool) {
	var outcome string
	if success {
		outcome = "earl:passed"
	} else {
		outcome = "earl:failed"
	}
	er.report["subjectOf"] = append(
		er.report["subjectOf"].([]interface{}),
		map[string]interface{}{
			"@type":           "earl:Assertion",
			"earl:assertedBy": assertor,
			"earl:mode":       "earl:automatic",
			"earl:test":       testName,
			"earl:result": map[string]interface{}{
				"@type":        "earl:TestResult",
				"dc:date":      time.Now().Format("2006-01-02T15:04:05.999999"),
				"earl:outcome": outcome,
			},
		},
	)
}

func (er *EarlReport) write(filename string) {
	b, _ := json.MarshalIndent(er.report, "", "  ")

	f, _ := os.Create(filename)
	defer f.Close()
	f.Write(b)
	f.WriteString("\n")
}
