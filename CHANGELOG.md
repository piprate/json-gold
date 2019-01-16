# JSON-goLD Change Log

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
