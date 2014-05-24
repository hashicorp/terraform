package digraph

// DepthFirstWalk performs a depth-first traversal of the nodes
// that can be reached from the initial input set. The callback is
// invoked for each visited node, and may return false to prevent
// vising any children of the current node
func DepthFirstWalk(node Node, cb func(n Node) bool) {
	frontier := []Node{node}
	seen := make(map[Node]struct{})
	for len(frontier) > 0 {
		// Pop the current node
		n := len(frontier)
		current := frontier[n-1]
		frontier = frontier[:n-1]

		// Check for potential cycle
		if _, ok := seen[current]; ok {
			continue
		}
		seen[current] = struct{}{}

		// Visit with the callback
		if !cb(current) {
			continue
		}

		// Add any new edges to visit, in reverse order
		edges := current.Edges()
		for i := len(edges) - 1; i >= 0; i-- {
			frontier = append(frontier, edges[i].Tail())
		}
	}
}

// FilterDegree returns only the nodes with the desired
// degree. This can be used with OutDegree or InDegree
func FilterDegree(degree int, degrees map[Node]int) []Node {
	var matching []Node
	for n, d := range degrees {
		if d == degree {
			matching = append(matching, n)
		}
	}
	return matching
}

// InDegree is used to compute the in-degree of nodes
func InDegree(nodes []Node) map[Node]int {
	degree := make(map[Node]int, len(nodes))
	for _, n := range nodes {
		if _, ok := degree[n]; !ok {
			degree[n] = 0
		}
		for _, e := range n.Edges() {
			degree[e.Tail()]++
		}
	}
	return degree
}

// OutDegree is used to compute the in-degree of nodes
func OutDegree(nodes []Node) map[Node]int {
	degree := make(map[Node]int, len(nodes))
	for _, n := range nodes {
		degree[n] = len(n.Edges())
	}
	return degree
}

// Sinks is used to get the nodes with out-degree of 0
func Sinks(nodes []Node) []Node {
	return FilterDegree(0, OutDegree(nodes))
}

// Sources is used to get the nodes with in-degree of 0
func Sources(nodes []Node) []Node {
	return FilterDegree(0, InDegree(nodes))
}

// Unreachable starts at a given start node, performs
// a DFS from there, and returns the set of unreachable nodes.
func Unreachable(start Node, nodes []Node) []Node {
	// DFS from the start ndoe
	frontier := []Node{start}
	seen := make(map[Node]struct{})
	for len(frontier) > 0 {
		// Pop the current node
		n := len(frontier)
		current := frontier[n-1]
		frontier = frontier[:n-1]

		// Check for potential cycle
		if _, ok := seen[current]; ok {
			continue
		}
		seen[current] = struct{}{}

		// Add any new edges to visit, in reverse order
		edges := current.Edges()
		for i := len(edges) - 1; i >= 0; i-- {
			frontier = append(frontier, edges[i].Tail())
		}
	}

	// Check for any unseen nodes
	var unseen []Node
	for _, node := range nodes {
		if _, ok := seen[node]; !ok {
			unseen = append(unseen, node)
		}
	}
	return unseen
}
