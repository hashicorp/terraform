package configs

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestWarnForDeprecatedInterpolationsInExpr(t *testing.T) {
	tests := []struct {
		Expr       string
		WantSubstr string
	}{
		{
			`"${foo}"`,
			"leaving just the inner expression",
		},
		{
			`{"${foo}" = 1}`,
			// Special message for object key expressions, because just
			// removing the interpolation markers would change the meaning
			// in that context.
			"opening and closing parentheses respectively",
		},
		{
			`{upper("${foo}") = 1}`,
			// But no special message if the template is just descended from an
			// object key, because the special interpretation applies only to
			// a naked reference in te object key position.
			"leaving just the inner expression",
		},
	}

	for _, test := range tests {
		t.Run(test.Expr, func(t *testing.T) {
			expr, diags := hclsyntax.ParseExpression([]byte(test.Expr), "", hcl.InitialPos)
			if diags.HasErrors() {
				t.Fatalf("parse error: %s", diags.Error())
			}

			diags = warnForDeprecatedInterpolationsInExpr(expr)
			if !diagWarningsContainSubstring(diags, test.WantSubstr) {
				t.Errorf("wrong warning message\nwant detail substring: %s\ngot: %s", test.WantSubstr, diags.Error())
			}
		})
	}
}

func diagWarningsContainSubstring(diags hcl.Diagnostics, want string) bool {
	for _, diag := range diags {
		if diag.Severity != hcl.DiagWarning {
			continue
		}
		if strings.Contains(diag.Detail, want) {
			return true
		}
	}
	return false
}
