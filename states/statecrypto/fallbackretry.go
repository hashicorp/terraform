package statecrypto

import (
	"github.com/hashicorp/terraform/states/statecrypto/implementations/passthrough"
	"log"
)

// FallbackRetryStateWrapper is a StateCryptoProvider that contains two other StateCryptoProviders,
// the first choice, and an optional fallback.
//
// encryption always uses the first choice.
//
// decryption first tries the first choice, if an error occurs and a fallback has been provided, the fallback
// is also tried, and a message is logged about this fact.
//
// exception: if the first choice is PassthroughStateWrapper and a fallback is configured,
// ONLY the fallback is tried for decryption. This is because PassthroughStateWrapper would have no way to determine
// if it got an unencrypted state or an encrypted state (any json could be valid state).
//
// Example use case: key rotation - first choice is encryption with the new key, fallback knows how to decrypt with the old key.
type FallbackRetryStateWrapper struct {
	firstChoice StateCryptoProvider
	fallback    StateCryptoProvider
}

func (f *FallbackRetryStateWrapper) Encrypt(data []byte) ([]byte, error) {
	return f.firstChoice.Encrypt(data)
}

func (f *FallbackRetryStateWrapper) Decrypt(data []byte) ([]byte, error) {
	_, firstChoiceIsPassthrough := f.firstChoice.(*passthrough.PassthroughStateWrapper)
	if firstChoiceIsPassthrough && f.fallback != nil {
		// try only the fallback, so encrypted state can be successfully decrypted using it
		// (note that all StateCryptoProviders are required to be able to pass through unencrypted state during decryption)
		candidate, err := f.fallback.Decrypt(data)
		if err != nil {
			log.Printf("[ERROR] failed to decrypt state with fallback configuration and main configuration is passthrough, bailing out")
			return []byte{}, err
		}
		log.Printf("[TRACE] successfully decrypted state using fallback configuration, input %d bytes, output %d bytes", len(data), len(candidate))
		return candidate, nil
	} else {
		candidate, err := f.firstChoice.Decrypt(data)
		if err != nil {
			if f.fallback != nil {
				log.Printf("[INFO] failed to decrypt state with main encryption configuration, trying fallback configuration")
				candidate2, err := f.fallback.Decrypt(data)
				if err != nil {
					log.Printf("[ERROR] failed to decrypt state with fallback configuration as well, bailing out")
					return []byte{}, err
				}
				log.Printf("[TRACE] successfully decrypted state using fallback configuration, input %d bytes, output %d bytes", len(data), len(candidate2))
				return candidate2, nil
			}
			log.Print("[TRACE] failed to decrypt state with first choice configuration and no fallback available")
			return []byte{}, err
		}
		log.Printf("[TRACE] successfully decrypted state using first choice configuration, input %d bytes, output %d bytes", len(data), len(candidate))
		return candidate, nil
	}
}

func fallbackRetryInstance(firstChoice StateCryptoProvider, fallback StateCryptoProvider) StateCryptoProvider {
	return &FallbackRetryStateWrapper{
		firstChoice: firstChoice,
		fallback:    fallback,
	}
}
