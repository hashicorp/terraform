package backend

import (
	"testing"
)

func TestNil_impl(t *testing.T) {
	var _ Backend = new(Nil)
}
