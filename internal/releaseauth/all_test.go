// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package releaseauth

import (
	"os"
	"testing"
)

func TestAll(t *testing.T) {
	// `sha256sum testdata/sample_release/sample_0.1.0_darwin_amd64.zip | cut -d' ' -f1`
	actualChecksum, err := SHA256FromHex("22db2f0c70b50cff42afd4878fea9f6848a63f1b6532bd8b64b899f574acb35d")
	if err != nil {
		t.Fatal(err)
	}
	sums, err := os.ReadFile("testdata/sample_release/sample_0.1.0_SHA256SUMS")
	if err != nil {
		t.Fatal(err)
	}
	signature, err := os.ReadFile("testdata/sample_release/sample_0.1.0_SHA256SUMS.sig")
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := os.ReadFile("testdata/sample.public.key")
	if err != nil {
		t.Fatal(err)
	}

	sigAuth := NewSignatureAuthentication(signature, sums)
	sigAuth.PublicKey = string(publicKey)

	all := AllAuthenticators(
		NewChecksumAuthentication(actualChecksum, "testdata/sample_release/sample_0.1.0_darwin_amd64.zip"),
		sigAuth,
	)

	if err := all.Authenticate(); err != nil {
		t.Fatal(err)
	}
}
