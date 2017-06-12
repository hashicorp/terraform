package mysql

import (
	"crypto/sha256"
	"fmt"
)

func hashSum(contents interface{}) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(contents.(string))))
}
