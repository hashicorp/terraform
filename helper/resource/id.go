package resource

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
)

const UniqueIdPrefix = `terraform-`

// Helper for a resource to generate a unique identifier
//
// This uses a simple RFC 4122 v4 UUID with some basic cosmetic filters
// applied (remove padding, downcase) to help distinguishing visually between
// identifiers.
func UniqueId() string {
	var uuid [16]byte
	rand.Read(uuid[:])
	return fmt.Sprintf("%s%s", UniqueIdPrefix,
		strings.ToLower(
			strings.Replace(
				base32.StdEncoding.EncodeToString(uuid[:]),
				"=", "", -1)))
}
