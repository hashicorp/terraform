package statecrypto

import (
	"github.com/hashicorp/terraform/states/statecrypto/cryptoconfig"
	"github.com/hashicorp/terraform/states/statecrypto/implementations/aes256state"
	"github.com/hashicorp/terraform/states/statecrypto/implementations/passthrough"
	"log"
)

func StateCryptoWrapper() StateCryptoProvider {
	return fallbackRetryInstance(firstChoice(), fallback())
}

func firstChoice() StateCryptoProvider {
	return instanceFromConfig(cryptoconfig.Configuration(), true)
}

func fallback() StateCryptoProvider {
	return instanceFromConfig(cryptoconfig.FallbackConfiguration(), false)
}

var logFatalf = log.Fatalf

func instanceFromConfig(config cryptoconfig.StateCryptoConfig, allowPassthrough bool) StateCryptoProvider {
	var implementation StateCryptoProvider
	var err error = nil

	switch config.Implementation {
	case cryptoconfig.ClientSide_Aes256cfb_Sha256:
		implementation, err = aes256state.New(config)
	// add additional implementations here
	case "":
		if allowPassthrough {
			implementation, err = passthrough.New(config)
		} else {
			// valid case for fallback - means no fallback available
			return nil
		}
	default:
		logFatalf("[ERROR] failed to configure remote state encryption: unsupported implementation '%s'", config.Implementation)
	}

	if err != nil {
		logFatalf("[ERROR] failed to configure remote state encryption: %s", err.Error())
	}

	return implementation
}
