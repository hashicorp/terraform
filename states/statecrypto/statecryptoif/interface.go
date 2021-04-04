package statecryptoif

// StateCrypto is the interface that must be implemented for a remote state
// encryption layer. It is used to encrypt/decrypt the state payload before writing
// to or after reading from the remote state driver
//
// Note that the encrypted payload must still be valid json, because some remote state drivers
// expect valid json
type StateCrypto interface {
	Decrypt(encryptedPayload []byte) ([]byte, error)
	Encrypt(plaintextPayload []byte) ([]byte, error)
}
