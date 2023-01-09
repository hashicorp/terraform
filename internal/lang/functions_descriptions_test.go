package lang

import (
	"testing"

	"github.com/hashicorp/terraform/internal/lang/funcs"
)

func TestFunctionDescriptions(t *testing.T) {
	scope := &Scope{
		ConsoleMode: true,
	}
	// This will implicitly test the parameter description count since
	// WithNewDescriptions will panic if the number doesn't match.
	allFunctions := scope.Functions()

	if len(allFunctions) != len(funcs.DescriptionList) {
		t.Errorf("DescriptionList length expected: %d, got %d", len(allFunctions), len(funcs.DescriptionList))
	}

	for name := range allFunctions {
		_, ok := funcs.DescriptionList[name]
		if !ok {
			t.Errorf("missing DescriptionList entry for function %q", name)
		}
	}
}
