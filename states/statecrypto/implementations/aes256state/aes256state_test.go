package aes256state

import (
	"encoding/hex"
	"fmt"
	"github.com/hashicorp/terraform/states/statecrypto/cryptoconfig"
	"testing"
)

const validKey1 = "a0a1a2a3a4a5a6a7a8a9b0b1b2b3b4b5b6b7b8b9c0c1c2c3c4c5c6c7c8c9d0d1"

const tooShortKey = "a0a1a2a3a4a5a6a7a8a9b0b1b2b3b4b5b6b7b8b9c0c1c2c3c4c5c6c7c8c9"
const tooLongKey = "a0a1a2a3a4a5a6a7a8a9b0b1b2b3b4b5b6b7b8b9c0c1c2c3c4c5c6c7c8c9d0d1d2d3d4d5"
const invalidChars = "somethingsomethinga9b0b1b2b3b4b5b6b7b8b9c0c1c2c3c4c5c6c7c8c9d0d1"

const validPlaintext = `{"animals":[{"species":"cheetah","genus":"acinonyx"}]}`
const validEncryptedKey1 = `{"crypted":"e93e3e7ad3434055251f695865a13c11744b97e54cb7dee8f8fb40d1fb096b728f2a00606e7109f0720aacb15008b410cf2f92dd7989c2ff10b9712b6ef7d69ecdad1dccd2f1bddd127f0f0d87c79c3c062e03c2297614e2effa2fb1f4072d86df0dda4fc061"}`
const invalidEncryptedHash = `{"crypted":"a6625332f6e3061e1202cea86d2ddf7cf6d5f296a9856fe989cd20b18c8522f670d368f523481876bb2b98eea1e8cf845b4e003de11153bc47b884ce907b1e6a075f515ddd2aa4fbdbc7bbab1b411e153d164f84990e9c6fa82d7cacde7401546b47b2f30000"}`
const invalidEncryptedCutoff = `{"crypted":"447c2fc8982ed203681298be9f1b03ed30dbfe794a68e4ad873fb68c34f10394ffddd9c76b2d3fdb006d75068453854af63766fc059a569d243eb7d8c92ec3a00535ccaab769bdafb534d5471ed01ca36f640d1f`
const invalidEncryptedChars = `{"crypted":"447c2fc8982ed203681298be9f1b03ed30dbfe794a68e4ad873fb68c34 SOMETHING WEIRD d3fdb006d75068453854af63766fc059a569d243eb7d8c92ec3a00535ccaab769bdafb534d5471ed01ca36f640d1f720c9a2bf0aa4e0a40496dacee92325a9f86"}`
const invalidEncryptedTooShort = `{"crypted":"a6625332"}`
const invalidEncryptedOddNumberCharacters = `{"crypted":"e93e3e7ad3434055251f695865a13c11744b97e54cb7dee8f8fb40d1fb096b728f2a00606e7109f0720aacb15008b410cf2f92dd7989c2ff10b9712b6ef7d69ecdad1dccd2f1bddd127f0f0d87c79c3c062e03c2297614e2effa2fb1f4072d86df0dda4fc06"}`

