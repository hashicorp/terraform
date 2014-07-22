package config

import (
	"testing"
)

func TestExprParse(t *testing.T) {
	exprParse(&exprLex{input: `lookup(var.foo)`})
}
