package passthrough

import "github.com/hashicorp/terraform/states/statecrypto/cryptoconfig"

// create a new passthrough state encryption wrapper (that does nothing)
func New(_ cryptoconfig.StateCryptoConfig) (*PassthroughStateWrapper, error) {
	return &PassthroughStateWrapper{}, nil
}
