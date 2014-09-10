package hcl

import (
	"sync"

	"github.com/hashicorp/terraform/helper/multierror"
)

// hclErrors are the errors built up from parsing. These should not
// be accessed directly.
var hclErrors []error
var hclLock sync.Mutex
var hclResult *Object

// Parse parses the given string and returns the result.
func Parse(v string) (*Object, error) {
	hclLock.Lock()
	defer hclLock.Unlock()
	hclErrors = nil
	hclResult = nil

	// Parse
	lex := &hclLex{Input: v}
	hclParse(lex)

	// If we have an error in the lexer itself, return it
	if lex.err != nil {
		return nil, lex.err
	}

	// Build up the errors
	var err error
	if len(hclErrors) > 0 {
		err = &multierror.Error{Errors: hclErrors}
		hclResult = nil
	}

	return hclResult, err
}