type parseKeyTestCase struct {
	description   string
	configuration cryptoconfig.StateCryptoConfig
	expectedError string
	expectedKey   []byte
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

func conf(key string) cryptoconfig.StateCryptoConfig {
	return cryptoconfig.StateCryptoConfig{
		Implementation: "client-side/AES256-cfb/SHA256",
		Parameters: map[string]string{
			"key": key,
		},
	}
}

func TestParseKeysFromConfiguration(t *testing.T) {
	k1, _ := hex.DecodeString(validKey1)

	testCases := []parseKeyTestCase{
		// happy cases
		{
			description:   "work on encrypted state files, no previous key",
			configuration: conf(validKey1),
			expectedKey:   k1,
		},
		{
			description:   "work on encrypted state files, empty previous key",
			configuration: conf(validKey1),
			expectedKey:   k1,
		},

		// error cases
		{
			description:   "key missing",
			configuration: conf(""),
			expectedError: "key was not a hex string representing 32 bytes, must match [0-9a-f]{64}",
		},
		{
			description:   "too short key",
			configuration: conf(tooShortKey),
			expectedError: "key was not a hex string representing 32 bytes, must match [0-9a-f]{64}",
		},
		{
			description:   "too long key",
			configuration: conf(tooLongKey),
			expectedError: "key was not a hex string representing 32 bytes, must match [0-9a-f]{64}",
		},
		{
			description:   "invalid chars in main key",
			configuration: conf(invalidChars),
			expectedError: "key was not a hex string representing 32 bytes, must match [0-9a-f]{64}",
		},
		{
			description:   "parse error",
			configuration: conf(`"`),
			expectedError: "key was not a hex string representing 32 bytes, must match [0-9a-f]{64}",
		},
	}

	for _, tc := range testCases {
		cut := &AES256StateWrapper{}
		err := cut.parseKeyFromConfiguration(tc.configuration)
		if comp := compareErrors(err, tc.expectedError); comp != "" {
			t.Error(comp)
		}
		if !compareSlices(cut.key, tc.expectedKey) {
			t.Errorf("unexpected key %#v; want %#v", cut.key, tc.expectedKey)
		}
	}
}

type roundtripTestCase struct {
	description      string
	configuration    cryptoconfig.StateCryptoConfig
	input            string
	injectOutput     string
	expectedNewError string
	expectedEncError string
	expectedDecError string
}

func TestEncryptDecrypt(t *testing.T) {
	testCases := []roundtripTestCase{
		// happy path cases
		{
			description:   "standard work on encrypted data",
			configuration: conf(validKey1),
			input:         validPlaintext,
		},
		{
			description:   "standard work on unencrypted data",
			configuration: conf(validKey1),
			input:         validPlaintext,
			injectOutput:  validPlaintext,
		},

		// error cases
		{
			description:      "invalid hash received on decrypt",
			configuration:    conf(validKey1),
			input:            validPlaintext,
			injectOutput:     invalidEncryptedHash,
			expectedDecError: "hash of decrypted payload did not match at position 30",
		},
		{
			description:      "decrypt received incomplete crypted json",
			configuration:    conf(validKey1),
			input:            validPlaintext,
			injectOutput:     invalidEncryptedCutoff,
			expectedDecError: "ciphertext contains invalid characters, possibly cut off or garbled",
		},
		{
			description:      "decrypt received invalid crypted json",
			configuration:    conf(validKey1),
			input:            validPlaintext,
			injectOutput:     invalidEncryptedChars,
			expectedDecError: "ciphertext contains invalid characters, possibly cut off or garbled",
		},
		{
			description:      "decrypt received crypted json too short even for iv",
			configuration:    conf(validKey1),
			input:            validPlaintext,
			injectOutput:     invalidEncryptedTooShort,
			expectedDecError: "ciphertext too short, did not contain initial vector",
		},
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
					// log.Printf("crypted json is %s", string(encOutput))

					if tc.injectOutput != "" {
						encOutput = []byte(tc.injectOutput)
					}

					decOutput, err := cut.Decrypt(encOutput)
					if comp := compareErrors(err, tc.expectedDecError); comp != "" {
						t.Error(comp)
					} else {
						if err == nil && !compareSlices(decOutput, []byte(tc.input)) {
							t.Errorf("round trip error, got %#v; want %#v", decOutput, []byte(tc.input))
						}
					}
				}
			}
		}
	}
}

func TestEncryptDoesNotUseSameIV(t *testing.T) {
	cut, _ := New(conf(validKey1))
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

func TestEncrypt_FailingCipherCreation(t *testing.T) {
	cut := &AES256StateWrapper{key: []byte{127, 42}}
	_, err := cut.Encrypt([]byte(validPlaintext))
	if comp := compareErrors(err, "crypto/aes: invalid key size 2"); comp != "" {
		t.Error(comp)
	}
}

func TestAttemptDecryption_FailingCipherCreation(t *testing.T) {
	cut := &AES256StateWrapper{key: []byte{127, 42}}
	_, err := cut.attemptDecryption([]byte(validEncryptedKey1), cut.key)
	if comp := compareErrors(err, "crypto/aes: invalid key size 2"); comp != "" {
		t.Error(comp)
	}
}

func TestAttemptDecryption_InvalidHexadecimal(t *testing.T) {
	cut := &AES256StateWrapper{}
	_, err := cut.attemptDecryption([]byte(invalidEncryptedOddNumberCharacters), cut.key)
	if comp := compareErrors(err, "encoding/hex: odd length hex string"); comp != "" {
		t.Error(comp)
	}
}
