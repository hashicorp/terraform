package statecrypto

import (
	"fmt"
	"github.com/hashicorp/terraform/internal/states/statecrypto/cryptoconfig"
	"github.com/hashicorp/terraform/internal/states/statecrypto/implementations/passthrough"
	"log"
	"testing"
)

func resetLogFatalf() {
	logFatalf = log.Fatalf
}

// instance creation

func TestCreation(t *testing.T) {
	cut := StateCryptoWrapper()
	if cut == nil {
		t.Fatal("instance creation failed")
	}
	cutTypecast, ok := cut.(*FallbackRetryStateWrapper)
	if !ok {
		t.Fatal("did not create a FallbackRetryStateWrapper")
	}
	if cutTypecast.fallback != nil {
		t.Fatal("default configuration unexpectedly has a decryption fallback")
	}
	if cutTypecast.firstChoice == nil {
		t.Fatal("default configuration unexpectedly has a nil first choice")
	}
	_, ok = cutTypecast.firstChoice.(*passthrough.PassthroughStateWrapper)
	if !ok {
		t.Fatal("default configuration unexpectedly created something other than passthrough as first choice")
	}
}

func creationErrorCase(t *testing.T, jsonConfig string, expectedError string) {
	var lastError string
	logFatalf = func(format string, v ...interface{}) {
		lastError = fmt.Sprintf(format, v...)
	}
	defer resetLogFatalf()

	lastError = ""
	mainConfig, err := cryptoconfig.Parse(jsonConfig)
	if err != nil {
		t.Fatal("error parsing configuration")
	}

	_ = instanceFromConfig(mainConfig, true)
	if lastError != expectedError {
		t.Errorf("got wrong error during instance creation '%s', expected '%s'", lastError, expectedError)
	}
}

const invalidConfigUnknownImpl = `{"implementation":"something-unknown","parameters":{"key":"a0a1a2a3a4a5a6a7a8a9b0b1b2b3b4b5b6b7b8b9c0c1c2c3c4c5c6c7c8c9d0d1"}}`
const invalidConfigNoKey = `{"implementation":"client-side/AES256-cfb/SHA256","parameters":{}}`

func TestCreation_invalidConfigUnknownImpl(t *testing.T) {
	creationErrorCase(t, invalidConfigUnknownImpl, "[ERROR] failed to configure remote state encryption: unsupported implementation 'something-unknown'")
}

func TestCreation_invalidConfigNoKey(t *testing.T) {
	creationErrorCase(t, invalidConfigNoKey, "[ERROR] failed to configure remote state encryption: configuration for AES256 needs the parameter 'key' set to a 32 byte lower case hexadecimal value")
}

// business scenarios

const validConfigWithKey1 = `{"implementation":"client-side/AES256-cfb/SHA256","parameters":{"key":"a0a1a2a3a4a5a6a7a8a9b0b1b2b3b4b5b6b7b8b9c0c1c2c3c4c5c6c7c8c9d0d1"}}`
const validConfigWithKey2 = `{"implementation":"client-side/AES256-cfb/SHA256","parameters":{"key":"89346775897897a35892735ffd34723489734ee238748293741abcdef0123456"}}`
const validConfigWithKey3 = `{"implementation":"client-side/AES256-cfb/SHA256","parameters":{"key":"33336775897897a35892735ffd34723489734ee238748293741abcdef0123456"}}`

const validPlaintext = `{"animals":[{"species":"cheetah","genus":"acinonyx"}]}`
const validEncryptedKey1 = `{"crypted":"e93e3e7ad3434055251f695865a13c11744b97e54cb7dee8f8fb40d1fb096b728f2a00606e7109f0720aacb15008b410cf2f92dd7989c2ff10b9712b6ef7d69ecdad1dccd2f1bddd127f0f0d87c79c3c062e03c2297614e2effa2fb1f4072d86df0dda4fc061"}`

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

type roundtripTestCase struct {
	description           string
	mainConfiguration     string
	fallbackConfiguration string
	input                 string
	injectOutput          string
	expectedEncError      string
	expectedDecError      string
}

