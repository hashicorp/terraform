// +build !windows

package env

import (
	"os"
)

func Getenv(s string) string {
	return os.Getenv(s)
}
