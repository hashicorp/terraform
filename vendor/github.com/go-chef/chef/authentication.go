package chef

import (
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"io"
	"math/big"
)

// GenerateSignature will generate a signature ( sign ) the given data
func GenerateSignature(priv *rsa.PrivateKey, data string) (enc []byte, err error) {
	sig, err := privateEncrypt(priv, []byte(data))
	if err != nil {
		return nil, err
	}

	return sig, nil
}

// privateEncrypt implements OpenSSL's RSA_private_encrypt function
func privateEncrypt(key *rsa.PrivateKey, data []byte) (enc []byte, err error) {
	k := (key.N.BitLen() + 7) / 8
	tLen := len(data)

	// rfc2313, section 8:
	// The length of the data D shall not be more than k-11 octets
	if tLen > k-11 {
		err = errors.New("Data too long")
		return
	}
	em := make([]byte, k)
	em[1] = 1
	for i := 2; i < k-tLen-1; i++ {
		em[i] = 0xff
	}
	copy(em[k-tLen:k], data)
	c := new(big.Int).SetBytes(em)
	if c.Cmp(key.N) > 0 {
		err = nil
		return
	}
	var m *big.Int
	var ir *big.Int
	if key.Precomputed.Dp == nil {
		m = new(big.Int).Exp(c, key.D, key.N)
	} else {
		// We have the precalculated values needed for the CRT.
		m = new(big.Int).Exp(c, key.Precomputed.Dp, key.Primes[0])
		m2 := new(big.Int).Exp(c, key.Precomputed.Dq, key.Primes[1])
		m.Sub(m, m2)
		if m.Sign() < 0 {
			m.Add(m, key.Primes[0])
		}
		m.Mul(m, key.Precomputed.Qinv)
		m.Mod(m, key.Primes[0])
		m.Mul(m, key.Primes[1])
		m.Add(m, m2)

		for i, values := range key.Precomputed.CRTValues {
			prime := key.Primes[2+i]
			m2.Exp(c, values.Exp, prime)
			m2.Sub(m2, m)
			m2.Mul(m2, values.Coeff)
			m2.Mod(m2, prime)
			if m2.Sign() < 0 {
				m2.Add(m2, prime)
			}
			m2.Mul(m2, values.R)
			m.Add(m, m2)
		}
	}

	if ir != nil {
		// Unblind.
		m.Mul(m, ir)
		m.Mod(m, key.N)
	}
	enc = m.Bytes()
	return
}

// HashStr returns the base64 encoded SHA1 sum of the toHash string
func HashStr(toHash string) string {
	h := sha1.New()
	io.WriteString(h, toHash)
	hashed := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return hashed
}

// Base64BlockEncode takes a byte slice and breaks it up into a
// slice of base64 encoded strings
func Base64BlockEncode(content []byte, limit int) []string {
	resultString := base64.StdEncoding.EncodeToString(content)
	var resultSlice []string

	index := 0

	var maxLengthPerSlice int

	// No limit
	if limit == 0 {
		maxLengthPerSlice = len(resultString)
	} else {
		maxLengthPerSlice = limit
	}

	// Iterate through the encoded string storing
	// a max of <limit> per slice item
	for i := 0; i < len(resultString)/maxLengthPerSlice; i++ {
		resultSlice = append(resultSlice, resultString[index:index+maxLengthPerSlice])
		index += maxLengthPerSlice
	}

	// Add remaining chunk to the end of the slice
	if len(resultString)%maxLengthPerSlice != 0 {
		resultSlice = append(resultSlice, resultString[index:])
	}

	return resultSlice
}
