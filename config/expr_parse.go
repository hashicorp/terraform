package config

import (
	"sync"

	"github.com/hashicorp/terraform/helper/multierror"
)

// exprErrors are the errors built up from parsing. These should not
// be accessed directly.
var exprErrors []error
var exprLock sync.Mutex
var exprResult Interpolation

// ExprParse parses the given expression and returns an executable
// Interpolation.
func ExprParse(v string) (Interpolation, error) {
	exprLock.Lock()
	defer exprLock.Unlock()
	exprErrors = nil
	exprResult = nil

	// Parse
	lex := &exprLex{input: v}
	exprParse(lex)

	// Build up the errors
	var err error
	if lex.Err != nil {
		err = multierror.ErrorAppend(err, lex.Err)
	}
	if len(exprErrors) > 0 {
		err = multierror.ErrorAppend(err, exprErrors...)
	}
	if err != nil {
		exprResult = nil
	}

	return exprResult, err
}
