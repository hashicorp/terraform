package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/dag"
)

// GraphSemanticChecker is the interface that semantic checks across
// the entire Terraform graph implement.
//
// The graph should NOT be modified by the semantic checker.
type GraphSemanticChecker interface {
	Check(*dag.Graph) error
}

// UnorderedSemanticCheckRunner is an implementation of GraphSemanticChecker
// that runs a list of SemanticCheckers against the vertices of the graph
// in no specified order.
type UnorderedSemanticCheckRunner struct {
	Checks []SemanticChecker
}

func (sc *UnorderedSemanticCheckRunner) Check(g *dag.Graph) error {
	var err error
	for _, v := range g.Vertices() {
		for _, check := range sc.Checks {
			if e := check.Check(g, v); e != nil {
				err = multierror.Append(err, e)
			}
		}
	}

	return err
}

// SemanticChecker is the interface that semantic checks across the
// Terraform graph implement. Errors are accumulated. Even after an error
// is returned, child vertices in the graph will still be visited.
//
// The graph should NOT be modified by the semantic checker.
//
// The order in which vertices are visited is left unspecified, so the
// semantic checks should not rely on that.
type SemanticChecker interface {
	Check(*dag.Graph, dag.Vertex) error
}

// SemanticCheckModulesExist is an implementation of SemanticChecker that
// verifies that all the modules that are referenced in the graph exist.
type SemanticCheckModulesExist struct{}

// TODO: test
func (*SemanticCheckModulesExist) Check(g *dag.Graph, v dag.Vertex) error {
	mn, ok := v.(*GraphNodeConfigModule)
	if !ok {
		return nil
	}

	if mn.Tree == nil {
		return fmt.Errorf(
			"module '%s' not found", mn.Module.Name)
	}

	return nil
}

// smcUserVariables does all the semantic checks to verify that the
// variables given satisfy the configuration itself.
func smcUserVariables(c *config.Config, vs map[string]interface{}) []error {
	var errs []error

	cvs := make(map[string]*config.Variable)
	for _, v := range c.Variables {
		cvs[v.Name] = v
	}

	// Check that all required variables are present
	required := make(map[string]struct{})
	for _, v := range c.Variables {
		if v.Required() {
			required[v.Name] = struct{}{}
		}
	}
	for k, _ := range vs {
		delete(required, k)
	}
	if len(required) > 0 {
		for k, _ := range required {
			errs = append(errs, fmt.Errorf(
				"Required variable not set: %s", k))
		}
	}

	// Check that types match up
	for name, proposedValue := range vs {
		// Check for "map.key" fields. These stopped working with Terraform
		// 0.7 but we do this to surface a better error message informing
		// the user what happened.
		if idx := strings.Index(name, "."); idx > 0 {
			key := name[:idx]
			if _, ok := cvs[key]; ok {
				errs = append(errs, fmt.Errorf(
					"%s: Overriding map keys with the format `name.key` is no "+
						"longer allowed. You may still override keys by setting "+
						"`name = { key = value }`. The maps will be merged. This "+
						"behavior appeared in 0.7.0.", name))
				continue
			}
		}

		schema, ok := cvs[name]
		if !ok {
			continue
		}

		declaredType := schema.Type()

		switch declaredType {
		case config.VariableTypeString:
			switch proposedValue.(type) {
			case string:
				continue
			}
		case config.VariableTypeMap:
			switch v := proposedValue.(type) {
			case map[string]interface{}:
				continue
			case []map[string]interface{}:
				// if we have a list of 1 map, it will get coerced later as needed
				if len(v) == 1 {
					continue
				}
			}
		case config.VariableTypeList:
			switch proposedValue.(type) {
			case []interface{}:
				continue
			}
		}
		errs = append(errs, fmt.Errorf("variable %s should be type %s, got %s",
			name, declaredType.Printable(), hclTypeName(proposedValue)))
	}

	// TODO(mitchellh): variables that are unknown

	return errs
}
