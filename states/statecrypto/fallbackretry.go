package statecrypto

import "log"

type FallbackRetryStateWrapper struct {
	firstChoice StateCryptoProvider
	fallback    StateCryptoProvider
}

func (f *FallbackRetryStateWrapper) Encrypt(data []byte) ([]byte, error) {
	return f.firstChoice.Encrypt(data)
}

func (f *FallbackRetryStateWrapper) Decrypt(data []byte) ([]byte, error) {
	candidate, err := f.firstChoice.Decrypt(data)
	if err != nil {
		if f.fallback != nil {
			log.Printf("failed to decrypt state with main encryption configuration, trying fallback configuration")
			candidate2, err := f.fallback.Decrypt(data)
			if err != nil {
				log.Printf("failed to decrypt state with fallback configuration as well, bailing out")
				return []byte{}, err
			}
			return candidate2, nil
		}
		return []byte{}, err
	}
	return candidate, nil
}