func TestEncryptDecrypt(t *testing.T) {
	// each test case first encrypts, then decrypts again
	testCases := []roundtripTestCase{
		// happy path cases
		{
			description: "unencrypted operation - no encryption configuration present, no fallback",
			input:       validPlaintext,
		},
		{
			description:       "normal operation on encrypted data - main configuration for aes256, no fallback",
			mainConfiguration: validConfigWithKey1,
			input:             validPlaintext,
		},
		{
			description:       "initial encryption - main configuration for aes256, no fallback - prints a warning but must work anyway",
			mainConfiguration: validConfigWithKey1,
			input:             validPlaintext,
			injectOutput:      validPlaintext,
		},
		{
			description:           "decryption - no main configuration, fallback aes256",
			fallbackConfiguration: validConfigWithKey1,
			input:                 validPlaintext, // exact value irrelevant for this test case
			injectOutput:          validEncryptedKey1,
		},
		{
			description:           "unencrypted operation with fallback still present (decryption edge case) - no encryption configuration present, fallback aes256 - prints a warning but must still work anyway",
			input:                 validPlaintext,
			fallbackConfiguration: validConfigWithKey1,
		},
		{
			description:           "key rotation - main configuration for aes256 key 2, fallback aes256 key 1, read state with key 1 encryption - prints a warning but must work anyway",
			mainConfiguration:     validConfigWithKey2,
			fallbackConfiguration: validConfigWithKey1,
			input:                 validPlaintext, // exact value irrelevant for this test case
			injectOutput:          validEncryptedKey1,
		},
		{
			description:           "key rotation - main configuration for aes256 key 2, fallback aes256 key 1, read state with key 2 encryption",
			mainConfiguration:     validConfigWithKey2,
			fallbackConfiguration: validConfigWithKey1,
			input:                 validPlaintext,
		},
		{
			description:           "initial encryption happens during key rotation (key rotation edge case) - main configuration for aes256 key 1, fallback for aes256 key 2 - prints a warning but must still work anyway",
			mainConfiguration:     validConfigWithKey1,
			fallbackConfiguration: validConfigWithKey2,
			input:                 validPlaintext, // exact value irrelevant for this test case
			injectOutput:          validPlaintext,
		},

		// error cases
		{
			description:       "decryption fails due to wrong key - main configuration for aes256 key 3 - but state was encrypted with key 1",
			mainConfiguration: validConfigWithKey3,
			input:             validPlaintext, // exact value irrelevant for this test case
			injectOutput:      validEncryptedKey1,
			expectedDecError:  "hash of decrypted payload did not match at position 0",
		},
		{
			description:           "decryption fails due to wrong fallback key during decrypt lifecycle - no main configuration, fallback configuration for aes256 key 3 - but state was encrypted with key 1 - must fail and not use passthrough",
			fallbackConfiguration: validConfigWithKey3,
			input:                 validPlaintext, // exact value irrelevant for this test case
			injectOutput:          validEncryptedKey1,
			expectedDecError:      "hash of decrypted payload did not match at position 0",
		},
		{
			description:           "decryption fails due to two wrong keys - main configuration for aes256 key 3, fallback for aes256 key 2 - but state was encrypted with key 1",
			mainConfiguration:     validConfigWithKey3,
			fallbackConfiguration: validConfigWithKey2,
			input:                 validPlaintext, // exact value irrelevant for this test case
			injectOutput:          validEncryptedKey1,
			expectedDecError:      "hash of decrypted payload did not match at position 0",
		},
	}

	var lastError string
	logFatalf = func(format string, v ...interface{}) {
		lastError = fmt.Sprintf(format, v...)
	}
	defer resetLogFatalf()

	for _, tc := range testCases {
		log.Printf("test case: %s", tc.description)

		lastError = ""
		mainConfig, err := cryptoconfig.Parse(tc.mainConfiguration)
		if err != nil {
			t.Fatal("error parsing main configuration")
		}
		fallbackConfig, err := cryptoconfig.Parse(tc.fallbackConfiguration)
		if err != nil {
			t.Fatal("error parsing fallback configuration")
		}
		cut := fallbackRetryInstance(
			instanceFromConfig(mainConfig, true),
			instanceFromConfig(fallbackConfig, false),
		)
		if lastError != "" {
			t.Error("skipping test case, got error during instance creation: " + lastError)
		} else {
			if cut == nil {
				t.Error("got unexpected nil implementation")
			} else {
				roundtripTestcase(t, cut, tc)
			}
		}
	}
}

func roundtripTestcase(t *testing.T, cut StateCryptoProvider, tc roundtripTestCase) {
	encOutput, err := cut.Encrypt([]byte(tc.input))
	if comp := compareErrors(err, tc.expectedEncError); comp != "" {
		t.Error(comp)
	} else {
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
