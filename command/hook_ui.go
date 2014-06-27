package command

import (
	"fmt"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

type UiHook struct {
	terraform.NilHook

	Ui cli.Ui
}

func (h *UiHook) PreRefresh(
	id string, s *terraform.ResourceState) (terraform.HookAction, error) {
	h.Ui.Output(fmt.Sprintf("Refreshing state for %s (ID: %s)", id, s.ID))
	return terraform.HookActionContinue, nil
}
