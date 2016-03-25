package command

import (
	"errors"

	"github.com/hashicorp/terraform/terraform"
)

// StateMeta is the meta struct that should be embedded in state subcommands.
type StateMeta struct{}

// filterInstance filters a single instance out of filter results.
func (c *StateMeta) filterInstance(rs []*terraform.StateFilterResult) (*terraform.StateFilterResult, error) {
	var result *terraform.StateFilterResult
	for _, r := range rs {
		if _, ok := r.Value.(*terraform.InstanceState); !ok {
			continue
		}

		if result != nil {
			return nil, errors.New(errStateMultiple)
		}

		result = r
	}

	return result, nil
}

const errStateMultiple = `Multiple instances found for the given pattern!

This command requires that the pattern match exactly one instance
of a resource. To view the matched instances, use "terraform state list".
Please modify the pattern to match only a single instance.`
