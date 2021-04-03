package statecrypto

// StateCryptoProvider is the interface that must be implemented for a transparent remote state
// encryption layer. It is used to encrypt/decrypt the state payload before writing
// to or after reading from the remote state driver
//
// Note that the encrypted payload must still be valid json, because some remote state drivers
// expect valid json
type StateCryptoProvider interface {
	// implement this method to decrypt the encrypted payload
	//
	// encryptedPayload is a json document passed in as a []byte
	//
	// if you do not return an error, you MUST ensure you return a json document as
	// a []byte, because some statefile storage backends rely on this
	Decrypt(encryptedPayload []byte) ([]byte, error)

	// implement this method to encrypt the plaintext payload
	//
	// plaintextPayload is a json document passed in as a []byte
	//
	// if you do not return an error, you MUST ensure you return a json document as
	// a []byte, because some statefile storage backends rely on this
	Encrypt(plaintextPayload []byte) ([]byte, error)
}
