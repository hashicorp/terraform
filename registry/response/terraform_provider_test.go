package response

import (
	"fmt"
	"testing"
)

var (
	testGPGKeyOne = &GPGKey{
		ASCIIArmor: "---\none\n---",
	}
	testGPGKeyTwo = &GPGKey{
		ASCIIArmor: "---\ntwo\n---",
	}
)

func TestSigningKeyList_GPGASCIIArmor(t *testing.T) {
	var tests = []struct {
		name     string
		gpgKeys  []*GPGKey
		expected string
	}{
		{
			name:     "no keys",
			gpgKeys:  []*GPGKey{},
			expected: "",
		},
		{
			name:     "one key",
			gpgKeys:  []*GPGKey{testGPGKeyOne},
			expected: testGPGKeyOne.ASCIIArmor,
		},
		{
			name:    "two keys",
			gpgKeys: []*GPGKey{testGPGKeyOne, testGPGKeyTwo},
			expected: fmt.Sprintf("%s\n%s",
				testGPGKeyOne.ASCIIArmor, testGPGKeyTwo.ASCIIArmor),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signingKeys := &SigningKeyList{
				GPGKeys: tt.gpgKeys,
			}
			actual := signingKeys.GPGASCIIArmor()

			if actual != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, actual)
			}
		})
	}
}
