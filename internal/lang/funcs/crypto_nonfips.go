//go:build !fips

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package funcs

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"

	"github.com/zclconf/go-cty/cty/function"
)

// Md5Func constructs a function that computes the MD5 hash of a given string and encodes it with hexadecimal digits.
var Md5Func = makeStringHashFunction(md5.New, hex.EncodeToString)

// MakeFileMd5Func constructs a function that is like Md5Func but reads the
// contents of a file rather than hashing a given literal string.
func MakeFileMd5Func(baseDir string, wrap ImplWrapper) function.Function {
	return makeFileHashFunction(baseDir, md5.New, hex.EncodeToString, wrap)
}

// Sha1Func contructs a function that computes the SHA1 hash of a given string
// and encodes it with hexadecimal digits.
var Sha1Func = makeStringHashFunction(sha1.New, hex.EncodeToString)

// MakeFileSha1Func constructs a function that is like Sha1Func but reads the
// contents of a file rather than hashing a given literal string.
func MakeFileSha1Func(baseDir string, wrap ImplWrapper) function.Function {
	return makeFileHashFunction(baseDir, sha1.New, hex.EncodeToString, wrap)
}
