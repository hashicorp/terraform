package dag

// StronglyConnected returns the list of strongly connected components
// within the Graph g. This information is primarily used by this package
// for cycle detection, but strongly connected components have widespread
// use.
func StronglyConnected(g *Graph) [][]Vertex {
	vs := g.Vertices()
	data := tarjanData{
		index:    make(map[interface{}]int),
		stack:    make([]*tarjanVertex, 0, len(vs)),
		vertices: make([]*tarjanVertex, 0, len(vs)),
	}

	for _, v := range vs {
		if _, ok := data.index[v]; !ok {
			strongConnect(g, v, &data)
		}
	}

	return data.result
}

type tarjanData struct {
	index    map[interface{}]int
	result   [][]Vertex
	stack    []*tarjanVertex
	vertices []*tarjanVertex
}

type tarjanVertex struct {
	V       Vertex
	Lowlink int
	Index   int
	Stack   bool
}

func strongConnect(g *Graph, v Vertex, data *tarjanData) *tarjanVertex {
	index := len(data.index)
	data.index[v] = index
	tv := &tarjanVertex{V: v, Lowlink: index, Index: index, Stack: true}
	data.stack = append(data.stack, tv)
	data.vertices = append(data.vertices, tv)

	for _, raw := range g.downEdges[v].List() {
		target := raw.(Vertex)

		if idx, ok := data.index[target]; !ok {
			if tv2 := strongConnect(g, target, data); tv2.Lowlink < tv.Lowlink {
				tv.Lowlink = tv2.Lowlink
			}
		} else if data.vertices[idx].Stack {
			if idx < tv.Lowlink {
				tv.Lowlink = idx
			}
		}
	}

	if tv.Lowlink == index {
		vs := make([]Vertex, 0, 2)
		for i := len(data.stack) - 1; i >= 0; i-- {
			v := data.stack[i]
			data.stack[i] = nil
			data.stack = data.stack[:i]
			data.vertices[data.index[v]].Stack = false
			vs = append(vs, v.V)
			if data.index[v] == i {
				break
			}
		}

		data.result = append(data.result, vs)
	}

	return tv
}
