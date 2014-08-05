package config

import (
	"testing"
)

func TestHCLConfigurableConfigurable(t *testing.T) {
	var _ configurable = new(hclConfigurable)
}
