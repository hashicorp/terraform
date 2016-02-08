package native

import (
	"crypto/sha1"
	"math"
)

// Borrowed from GoMySQL
// SHA1(SHA1(SHA1(password)), scramble) XOR SHA1(password)
func encryptedPasswd(password string, scramble []byte) (out []byte) {
	if len(password) == 0 {
		return
	}
	// stage1_hash = SHA1(password)
	// SHA1 encode
	crypt := sha1.New()
	crypt.Write([]byte(password))
	stg1Hash := crypt.Sum(nil)
	// token = SHA1(SHA1(stage1_hash), scramble) XOR stage1_hash
	// SHA1 encode again
	crypt.Reset()
	crypt.Write(stg1Hash)
	stg2Hash := crypt.Sum(nil)
	// SHA1 2nd hash and scramble
	crypt.Reset()
	crypt.Write(scramble)
	crypt.Write(stg2Hash)
	stg3Hash := crypt.Sum(nil)
	// XOR with first hash
	out = make([]byte, len(scramble))
	for ii := range scramble {
		out[ii] = stg3Hash[ii] ^ stg1Hash[ii]
	}
	return
}

// Old password handling based on translating to Go some functions from
// libmysql

// The main idea is that no password are sent between client & server on
// connection and that no password are saved in mysql in a decodable form.
//
// On connection a random string is generated and sent to the client.
// The client generates a new string with a random generator inited with
// the hash values from the password and the sent string.
// This 'check' string is sent to the server where it is compared with
// a string generated from the stored hash_value of the password and the
// random string.

// libmysql/my_rnd.c
type myRnd struct {
	seed1, seed2 uint32
}

const myRndMaxVal = 0x3FFFFFFF

func newMyRnd(seed1, seed2 uint32) *myRnd {
	r := new(myRnd)
	r.seed1 = seed1 % myRndMaxVal
	r.seed2 = seed2 % myRndMaxVal
	return r
}

func (r *myRnd) Float64() float64 {
	r.seed1 = (r.seed1*3 + r.seed2) % myRndMaxVal
	r.seed2 = (r.seed1 + r.seed2 + 33) % myRndMaxVal
	return float64(r.seed1) / myRndMaxVal
}

// libmysql/password.c
func pwHash(password []byte) (result [2]uint32) {
	var nr, add, nr2, tmp uint32
	nr, add, nr2 = 1345345333, 7, 0x12345671

	for _, c := range password {
		if c == ' ' || c == '\t' {
			continue // skip space in password
		}

		tmp = uint32(c)
		nr ^= (((nr & 63) + add) * tmp) + (nr << 8)
		nr2 += (nr2 << 8) ^ nr
		add += tmp
	}

	result[0] = nr & ((1 << 31) - 1) // Don't use sign bit (str2int)
	result[1] = nr2 & ((1 << 31) - 1)
	return
}

func encryptedOldPassword(password string, scramble []byte) []byte {
	if len(password) == 0 {
		return nil
	}
	scramble = scramble[:8]
	hashPw := pwHash([]byte(password))
	hashSc := pwHash(scramble)
	r := newMyRnd(hashPw[0]^hashSc[0], hashPw[1]^hashSc[1])
	var out [8]byte
	for i := range out {
		out[i] = byte(math.Floor(r.Float64()*31) + 64)
	}
	extra := byte(math.Floor(r.Float64() * 31))
	for i := range out {
		out[i] ^= extra
	}
	return out[:]
}
