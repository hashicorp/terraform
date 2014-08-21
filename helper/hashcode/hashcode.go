package hashcode

import (
	"hash/crc32"
)

// String hashes a string to a unique hashcode.
func String(s string) int {
	return int(crc32.ChecksumIEEE([]byte(s)))
}
