package zclwrite

import (
	"github.com/zclconf/go-zcl/zcl/zclsyntax"
)

type nativeNodeSorter struct {
	Nodes []zclsyntax.Node
}

func (s nativeNodeSorter) Len() int {
	return len(s.Nodes)
}

func (s nativeNodeSorter) Less(i, j int) bool {
	rangeI := s.Nodes[i].Range()
	rangeJ := s.Nodes[j].Range()
	return rangeI.Start.Byte < rangeJ.Start.Byte
}

func (s nativeNodeSorter) Swap(i, j int) {
	s.Nodes[i], s.Nodes[j] = s.Nodes[j], s.Nodes[i]
}
