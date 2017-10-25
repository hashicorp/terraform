package hclsyntax

import (
	"bytes"
	"fmt"
)

type navigation struct {
	root *Body
}

// Implementation of zcled.ContextString
func (n navigation) ContextString(offset int) string {
	// We will walk our top-level blocks until we find one that contains
	// the given offset, and then construct a representation of the header
	// of the block.

	var block *Block
	for _, candidate := range n.root.Blocks {
		if candidate.Range().ContainsOffset(offset) {
			block = candidate
			break
		}
	}

	if block == nil {
		return ""
	}

	if len(block.Labels) == 0 {
		// Easy case!
		return block.Type
	}

	buf := &bytes.Buffer{}
	buf.WriteString(block.Type)
	for _, label := range block.Labels {
		fmt.Fprintf(buf, " %q", label)
	}
	return buf.String()
}
