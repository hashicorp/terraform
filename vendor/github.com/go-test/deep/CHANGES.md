# go-test/deep Changelog

## v1.0.3

* Fixed issue #31: panic on typed primitives that implement error interface

## v1.0.2 released 2019-07-14

* Enabled Go module (@radeksimko)
* Changed supported and tested Go versions: 1.10, 1.11, and 1.12 (dropped 1.9)
* Changed Error equality: additional struct fields are compared too (PR #29) (@andrewmostello)
* Fixed typos and ineffassign issues (PR #25) (@tariq1890)
* Fixed diff order for nil comparison (PR #16) (@gmarik)
* Fixed slice equality when slices are extracted from the same array (PR #11) (@risteli)
* Fixed test spelling and messages (PR #19) (@sofuture)
* Fixed issue #15: panic on comparing struct with anonymous time.Time
* Fixed issue #18: Panic when comparing structs with time.Time value and CompareUnexportedFields is true
* Fixed issue #21: Set default MaxDepth = 0 (disabled) (PR #23)

## v1.0.1 released 2018-01-28

* Fixed issue #12: Arrays are not properly compared (@samlitowitz)

## v1.0.0 releaesd 2017-10-27 

* First release
