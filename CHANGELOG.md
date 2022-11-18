# JSON-goLD Change Log

## v0.5.0 - 2022-11-18

- Add GitHub workflows for CI
- Bump the package to Go 1.18
- Address linter feedback, including regexes
- New option: Safe Mode. If set to `true`, Expansion operation will fail if it encounters properties that aren't defined in the context

## v0.4.2 - 2022-10-04

- Move initialization of rxURL from global to function scope to reduce stack size usage while initialization

## v0.4.1 - 2021-12-20

- Bug fix: float to int conversion in 32-bit systems while transforming object to RDF
- Performance improvements for processing big documents

## v0.4.0 - 2021-03-03

- Partial support for JSON literals (`@json`)
- Performance improvements
- Minor bug fixes

## v0.3.0 - 2020-01-08

- Substantial conformance to the latest W3C Recommendation for JSON-LD 1.1 specification.
- Default processing mode set to JSON-LD 1.1

### IMPORTANT NOTES

- JSON-LD 1.1 introduces several changes in internal (and some external) interfaces
- JSON-LD 1.1 algorithms are considerably more complex than 1.0. Performance impact hasn't been evaluated yet. There were no attempts yet to optimise the implementation.

## v0.2.0 - 2019-01-16

- JSON 1.1 support
  - **Breaking interface change**: JsonLdOptions.Embed is now _string_ (used to be _bool_)
- Do not use native types to create IRIs in value expansion.
- Improve acceptable N-Quads blank node labels.
- Compact the @context array if it contains a single element
- Fix a bug which omitted the context if provided in a URL form
- Provide error message when output form for FromRDF operation is unknown
- Pass array compaction flag to compaction inside of framing

## v0.1.1 - 2018-12-12

- RFC7324 compliant caching
- Go 1.11 module support

## v0.1.0 - 2017-12-24

- Copy the [original library](https://github.com/kazarena/json-gold) under the following terms:
                                                                     
  - @piprate team will be the maintainer of the new library (github.com/piprate/json-gold)
  - The original repo (github.com/kazarena/json-gold) will remain available
  - Interfaces of the new library will be preserved, but may deviate in future versions
  - Licensing will not change
  - Past contributors will be recognised
  - Commit history will not be preserved in the new library
  - Versions of the new library will be reset
  
  See the full announcement [here](https://github.com/kazarena/json-gold/issues/20).
