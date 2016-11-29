package command

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

const periodicUiTimer = 10 * time.Second

type UiHook struct {
	terraform.NilHook

	Colorize *colorstring.Colorize
	Ui       cli.Ui

	l         sync.Mutex
	once      sync.Once
	resources map[string]uiResourceState
	ui        cli.Ui
}

// uiResourceState tracks the state of a single resource
type uiResourceState struct {
	Op    uiResourceOp
	Start time.Time
}

// uiResourceOp is an enum for operations on a resource
type uiResourceOp byte

const (
	uiResourceUnknown uiResourceOp = iota
	uiResourceCreate
	uiResourceModify
	uiResourceDestroy
)

func (h *UiHook) PreApply(
	n *terraform.InstanceInfo,
	s *terraform.InstanceState,
	d *terraform.InstanceDiff) (terraform.HookAction, error) {
	h.once.Do(h.init)

	id := n.HumanId()

	op := uiResourceModify
	if d.Destroy {
		op = uiResourceDestroy
	} else if s.ID == "" {
		op = uiResourceCreate
	}

	h.l.Lock()
	h.resources[id] = uiResourceState{
		Op:    op,
		Start: time.Now().Round(time.Second),
	}
	h.l.Unlock()

	var operation string
	switch op {
	case uiResourceModify:
		operation = "Modifying..."
	case uiResourceDestroy:
		operation = "Destroying..."
	case uiResourceCreate:
		operation = "Creating..."
	case uiResourceUnknown:
		return terraform.HookActionContinue, nil
	}

	attrBuf := new(bytes.Buffer)

	// Get all the attributes that are changing, and sort them. Also
	// determine the longest key so that we can align them all.
	keyLen := 0

	dAttrs := d.CopyAttributes()
	keys := make([]string, 0, len(dAttrs))
	for key, _ := range dAttrs {
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
		attrDiff, _ := d.GetAttribute(attrK)

		v := attrDiff.New
		u := attrDiff.Old
		if attrDiff.NewComputed {
			v = "<computed>"
		}

		if attrDiff.Sensitive {
			u = "<sensitive>"
			v = "<sensitive>"
		}

		attrBuf.WriteString(fmt.Sprintf(
			"  %s:%s %#v => %#v\n",
			attrK,
			strings.Repeat(" ", keyLen-len(attrK)),
			u,
			v))
	}

	attrString := strings.TrimSpace(attrBuf.String())
	if attrString != "" {
		attrString = "\n  " + attrString
	}

	h.ui.Output(h.Colorize.Color(fmt.Sprintf(
		"[reset][bold]%s: %s[reset]%s",
		id,
		operation,
		attrString)))

	// Set a timer to show an operation is still happening
	time.AfterFunc(periodicUiTimer, func() { h.stillApplying(id) })

	return terraform.HookActionContinue, nil
}

func (h *UiHook) stillApplying(id string) {
	// Grab the operation. We defer the lock here to avoid the "still..."
	// message showing up after a completion message.
	h.l.Lock()
	defer h.l.Unlock()
	state, ok := h.resources[id]

	// If the resource is out of the map it means we're done with it
	if !ok {
		return
	}

	var msg string
	switch state.Op {
	case uiResourceModify:
		msg = "Still modifying..."
	case uiResourceDestroy:
		msg = "Still destroying..."
	case uiResourceCreate:
		msg = "Still creating..."
	case uiResourceUnknown:
		return
	}

	h.ui.Output(h.Colorize.Color(fmt.Sprintf(
		"[reset][bold]%s: %s (%s elapsed)[reset]",
		id,
		msg,
		time.Now().Round(time.Second).Sub(state.Start),
	)))

	// Reschedule
	time.AfterFunc(periodicUiTimer, func() { h.stillApplying(id) })
}

