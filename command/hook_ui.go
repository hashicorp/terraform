package command

import (
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

type UiHook struct {
	terraform.NilHook

	Ui cli.Ui

	once sync.Once
	ui   cli.Ui
}

func (h *UiHook) PreDiff(
	id string, s *terraform.ResourceState) (terraform.HookAction, error) {
	h.once.Do(h.init)

	h.ui.Output(fmt.Sprintf("%s: Calculating diff", id))
	return terraform.HookActionContinue, nil
}

func (h *UiHook) PreRefresh(
	id string, s *terraform.ResourceState) (terraform.HookAction, error) {
	h.once.Do(h.init)

	h.ui.Output(fmt.Sprintf("%s: Refreshing state (ID: %s)", id, s.ID))
	return terraform.HookActionContinue, nil
}

func (h *UiHook) init() {
	// Wrap the ui so that it is safe for concurrency regardless of the
	// underlying reader/writer that is in place.
	h.ui = &cli.ConcurrentUi{Ui: h.Ui}
}
