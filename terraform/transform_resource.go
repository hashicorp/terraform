package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/config"
)

type graphNodeExpandedResource struct {
	Index    int
	Resource *config.Resource
	Path     []string
}

func (n *graphNodeExpandedResource) Name() string {
	if n.Index == -1 {
		return n.Resource.Id()
	}

	return fmt.Sprintf("%s #%d", n.Resource.Id(), n.Index)
}

// GraphNodeDependent impl.
func (n *graphNodeExpandedResource) DependentOn() []string {
	configNode := &GraphNodeConfigResource{Resource: n.Resource}
	result := configNode.DependentOn()

	// Walk the variables to find any count-specific variables we depend on.
	configNode.VarWalk(func(v config.InterpolatedVariable) {
		rv, ok := v.(*config.ResourceVariable)
		if !ok {
			return
		}

		// We only want ourselves
		if rv.ResourceId() != n.Resource.Id() {
			return
		}

		// If this isn't a multi-access (which shouldn't be allowed but
		// is verified elsewhere), then we depend on the specific count
		// of this resource, ignoring ourself (which again should be
		// validated elsewhere).
		if rv.Index > -1 {
			id := fmt.Sprintf("%s.%d", rv.ResourceId(), rv.Index)
			if id != n.stateId() && id != n.stateId()+".0" {
				result = append(result, id)
			}
		}
	})

	return result
}

func (n *graphNodeExpandedResource) StateDependencies() []string {
	depsRaw := n.DependentOn()
	deps := make([]string, 0, len(depsRaw))
	for _, d := range depsRaw {
		// Ignore any variable dependencies
		if strings.HasPrefix(d, "var.") {
			continue
		}

		// This is sad. The dependencies are currently in the format of
		// "module.foo.bar" (the full field). This strips the field off.
		if strings.HasPrefix(d, "module.") {
			parts := strings.SplitN(d, ".", 3)
			d = strings.Join(parts[0:2], ".")
		}
		deps = append(deps, d)
	}

	return deps
}

// stateId is the name used for the state key
func (n *graphNodeExpandedResource) stateId() string {
	if n.Index == -1 {
		return n.Resource.Id()
	}

	return fmt.Sprintf("%s.%d", n.Resource.Id(), n.Index)
}
