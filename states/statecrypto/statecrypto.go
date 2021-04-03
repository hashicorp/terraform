package statecrypto

import (
	"github.com/hashicorp/terraform/states/statecrypto/cryptoconfig"
	"github.com/hashicorp/terraform/states/statecrypto/implementations/aes256state"
	"github.com/hashicorp/terraform/states/statecrypto/implementations/passthrough"
	"log"
)

func instanceFromConfig(config cryptoconfig.StateCryptoConfig, allowPassthrough bool) StateCryptoProvider {
	var implementation StateCryptoProvider
	var err error = nil

	switch config.Implementation {
	case "client-side/AES256-cfb/SHA256":
		implementation, err = aes256state.New(config)
	case "":
		if allowPassthrough {
			implementation, err = passthrough.New(config)
		} else {
			// valid case for fallback
			return nil
		}
	default:
		log.Fatalf("error configuring state file crypto: unsupported implementation '%s'", config.Implementation)
	}

	if err != nil {
		log.Fatalf("error configuring state file crypto: %v", err)
	}

	return implementation
}

func firstChoice() StateCryptoProvider {
	return instanceFromConfig(cryptoconfig.Configuration(), true)
}

func fallback() StateCryptoProvider {
	return instanceFromConfig(cryptoconfig.FallbackConfiguration(), false)
}

func StateCryptoWrapper() StateCryptoProvider {
	return &FallbackRetryStateWrapper{
		firstChoice: firstChoice(),
		fallback:    fallback(),
	}
}
