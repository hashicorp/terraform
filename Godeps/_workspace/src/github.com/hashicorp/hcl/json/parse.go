package json

import (
	"sync"

	"github.com/hashicorp/hcl/hcl"
	"github.com/hashicorp/terraform/helper/multierror"
)

// jsonErrors are the errors built up from parsing. These should not
// be accessed directly.
var jsonErrors []error
var jsonLock sync.Mutex
var jsonResult *hcl.Object

// Parse parses the given string and returns the result.
func Parse(v string) (*hcl.Object, error) {
	jsonLock.Lock()
	defer jsonLock.Unlock()
	jsonErrors = nil
	jsonResult = nil

	// Parse
	lex := &jsonLex{Input: v}
	jsonParse(lex)

	// If we have an error in the lexer itself, return it
	if lex.err != nil {
		return nil, lex.err
	}

	// Build up the errors
	var err error
	if len(jsonErrors) > 0 {
		err = &multierror.Error{Errors: jsonErrors}
		jsonResult = nil
	}

	return jsonResult, err
}
