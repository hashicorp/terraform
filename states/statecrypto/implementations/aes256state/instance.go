package aes256state

import "github.com/hashicorp/terraform/states/statecrypto/statecryptoif"

// New creates a new AES256 state encryption wrapper.
func New(configuration []string) (statecryptoif.StateCrypto, error) {
	instance := &AES256StateWrapper{}
	err := instance.parseKeysFromConfiguration(configuration)
	return instance, err
}
