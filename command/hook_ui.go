package command

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

type UiHook struct {
	terraform.NilHook

	Colorize *colorstring.Colorize
	Ui       cli.Ui

	once sync.Once
	ui   cli.Ui
}

func (h *UiHook) PreApply(
	id string,
	s *terraform.ResourceState,
	d *terraform.ResourceDiff) (terraform.HookAction, error) {
	h.once.Do(h.init)

	operation := "Modifying..."
	if d.Destroy {
		operation = "Destroying..."
	} else if s.ID == "" {
		operation = "Creating..."
	}

	attrBuf := new(bytes.Buffer)

	// Get all the attributes that are changing, and sort them. Also
	// determine the longest key so that we can align them all.
	keyLen := 0
	keys := make([]string, 0, len(d.Attributes))
	for key, _ := range d.Attributes {
		// Skip the ID since we do that specially
		if key == "id" {
			continue
		}

		keys = append(keys, key)
		if len(key) > keyLen {
			keyLen = len(key)
		}
	}
	sort.Strings(keys)

	// Go through and output each attribute
	for _, attrK := range keys {
		attrDiff := d.Attributes[attrK]

		v := attrDiff.New
		if attrDiff.NewComputed {
			v = "<computed>"
		}

		attrBuf.WriteString(fmt.Sprintf(
			"  %s:%s %#v => %#v\n",
			attrK,
			strings.Repeat(" ", keyLen-len(attrK)),
			attrDiff.Old,
			v))
	}

	h.ui.Output(h.Colorize.Color(fmt.Sprintf(
		"[reset][bold]%s: %s[reset_bold]\n  %s",
		id,
		operation,
		strings.TrimSpace(attrBuf.String()))))

	return terraform.HookActionContinue, nil
}

func (h *UiHook) PreDiff(
	id string, s *terraform.ResourceState) (terraform.HookAction, error) {
	return terraform.HookActionContinue, nil
}

func (h *UiHook) PreRefresh(
	id string, s *terraform.ResourceState) (terraform.HookAction, error) {
	h.once.Do(h.init)

	h.ui.Output(h.Colorize.Color(fmt.Sprintf(
		"[reset][bold]%s: Refreshing (ID: %s)",
		id, s.ID)))
	return terraform.HookActionContinue, nil
}

func (h *UiHook) init() {
	if h.Colorize == nil {
		panic("colorize not given")
	}

	// Wrap the ui so that it is safe for concurrency regardless of the
	// underlying reader/writer that is in place.
	h.ui = &cli.ConcurrentUi{Ui: h.Ui}
}
