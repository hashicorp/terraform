package aes256state

import "github.com/hashicorp/terraform/internal/states/statecrypto/cryptoconfig"

// New creates a new client-side/AES256-cfb/SHA256 state encryption wrapper.
func New(configuration cryptoconfig.StateCryptoConfig) (*AES256StateWrapper, error) {
	instance := &AES256StateWrapper{}
	err := instance.parseKeyFromConfiguration(configuration)
	return instance, err
}
