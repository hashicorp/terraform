package stackeval

import "github.com/hashicorp/terraform/internal/tfdiags"

type withDiagnostics[T any] struct {
	Result      T
	Diagnostics tfdiags.Diagnostics
}
