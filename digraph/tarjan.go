package digraph

// sccAcct is used ot pass around accounting information for
// the StronglyConnectedComponents algorithm
type sccAcct struct {
	ExcludeSingle bool
	NextIndex     int
	NodeIndex     map[Node]int
	Stack         []Node
	SCC           [][]Node
}

// visit assigns an index and pushes a node onto the stack
func (s *sccAcct) visit(n Node) int {
	idx := s.NextIndex
	s.NodeIndex[n] = idx
	s.NextIndex++
	s.push(n)
	return idx
}

// push adds a node to the stack
func (s *sccAcct) push(n Node) {
	s.Stack = append(s.Stack, n)
}

// pop removes a node from the stack
func (s *sccAcct) pop() Node {
	n := len(s.Stack)
	if n == 0 {
		return nil
	}
	node := s.Stack[n-1]
	s.Stack = s.Stack[:n-1]
	return node
}

// inStack checks if a node is in the stack
func (s *sccAcct) inStack(needle Node) bool {
	for _, n := range s.Stack {
		if n == needle {
			return true
		}
	}
	return false
}

// StronglyConnectedComponents implements Tarjan's algorithm to
// find all the strongly connected components in a graph. This can
// be used to detected any cycles in a graph, as well as which nodes
// partipate in those cycles. excludeSingle is used to exclude strongly
// connected components of size one.
func StronglyConnectedComponents(nodes []Node, excludeSingle bool) [][]Node {
	acct := sccAcct{
		ExcludeSingle: excludeSingle,
		NextIndex:     1,
		NodeIndex:     make(map[Node]int, len(nodes)),
	}
	for _, node := range nodes {
		// Recurse on any non-visited nodes
		if acct.NodeIndex[node] == 0 {
			stronglyConnected(&acct, node)
		}
	}
	return acct.SCC
}

func stronglyConnected(acct *sccAcct, node Node) int {
	// Initial node visit
	index := acct.visit(node)
	minIdx := index

	for _, edge := range node.Edges() {
		target := edge.Tail()
		targetIdx := acct.NodeIndex[target]

		// Recurse on successor if not yet visited
		if targetIdx == 0 {
			minIdx = min(minIdx, stronglyConnected(acct, target))

		} else if acct.inStack(target) {
			// Check if the node is in the stack
			minIdx = min(minIdx, targetIdx)
		}
	}

	// Pop the strongly connected components off the stack if
	// this is a root node
	if index == minIdx {
		var scc []Node
		for {
			n := acct.pop()
			scc = append(scc, n)
			if n == node {
				break
			}
		}
		if !(acct.ExcludeSingle && len(scc) == 1) {
			acct.SCC = append(acct.SCC, scc)
		}
	}

	return minIdx
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
