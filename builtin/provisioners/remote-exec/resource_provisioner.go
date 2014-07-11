package remoteexec

import (
	"fmt"

	"github.com/hashicorp/terraform/terraform"
)

type ResourceProvisioner struct{}

func (p *ResourceProvisioner) Apply(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig) (*terraform.ResourceState, error) {
	panic("not implemented")
	return s, nil
}

func (p *ResourceProvisioner) Validate(c *terraform.ResourceConfig) (ws []string, es []error) {
	var hasCommand, hasInline bool
	for name := range c.Raw {
		switch name {
		case "command":
			hasCommand = true
		case "inline":
			hasInline = true
		default:
			es = append(es, fmt.Errorf("Unknown configuration '%s'", name))
		}
	}
	if hasInline && hasCommand {
		es = append(es, fmt.Errorf("Cannot provide both 'command' and 'inline' to remote-exec"))
	} else if !hasInline && !hasCommand {
		es = append(es, fmt.Errorf("Must provide 'command' or 'inline' to remote-exec"))
	}
	return
}
