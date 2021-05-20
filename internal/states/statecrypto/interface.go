package statecrypto

// StateCryptoProvider is the interface that must be implemented for a transparent client side
// remote state encryption wrapper. It is used to encrypt/decrypt the state payload before writing
// to or after reading from the remote state backend.
//
// Note that the encrypted payload must still be valid json, because some remote state backends
// expect valid json.
//
// Also note that all implementations must gracefully handle unencrypted state being passed into Decrypt(),
// because this will inevitably happen when first encrypting previously unencrypted state.
// You should log a warning, though.
type StateCryptoProvider interface {
	// Decrypt the state if encrypted, otherwise pass through unmodified.
	//
	// encryptedPayload is a json document passed in as a []byte.
	//
	// if you do not return an error, you must ensure you return a json document as a []byte.
	Decrypt(encryptedPayload []byte) ([]byte, error)

	// Encrypt the plaintext state.
	//
	// plaintextPayload is a json document passed in as a []byte.
	//
	// if you do not return an error, you must ensure you return a json document as
	// a []byte, because some remote state storage backends rely on this.
	Encrypt(plaintextPayload []byte) ([]byte, error)
}