func (h *UiHook) PostApply(
	n *terraform.InstanceInfo,
	s *terraform.InstanceState,
	applyerr error) (terraform.HookAction, error) {
	id := n.HumanId()

	h.l.Lock()
	state := h.resources[id]
	delete(h.resources, id)
	h.l.Unlock()

	var msg string
	switch state.Op {
	case uiResourceModify:
		msg = "Modifications complete"
	case uiResourceDestroy:
		msg = "Destruction complete"
	case uiResourceCreate:
		msg = "Creation complete"
	case uiResourceUnknown:
		return terraform.HookActionContinue, nil
	}

	if applyerr != nil {
		// Errors are collected and printed in ApplyCommand, no need to duplicate
		return terraform.HookActionContinue, nil
	}

	h.ui.Output(h.Colorize.Color(fmt.Sprintf(
		"[reset][bold]%s: %s[reset]",
		id, msg)))

	return terraform.HookActionContinue, nil
}

func (h *UiHook) PreDiff(
	n *terraform.InstanceInfo,
	s *terraform.InstanceState) (terraform.HookAction, error) {
	return terraform.HookActionContinue, nil
}

func (h *UiHook) PreProvision(
	n *terraform.InstanceInfo,
	provId string) (terraform.HookAction, error) {
	id := n.HumanId()
	h.ui.Output(h.Colorize.Color(fmt.Sprintf(
		"[reset][bold]%s: Provisioning with '%s'...[reset]",
		id, provId)))
	return terraform.HookActionContinue, nil
}

func (h *UiHook) ProvisionOutput(
	n *terraform.InstanceInfo,
	provId string,
	msg string) {
	id := n.HumanId()
	var buf bytes.Buffer
	buf.WriteString(h.Colorize.Color("[reset]"))

	prefix := fmt.Sprintf("%s (%s): ", id, provId)
	s := bufio.NewScanner(strings.NewReader(msg))
	s.Split(scanLines)
	for s.Scan() {
		line := strings.TrimRightFunc(s.Text(), unicode.IsSpace)
		if line != "" {
			buf.WriteString(fmt.Sprintf("%s%s\n", prefix, line))
		}
	}

	h.ui.Output(strings.TrimSpace(buf.String()))
}

func (h *UiHook) PreRefresh(
	n *terraform.InstanceInfo,
	s *terraform.InstanceState) (terraform.HookAction, error) {
	h.once.Do(h.init)

	id := n.HumanId()

	var stateIdSuffix string
	// Data resources refresh before they have ids, whereas managed
	// resources are only refreshed when they have ids.
	if s.ID != "" {
		stateIdSuffix = fmt.Sprintf(" (ID: %s)", s.ID)
	}

	h.ui.Output(h.Colorize.Color(fmt.Sprintf(
		"[reset][bold]%s: Refreshing state...%s",
		id, stateIdSuffix)))
	return terraform.HookActionContinue, nil
}

func (h *UiHook) PreImportState(
	n *terraform.InstanceInfo,
	id string) (terraform.HookAction, error) {
	h.once.Do(h.init)

	h.ui.Output(h.Colorize.Color(fmt.Sprintf(
		"[reset][bold]%s: Importing from ID %q...",
		n.HumanId(), id)))
	return terraform.HookActionContinue, nil
}

func (h *UiHook) PostImportState(
	n *terraform.InstanceInfo,
	s []*terraform.InstanceState) (terraform.HookAction, error) {
	h.once.Do(h.init)

	id := n.HumanId()
	h.ui.Output(h.Colorize.Color(fmt.Sprintf(
		"[reset][bold][green]%s: Import complete!", id)))
	for _, s := range s {
		h.ui.Output(h.Colorize.Color(fmt.Sprintf(
			"[reset][green]  Imported %s (ID: %s)",
			s.Ephemeral.Type, s.ID)))
	}

	return terraform.HookActionContinue, nil
}

func (h *UiHook) init() {
	if h.Colorize == nil {
		panic("colorize not given")
	}

	h.resources = make(map[string]uiResourceState)

	// Wrap the ui so that it is safe for concurrency regardless of the
	// underlying reader/writer that is in place.
	h.ui = &cli.ConcurrentUi{Ui: h.Ui}
}

// scanLines is basically copied from the Go standard library except
// we've modified it to also fine `\r`.
func scanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, dropCR(data[0:i]), nil
	}
	if i := bytes.IndexByte(data, '\r'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, dropCR(data[0:i]), nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), dropCR(data), nil
	}
	// Request more data.
	return 0, nil, nil
}

// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}
