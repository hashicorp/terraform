package passthrough

import (
	"github.com/hashicorp/terraform/states/statecrypto/cryptoconfig"
	"testing"
)

func TestPassthroughWorks(t *testing.T) {
	cut, err := New(cryptoconfig.StateCryptoConfig{})
	if err != nil {
		t.Fatalf("got unexpected error during passthrough instantiate: %s", err.Error())
	}

	data := []byte(`{"some":"json","document":[{"with":"an"},{"array","inside"}]}`)

	dataPassthroughEncrypted, err := cut.Encrypt(data)
	if err != nil {
		t.Errorf("got unexpected error during passthrough encryption: %s", err.Error())
	}
	if !compareSlices(dataPassthroughEncrypted, data) {
		t.Error("passthrough encryption changed the data")
	}

	dataPassthroughDecrypted, err := cut.Decrypt(data)
	if err != nil {
		t.Errorf("got unexpected error during passthrough decryption: %s", err.Error())
	}
	if !compareSlices(dataPassthroughDecrypted, data) {
		t.Error("passthrough decryption changed the data")
	}
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
