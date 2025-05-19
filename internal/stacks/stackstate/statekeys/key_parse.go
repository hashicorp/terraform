// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package statekeys

import (
	"fmt"
)

// Parse attempts to parse the given string as a state key, and returns the
// result if successful.
//
// A returned error means that the given string is syntactically invalid,
// which could mean either that it doesn't meet the basic requirements for
// any state key, or that it has a recognized key type but the remainder is
// not valid for that type.
//
// Parse DOES NOT return an error for a syntactically-valid key of an
// unrecognized type. Instead, it returns an [UnrecognizedKey] value which
// callers can detect using [RecognizedType], which will return false for
// a key of an unrecognized type.
func Parse(raw string) (Key, error) {
	if len(raw) < 4 {
		// All state keys must have at least four characters, since that's
		// how long a key prefix is.
		return nil, fmt.Errorf("too short to be a valid state key")
	}
	keyType := KeyType(raw[:4])
	remain := raw[4:]
	parser := keyParsers[keyType]
	if parser == nil {
		if !isPlausibleRawKeyType(string(keyType)) {
			return nil, fmt.Errorf("invalid key type prefix %q", keyType)
		}
		return Unrecognized{
			ApparentKeyType: keyType,
			remainder:       remain,
		}, nil
	}
	return parser(remain)
}

var keyParsers = map[KeyType]func(string) (Key, error){
	ResourceInstanceObjectType: parseResourceInstanceObject,
	ComponentInstanceType:      parseComponentInstance,
	OutputType:                 parseOutput,
	VariableType:               parseVariable,
}

// cutKeyField is a key parsing helper for key types that consist of
// multiple fields concatenated together.
//
// cutKeyField returns the raw string content of the next field, and
// also returns any remaining text after the field delimeter which
// could therefore be used in a subsequent call to cutKeyField.
//
// The field delimiter is a comma, but the parser ignores any comma
// that appears to be inside a pair of double-quote characters (")
// so that it's safe to include an address with a string-based instance key
// (which could potentially contain a literal comma) and get back that same
// address as a single field.
//
// If the given string does not contain any delimiters, the result is the
// same string verbatim and an empty "remain" result.
func cutKeyField(raw string) (field, remain string) {
	i := keyDelimiterIdx(raw)
	if i == -1 {
		return raw, ""
	}
	return raw[:i], raw[i+1:]
}

// finalKeyField returns the given string and true if it doesn't contain a key
// field delimiter, or "", false if the string does have a delimiter.
func finalKeyField(raw string) (string, bool) {
	i := keyDelimiterIdx(raw)
	if i != -1 {
		return "", false
	}
	return raw, true
}

// keyDelimiterIdx finds the index of the first delimiter in the given
// string, or returns -1 if there is no delimiter in the string.
func keyDelimiterIdx(raw string) int {
	inQuotes := false
	escape := false
	for i, c := range raw {
		if c == ',' && !inQuotes {
			return i
		}
		if c == '\\' {
			escape = true
			continue
		}
		if c == '"' && !escape {
			inQuotes = !inQuotes
		}
		escape = false
	}
	// If we fall out here then the entire string seems to be
	// a single field, with no delimiters.
	return -1
}
