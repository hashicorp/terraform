// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"fmt"
	"strings"
	"testing"
)

func TestIsValidKeyVaultSecretURI(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Valid URIs
		{"https://mykeyvault.vault.azure.net/secrets/mysecret", true},
		{"https://another-vault.vault.azure.net/secrets/another-secret", true},
		{"https://vault123.vault.azure.net/secrets/secret123/version456", true},

		// Invalid URIs
		{"https://example.com/not-keyvault-uri", false},
		{"ftp://mykeyvault.vault.azure.net/secrets/mysecret", false},
		{"https://mykeyvault.vault.azure.net/secrets/mysecret/version456/extra", false},
		{"https://invalidvaultname.123.vault.azure.net/secrets/secret123", false},
		{"https://mykeyvault.vault.azure.net/secrets/invalid-secret!@#", false},
		{"https://mykeyvault.vault.azure.net/secrets/secret123/invalid-version!@#", false},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Testing URI: %s", test.input), func(t *testing.T) {
			result := IsValidKeyVaultSecretURI(test.input)
			if result != test.expected {
				t.Fatalf("Expected result: %t, Got: %t", test.expected, result)
			}
		})
	}
}

// These tests also cover calculateSHA256() func
func TestSetEncryptionHeaders(t *testing.T) {
	tests := []struct {
		secretValue         string
		expectedHeaders     map[string]interface{}
		expectedErrorString string
	}{
		// Valid secret value
		{
			secretValue: "7465737456616c7565", // Hex encoding of "testValue"
			expectedHeaders: map[string]interface{}{
				cmkEncryptionAlgorithmHeader: encryptionAlgorithm,
				cmkEncryptionKeyHeader:       "dGVzdFZhbHVl",                                 // Base64 encoding of "testValue"
				cmkEncryptionKeySHA256Header: "gv4Mg0y+oGkBPF63go5Zmmk+DSQRiH4qsnMnFmKXMII=", // SHA256 hash of "testValue"
			},
			expectedErrorString: "",
		},

		// Invalid hex encoding
		{
			secretValue:         "invalidHex",
			expectedHeaders:     nil,
			expectedErrorString: "Error decoding keyvault secret from hex",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Testing with secret value: %s", test.secretValue), func(t *testing.T) {
			headers, err := setEncryptionHeaders(test.secretValue)

			if test.expectedErrorString != "" {
				// Expecting an error
				if err == nil && strings.Contains(err.Error(), test.expectedErrorString) {
					t.Fatalf("Expected error: %s, Got: %v", test.expectedErrorString, err)
				}
			} else {
				// Expecting no error
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				// Compare headers
				if !areHeadersEqual(headers, test.expectedHeaders) {
					t.Fatalf("Expected headers: %v, Got: %v", test.expectedHeaders, headers)
				}
			}
		})
	}
}

// areHeadersEqual checks if two maps of headers are equal
func areHeadersEqual(headers1, headers2 map[string]interface{}) bool {
	if len(headers1) != len(headers2) {
		return false
	}

	for key, value1 := range headers1 {
		value2, exists := headers2[key]
		if !exists || value1 != value2 {
			return false
		}
	}

	return true
}
