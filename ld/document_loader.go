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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
)

const (
	// An HTTP Accept header that prefers JSONLD.
	acceptHeader = "application/ld+json, application/json;q=0.9, application/javascript;q=0.5, text/javascript;q=0.5, text/plain;q=0.2, */*;q=0.1"

	// JSON-LD link header rel
	linkHeaderRel = "http://www.w3.org/ns/json-ld#context"
)

// RemoteDocument is a document retrieved from a remote source.
type RemoteDocument struct {
	DocumentURL string
	Document    interface{}
	ContextURL  string
}

// DocumentLoader knows how to load remote documents.
type DocumentLoader interface {
	LoadDocument(u string) (*RemoteDocument, error)
}

// DefaultDocumentLoader is a standard implementation of DocumentLoader
// which can retrieve documents via HTTP.
type DefaultDocumentLoader struct {
	httpClient *http.Client
}

// NewDefaultDocumentLoader creates a new instance of DefaultDocumentLoader
func NewDefaultDocumentLoader(httpClient *http.Client) *DefaultDocumentLoader {
	rval := &DefaultDocumentLoader{httpClient: httpClient}

	if rval.httpClient == nil {
		rval.httpClient = http.DefaultClient
	}
	return rval
}

// DocumentFromReader returns a document containing the contents of the JSON resource,
// streamed from the given Reader.
func DocumentFromReader(r io.Reader) (interface{}, error) {
	var document interface{}
	dec := json.NewDecoder(r)

	// If dec.UseNumber() were invoked here, all numbers would be decoded as json.Number.
	// json-gold supports both the default and json.Number options.

	if err := dec.Decode(&document); err != nil {
		return nil, NewJsonLdError(LoadingDocumentFailed, err)
	}
	return document, nil
}

// LoadDocument returns a RemoteDocument containing the contents of the JSON resource
// from the given URL.
func (dl *DefaultDocumentLoader) LoadDocument(u string) (*RemoteDocument, error) {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return nil, NewJsonLdError(LoadingDocumentFailed, err)
	}

	var documentBody io.Reader
	var finalURL, contextURL string

	protocol := parsedURL.Scheme
	if protocol != "http" && protocol != "https" {
		// Can't use the HTTP client for those!
		finalURL = u
		var file *os.File
		file, err = os.Open(u)
		if err != nil {
			return nil, NewJsonLdError(LoadingDocumentFailed, err)
		}
		defer file.Close()
		documentBody = file
	} else {

		req, err := http.NewRequest("GET", u, nil)
		// We prefer application/ld+json, but fallback to application/json
		// or whatever is available
		req.Header.Add("Accept", acceptHeader)

		res, err := dl.httpClient.Do(req)

		if err != nil {
			return nil, NewJsonLdError(LoadingDocumentFailed, err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return nil, NewJsonLdError(LoadingDocumentFailed,
				fmt.Sprintf("Bad response status code: %d", res.StatusCode))
		}

		finalURL = res.Request.URL.String()

		contentType := res.Header.Get("Content-Type")
		linkHeader := res.Header.Get("Link")

		if len(linkHeader) > 0 && contentType != "application/ld+json" {
			header := ParseLinkHeader(linkHeader)[linkHeaderRel]
			if len(header) > 1 {
				return nil, NewJsonLdError(MultipleContextLinkHeaders, nil)
			} else if len(header) == 1 {
				contextURL = header[0]["target"]
			}
		}

		documentBody = res.Body
	}
	if err != nil {
		return nil, NewJsonLdError(LoadingDocumentFailed, err)
	}
	document, err := DocumentFromReader(documentBody)
	if err != nil {
		return nil, err
	}
	return &RemoteDocument{DocumentURL: finalURL, Document: document, ContextURL: contextURL}, nil
}

