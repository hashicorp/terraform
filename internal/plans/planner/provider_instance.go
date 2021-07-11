package planner

import (
	"github.com/hashicorp/terraform/internal/providers"
	opentracing "github.com/opentracing/opentracing-go"
)

// providerInstance is a wrapper around a real provider instance that
// allows planner to intercept "Close".
type providerInstance struct {
	providers.Interface
	onClose      func() error
	refCount     int
	lifetimeSpan opentracing.Span
}

func (p *providerInstance) Close() error {
	return p.onClose()
}
