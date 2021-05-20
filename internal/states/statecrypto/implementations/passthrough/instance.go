package passthrough

import "github.com/hashicorp/terraform/internal/states/statecrypto/cryptoconfig"

func New(_ cryptoconfig.StateCryptoConfig) (*PassthroughStateWrapper, error) {
	return &PassthroughStateWrapper{}, nil
}