var rSplitOnComma = regexp.MustCompile("(?:<[^>]*?>|\"[^\"]*?\"|[^,])+")
var rLinkHeader = regexp.MustCompile("\\s*<([^>]*?)>\\s*(?:;\\s*(.*))?")
var rParams = regexp.MustCompile("(.*?)=(?:(?:\"([^\"]*?)\")|([^\"]*?))\\s*(?:(?:;\\s*)|$)")

// ParseLinkHeader parses a link header. The results will be keyed by the value of "rel".
//
// Link: <http://json-ld.org/contexts/person.jsonld>; \
//   rel="http://www.w3.org/ns/json-ld#context"; type="application/ld+json"
//
// Parses as: {
//   'http://www.w3.org/ns/json-ld#context': {
//     target: http://json-ld.org/contexts/person.jsonld,
//     rel:    http://www.w3.org/ns/json-ld#context
//   }
// }
//
// If there is more than one "rel" with the same IRI, then entries in the
// resulting map for that "rel" will be lists.
func ParseLinkHeader(header string) map[string][]map[string]string {

	rval := make(map[string][]map[string]string)

	// split on unbracketed/unquoted commas
	entries := rSplitOnComma.FindAllString(header, -1)
	if len(entries) == 0 {
		return rval
	}

	for _, entry := range entries {
		if !rLinkHeader.MatchString(entry) {
			continue
		}
		match := rLinkHeader.FindStringSubmatch(entry)

		result := map[string]string{
			"target": match[1],
		}
		params := match[2]
		matches := rParams.FindAllStringSubmatch(params, -1)
		for _, match := range matches {
			if match[2] == "" {
				result[match[1]] = match[3]
			} else {
				result[match[1]] = match[2]
			}
		}
		rel := result["rel"]
		relVal, hasRel := rval[rel]
		if hasRel {
			rval[rel] = append(relVal, result)
		} else {
			rval[rel] = []map[string]string{result}
		}
	}
	return rval
}

// CachingDocumentLoader is an overlay on top of DocumentLoader instance
// which allows caching documents as soon as they get retrieved
// from the underlying loader. You may also preload it with documents -
// this is useful for testing.
type CachingDocumentLoader struct {
	nextLoader DocumentLoader
	cache      map[string]*RemoteDocument
}

// NewCachingDocumentLoader creates a new instance of CachingDocumentLoader.
func NewCachingDocumentLoader(nextLoader DocumentLoader) *CachingDocumentLoader {
	rval := &CachingDocumentLoader{
		nextLoader: nextLoader,
		cache:      make(map[string]*RemoteDocument),
	}

	return rval
}

// LoadDocument returns a RemoteDocument containing the contents of the JSON resource
// from the given URL.
func (cdl *CachingDocumentLoader) LoadDocument(u string) (*RemoteDocument, error) {
	if doc, cached := cdl.cache[u]; cached {
		return doc, nil
	} else {
		doc, err := cdl.nextLoader.LoadDocument(u)
		if err != nil {
			return nil, err
		}
		cdl.cache[u] = doc
		return doc, nil
	}
}

// AddDocument populates the cache with the given document (doc) for the provided URL (u).
func (cdl *CachingDocumentLoader) AddDocument(u string, doc interface{}) {
	cdl.cache[u] = &RemoteDocument{DocumentURL: u, Document: doc, ContextURL: ""}
}

// PreloadWithMapping populates the cache with a number of documents which may be loaded
// from location different from the original URL (most importantly, from local files).
//
// Example:
//     l.PreloadWithMapping(map[string]string{
//         "http://www.example.com/context.json": "/home/me/cache/example_com_context.json",
//     })
//
func (cdl *CachingDocumentLoader) PreloadWithMapping(urlMap map[string]string) error {
	for srcURL, mappedURL := range urlMap {
		doc, err := cdl.nextLoader.LoadDocument(mappedURL)
		if err != nil {
			return err
		}
		cdl.cache[srcURL] = doc
	}
	return nil
}
