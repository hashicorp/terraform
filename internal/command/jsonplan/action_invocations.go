package jsonplan

import (
	"github.com/hashicorp/terraform/internal/plans"
)

type ActionInvocation struct {
	// Address is the absolute action address
	Address string `json:"address,omitempty"`

	// ProviderName allows the property "type" to be interpreted unambiguously
	// in the unusual situation where a provider offers a type whose
	// name does not start with its own name, such as the "googlebeta" provider
	// offering "google_compute_instance".
	ProviderName string `json:"provider_name,omitempty"`
}

func MarshalActionInvocations(actions []*plans.ActionInvocationInstanceSrc) ([]ActionInvocation, error) {
	ret := make([]ActionInvocation, 0, len(actions))

	for _, action := range actions {
		ai := ActionInvocation{
			Address:      action.Addr.String(),
			ProviderName: action.ProviderAddr.String(),
		}

		ret = append(ret, ai)
	}

	return ret, nil
}
