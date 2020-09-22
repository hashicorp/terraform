package macosbase

import (
	"github.com/apparentlymart/go-userdirs/internal/unix"
)

func home() string {
	return unix.Home()
}
