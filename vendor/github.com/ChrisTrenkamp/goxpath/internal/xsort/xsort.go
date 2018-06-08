package xsort

import (
	"sort"

	"github.com/ChrisTrenkamp/goxpath/tree"
)

type nodeSort []tree.Node

func (ns nodeSort) Len() int      { return len(ns) }
func (ns nodeSort) Swap(i, j int) { ns[i], ns[j] = ns[j], ns[i] }
func (ns nodeSort) Less(i, j int) bool {
	return ns[i].Pos() < ns[j].Pos()
}

//SortNodes sorts the array by the node document order
func SortNodes(res []tree.Node) {
	sort.Sort(nodeSort(res))
}
