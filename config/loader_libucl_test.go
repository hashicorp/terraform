package config

import (
	"io"
	"testing"
)

func TestLibuclConfigurableCloser(t *testing.T) {
	var _ io.Closer = new(libuclConfigurable)
}

func TestLibuclConfigurableConfigurable(t *testing.T) {
	var _ configurable = new(libuclConfigurable)
}
