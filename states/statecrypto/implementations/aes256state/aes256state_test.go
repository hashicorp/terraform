package aes256state

import (
	"encoding/hex"
	"fmt"
	"log"
	"testing"
)

const validKey1 = "a0a1a2a3a4a5a6a7a8a9b0b1b2b3b4b5b6b7b8b9c0c1c2c3c4c5c6c7c8c9d0d1"
const validKey2 = "89346775897897a35892735ffd34723489734ee238748293741abcdef0123456"

const tooShortKey = "a0a1a2a3a4a5a6a7a8a9b0b1b2b3b4b5b6b7b8b9c0c1c2c3c4c5c6c7c8c9"
const tooLongKey = "a0a1a2a3a4a5a6a7a8a9b0b1b2b3b4b5b6b7b8b9c0c1c2c3c4c5c6c7c8c9d0d1d2d3d4d5"
const invalidChars = "somethingsomethinga9b0b1b2b3b4b5b6b7b8b9c0c1c2c3c4c5c6c7c8c9d0d1"

const validPlaintext = `{"animals":[{"species":"cheetah","genus":"acinonyx"}]}`
const validEncryptedKey1 = `{"crypted":"e2222f79474c4a61c4407cd73711297c39638850298fa5152b9a5e4d15a994de77d8bdb5ad47af51793f83b0bd838775990c454015244dd8a37e145f2cd5ee859b1f6a9697d7"}`

type parseKeysTestCase struct {
	description       string
	configuration     []string
	expectedError     string
	expectedKey       []byte
	expectedPrvKey    []byte
}

func compareSlices(got []byte, expected []byte) bool {
	eEmpty := expected == nil || len(expected) == 0
	gEmpty := got == nil || len(got) == 0
	if eEmpty != gEmpty {
		return false
	}
	if eEmpty {
		return true
	}
	if len(expected) != len(got) {
		return false
	}
	for i, v := range expected {
		if v != got[i] {
			return false
		}
	}
	return true
}

func compareErrors(got error, expected string) string {
	if got != nil {
		if got.Error() != expected {
			return fmt.Sprintf("unexpected error '%s'; want '%s'", got.Error(), expected)
		}
	} else {
		if expected != "" {
			return fmt.Sprintf("did not get expected error '%s'", expected)
		}
	}
	return ""
}

func TestParseKeysFromConfiguration(t *testing.T) {
	k1, _ := hex.DecodeString(validKey1)
	k2, _ := hex.DecodeString(validKey2)

	testCases := []parseKeysTestCase{
		// happy cases
		{
			description: "work on encrypted state files, no previous key",
			configuration: []string{"AES256", validKey1},
			expectedKey:   k1,
		},
		{
			description: "work on encrypted state files, empty previous key",
			configuration: []string{"AES256", validKey1, ""},
			expectedKey:   k1,
		},
		{
			description: "key rotation case (with previous key allowed for reading)",
			configuration: []string{"AES256", validKey1, validKey2},
			expectedKey:   k1,
			expectedPrvKey: k2,
		},
		{
			description: "decryption case",
			configuration: []string{"AES256", "", validKey2},
			expectedPrvKey: k2,
		},

		// error cases
		{
			description: "too few parts of configuration",
			configuration: []string{"AES256"},
			expectedError: "configuration for AES256 needs to be AES256:key[:previousKey] where keys are 32 byte lower case hexadecimals and previous key is optional",
		},
		{
			description: "too many parts of configuration",
			configuration: []string{"AES256", "", "", ""},
			expectedError: "configuration for AES256 needs to be AES256:key[:previousKey] where keys are 32 byte lower case hexadecimals and previous key is optional",
		},
		{
			description: "too short main key",
			configuration: []string{"AES256", tooShortKey},
			expectedError: "main key was not a hex string representing 32 bytes, must match [0-9a-f]{64}",
		},
		{
			description: "too long previous key",
			configuration: []string{"AES256", validKey1, tooLongKey},
			expectedError: "previous key was not a hex string representing 32 bytes, must match [0-9a-f]{64}",
			expectedKey:   k1,
		},
		{
			description: "invalid chars in main key",
			configuration: []string{"AES256", invalidChars},
			expectedError: "main key was not a hex string representing 32 bytes, must match [0-9a-f]{64}",
		},
	}

	for _, tc := range testCases {
		cut := &AES256StateWrapper{}
		err := cut.parseKeysFromConfiguration(tc.configuration)
		if comp := compareErrors(err, tc.expectedError); comp != "" {
			t.Error(comp)
		}
		if !compareSlices(cut.key, tc.expectedKey) {
			t.Errorf("unexpected key %#v; want %#v", cut.key, tc.expectedKey)
		}
		if !compareSlices(cut.previousKey, tc.expectedPrvKey) {
			t.Errorf("unexpected key %#v; want %#v", cut.previousKey, tc.expectedPrvKey)
		}
	}
}

type roundtripTestCase struct {
	description       string
	configuration     []string
	input             string
	expectedNewError  string
	expectedEncError  string
	expectedDecError  string
}

func TestEncryptDecrypt(t *testing.T) {
	testCases := []roundtripTestCase{
		// happy path cases
		{
			description:   "standard work on encrypted data",
			configuration: []string{"AES256", validKey1, ""},
			input:         validPlaintext,
		},
		{
			description:   "no keys either direction",
			configuration: []string{"AES256", "", ""},
			input:         validPlaintext,
		},

		// error cases
	}
	for _, tc := range testCases {
		cut, err := New(tc.configuration)
		if comp := compareErrors(err, tc.expectedNewError); comp != "" {
			t.Error(comp)
		}
		if err == nil {
			if cut == nil {
				t.Error("got unexpected nil implementation")
			} else {
				encOutput, err := cut.Encrypt([]byte(tc.input))
				if comp := compareErrors(err, tc.expectedEncError); comp != "" {
					t.Error(comp)
				} else {
					log.Printf("crypted json is %s", string(encOutput))

					decOutput, err := cut.Decrypt(encOutput)
					if comp := compareErrors(err, tc.expectedDecError); comp != "" {
						t.Error(comp)
					} else {
						if !compareSlices(decOutput, []byte(tc.input)) {
							t.Errorf("round trip error, got %#v; want %#v", decOutput, []byte(tc.input))
						}
					}
				}
			}
		}
	}
}

func TestEncryptDoesNotUseSameIV(t *testing.T) {
	cut, _ := New([]string{"AES256", validKey1, ""})
	encOutput1, _ := cut.Encrypt([]byte(validPlaintext))
	if len(encOutput1) != len([]byte(validEncryptedKey1)) {
		t.Error("encryption output 1 did not have the expected length")
	}
	encOutput2, _ := cut.Encrypt([]byte(validPlaintext))
	if len(encOutput2) != len([]byte(validEncryptedKey1)) {
		t.Error("encryption output 2 did not have the expected length")
	}
	if compareSlices(encOutput1, []byte(validEncryptedKey1)) {
		t.Error("random iv created same vector as in recorded run! SECURITY PROBLEM!")
	}
	if compareSlices(encOutput1, encOutput2) {
		t.Error("random iv created same vector as in previous call! SECURITY PROBLEM!")
	}
}