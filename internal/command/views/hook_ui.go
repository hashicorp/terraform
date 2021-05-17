package views

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
)

const defaultPeriodicUiTimer = 10 * time.Second
const maxIdLen = 80

func NewUiHook(view *View) *UiHook {
	return &UiHook{
		view:            view,
		periodicUiTimer: defaultPeriodicUiTimer,
		resources:       make(map[string]uiResourceState),
	}
}

type UiHook struct {
	terraform.NilHook

	view     *View
	viewLock sync.Mutex

	periodicUiTimer time.Duration

	resources     map[string]uiResourceState
	resourcesLock sync.Mutex
}

var _ terraform.Hook = (*UiHook)(nil)

// uiResourceState tracks the state of a single resource
type uiResourceState struct {
	DispAddr       string
	IDKey, IDValue string
	Op             uiResourceOp
	Start          time.Time

	DoneCh chan struct{} // To be used for cancellation

	done chan struct{} // used to coordinate tests
}

// uiResourceOp is an enum for operations on a resource
type uiResourceOp byte

const (
	uiResourceUnknown uiResourceOp = iota
	uiResourceCreate
	uiResourceModify
	uiResourceDestroy
	uiResourceRead
)

func (h *UiHook) PreApply(addr addrs.AbsResourceInstance, gen states.Generation, action plans.Action, priorState, plannedNewState cty.Value) (terraform.HookAction, error) {
	dispAddr := addr.String()
	if gen != states.CurrentGen {
		dispAddr = fmt.Sprintf("%s (deposed object %s)", dispAddr, gen)
	}

	var operation string
	var op uiResourceOp
	idKey, idValue := format.ObjectValueIDOrName(priorState)
	switch action {
	case plans.Delete:
		operation = "Destroying..."
		op = uiResourceDestroy
	case plans.Create:
		operation = "Creating..."
		op = uiResourceCreate
	case plans.Update:
		operation = "Modifying..."
		op = uiResourceModify
	case plans.Read:
		operation = "Reading..."
		op = uiResourceRead
	default:
		// We don't expect any other actions in here, so anything else is a
		// bug in the caller but we'll ignore it in order to be robust.
		h.println(fmt.Sprintf("(Unknown action %s for %s)", action, dispAddr))
		return terraform.HookActionContinue, nil
	}

	var stateIdSuffix string
	if idKey != "" && idValue != "" {
		stateIdSuffix = fmt.Sprintf(" [%s=%s]", idKey, idValue)
	} else {
		// Make sure they are both empty so we can deal with this more
		// easily in the other hook methods.
		idKey = ""
		idValue = ""
	}

	h.println(fmt.Sprintf(
		h.view.colorize.Color("[reset][bold]%s: %s%s[reset]"),
		dispAddr,
		operation,
		stateIdSuffix,
	))

	key := addr.String()
	uiState := uiResourceState{
		DispAddr: key,
		IDKey:    idKey,
		IDValue:  idValue,
		Op:       op,
		Start:    time.Now().Round(time.Second),
		DoneCh:   make(chan struct{}),
		done:     make(chan struct{}),
	}

	h.resourcesLock.Lock()
	h.resources[key] = uiState
	h.resourcesLock.Unlock()

	// Start goroutine that shows progress
	go h.stillApplying(uiState)

	return terraform.HookActionContinue, nil
}

func (h *UiHook) stillApplying(state uiResourceState) {
	defer close(state.done)
	for {
		select {
		case <-state.DoneCh:
			return

		case <-time.After(h.periodicUiTimer):
			// Timer up, show status
		}

		var msg string
		switch state.Op {
		case uiResourceModify:
			msg = "Still modifying..."
		case uiResourceDestroy:
			msg = "Still destroying..."
		case uiResourceCreate:
			msg = "Still creating..."
		case uiResourceRead:
			msg = "Still reading..."
		case uiResourceUnknown:
			return
		}

		idSuffix := ""
		if state.IDKey != "" {
			idSuffix = fmt.Sprintf("%s=%s, ", state.IDKey, truncateId(state.IDValue, maxIdLen))
		}

		h.println(fmt.Sprintf(
			h.view.colorize.Color("[reset][bold]%s: %s [%s%s elapsed][reset]"),
			state.DispAddr,
			msg,
			idSuffix,
			time.Now().Round(time.Second).Sub(state.Start),
		))
	}
}

