// Package msgpack provides functions for serializing cty values in the
// msgpack encoding, and decoding them again.
//
// If the same type information is provided both at encoding and decoding time
// then values can be round-tripped without loss, except for capsule types
// which are not currently supported.
//
// If any unknown values are passed to Marshal then they will be represented
// using a msgpack extension with type code zero, which is understood by
// the Unmarshal function within this package but will not be understood by
// a generic (non-cty-aware) msgpack decoder. Ensure that no unknown values
// are used if interoperability with other msgpack implementations is
// required.
package msgpack
