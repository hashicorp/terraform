package terraform

import (
	"crypto/md5"
	"encoding/hex"
)

// PathCacheKey returns a cache key for a module path.
//
// TODO: test
func PathCacheKey(path []string) string {
	// There is probably a better way to do this, but this is working for now.
	// We just create an MD5 hash of all the MD5 hashes of all the path
	// elements. This gets us the property that it is unique per ordering.
	hash := md5.New()
	for _, p := range path {
		single := md5.Sum([]byte(p))
		if _, err := hash.Write(single[:]); err != nil {
			panic(err)
		}
	}

	return hex.EncodeToString(hash.Sum(nil))
}