func (h *UiHook) PostApply(addr addrs.AbsResourceInstance, gen states.Generation, newState cty.Value, applyerr error) (terraform.HookAction, error) {
	id := addr.String()

	h.resourcesLock.Lock()
	state := h.resources[id]
	if state.DoneCh != nil {
		close(state.DoneCh)
	}

	delete(h.resources, id)
	h.resourcesLock.Unlock()

	var stateIdSuffix string
	if k, v := format.ObjectValueID(newState); k != "" && v != "" {
		stateIdSuffix = fmt.Sprintf(" [%s=%s]", k, v)
	}

	var msg string
	switch state.Op {
	case uiResourceModify:
		msg = "Modifications complete"
	case uiResourceDestroy:
		msg = "Destruction complete"
	case uiResourceCreate:
		msg = "Creation complete"
	case uiResourceRead:
		msg = "Read complete"
	case uiResourceUnknown:
		return terraform.HookActionContinue, nil
	}

	if applyerr != nil {
		// Errors are collected and printed in ApplyCommand, no need to duplicate
		return terraform.HookActionContinue, nil
	}

	addrStr := addr.String()
	if depKey, ok := gen.(states.DeposedKey); ok {
		addrStr = fmt.Sprintf("%s (deposed object %s)", addrStr, depKey)
	}

	colorized := fmt.Sprintf(
		h.view.colorize.Color("[reset][bold]%s: %s after %s%s"),
		addrStr, msg, time.Now().Round(time.Second).Sub(state.Start), stateIdSuffix)

	h.println(colorized)

	return terraform.HookActionContinue, nil
}

func (h *UiHook) PreProvisionInstanceStep(addr addrs.AbsResourceInstance, typeName string) (terraform.HookAction, error) {
	h.println(fmt.Sprintf(
		h.view.colorize.Color("[reset][bold]%s: Provisioning with '%s'...[reset]"),
		addr, typeName,
	))
	return terraform.HookActionContinue, nil
}

func (h *UiHook) ProvisionOutput(addr addrs.AbsResourceInstance, typeName string, msg string) {
	var buf bytes.Buffer

	prefix := fmt.Sprintf(
		h.view.colorize.Color("[reset][bold]%s (%s):[reset] "),
		addr, typeName,
	)
	s := bufio.NewScanner(strings.NewReader(msg))
	s.Split(scanLines)
	for s.Scan() {
		line := strings.TrimRightFunc(s.Text(), unicode.IsSpace)
		if line != "" {
			buf.WriteString(fmt.Sprintf("%s%s\n", prefix, line))
		}
	}

	h.println(strings.TrimSpace(buf.String()))
}

func (h *UiHook) PreRefresh(addr addrs.AbsResourceInstance, gen states.Generation, priorState cty.Value) (terraform.HookAction, error) {
	var stateIdSuffix string
	if k, v := format.ObjectValueID(priorState); k != "" && v != "" {
		stateIdSuffix = fmt.Sprintf(" [%s=%s]", k, v)
	}

	addrStr := addr.String()
	if depKey, ok := gen.(states.DeposedKey); ok {
		addrStr = fmt.Sprintf("%s (deposed object %s)", addrStr, depKey)
	}

	h.println(fmt.Sprintf(
		h.view.colorize.Color("[reset][bold]%s: Refreshing state...%s"),
		addrStr, stateIdSuffix))
	return terraform.HookActionContinue, nil
}

func (h *UiHook) PreImportState(addr addrs.AbsResourceInstance, importID string) (terraform.HookAction, error) {
	h.println(fmt.Sprintf(
		h.view.colorize.Color("[reset][bold]%s: Importing from ID %q..."),
		addr, importID,
	))
	return terraform.HookActionContinue, nil
}

func (h *UiHook) PostImportState(addr addrs.AbsResourceInstance, imported []providers.ImportedResource) (terraform.HookAction, error) {
	h.println(fmt.Sprintf(
		h.view.colorize.Color("[reset][bold][green]%s: Import prepared!"),
		addr,
	))
	for _, s := range imported {
		h.println(fmt.Sprintf(
			h.view.colorize.Color("[reset][green]  Prepared %s for import"),
			s.TypeName,
		))
	}

	return terraform.HookActionContinue, nil
}

// Wrap calls to the view so that concurrent calls do not interleave println.
func (h *UiHook) println(s string) {
	h.viewLock.Lock()
	defer h.viewLock.Unlock()
	h.view.streams.Println(s)
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
		// We have a full carriage-return-terminated line.
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

func truncateId(id string, maxLen int) string {
	// Note that the id may contain multibyte characters.
	// We need to truncate it to maxLen characters, not maxLen bytes.
	rid := []rune(id)
	totalLength := len(rid)
	if totalLength <= maxLen {
		return id
	}
	if maxLen < 5 {
		// We don't shorten to less than 5 chars
		// as that would be pointless with ... (3 chars)
		maxLen = 5
	}

	dots := []rune("...")
	partLen := maxLen / 2

	leftIdx := partLen - 1
	leftPart := rid[0:leftIdx]

	rightIdx := totalLength - partLen - 1

	overlap := maxLen - (partLen*2 + len(dots))
	if overlap < 0 {
		rightIdx -= overlap
	}

	rightPart := rid[rightIdx:]

	return string(leftPart) + string(dots) + string(rightPart)
}
