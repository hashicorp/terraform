# Notes: cycle rendering and deterministic traversal

These are human reference notes from our discussion, not agent instructions.

## What `TestGraphDot_flakeyCycles` demonstrates

The test in [graph_representations_test.go](../terraform/graph_representations_test.go)
constructs a hub with three bidirectional spokes, plus an edge to a root:

```text
A -> B    B -> A
A -> C    C -> A
A -> D    D -> A
A -> root
```

All four vertices A, B, C, and D belong to one strongly connected component:
each can reach every other. All six spoke edges participate in cycles; the
edge to root does not.

[`Cycles()`](dag.go) returns strongly connected components with more than one
vertex. It does not return ordered edge paths or enumerate individual cycles.
Self-loops are excluded and need separate handling.

### Why the output varies

The Tarjan implementation in [tarjan.go](tarjan.go) iterates both the graph's
vertices and each vertex's outgoing neighbors using map-backed sets.
Go map iteration order is unspecified, so the depth-first traversal can start
at different vertices and explore neighbors in different orders.

Vertices are appended to a component as they are popped from Tarjan's stack.
The component's membership is unchanged by traversal order, but its returned
vertex ordering can change. For a simple cycle, that ordering can appear
reversed relative to the actual edges; for a more complex component, it need
not describe a cycle path in either direction.

### Why this is a rendering bug, not multiple valid outputs

The DOT cycle-highlighting logic discussed in [dot.go](dot.go) treats consecutive
component entries, including wraparound, as directed edges. It constructs those
edges without checking that they exist in the graph.

For example, interpreting `[B, C, D, A]` as a cycle creates:

```text
B -> C -> D -> A -> B
```

The input graph has neither `B -> C` nor `C -> D`. The test's expected outputs
include such invented edges. Calling these "multiple valid ways" to represent
the graph is therefore misleading. Accepting several snapshots accommodates
the bug, and the three snapshots discussed are not guaranteed to cover all
possible traversal orders.

The desired output should preserve the seven actual edges, highlight all six
spoke edges, and leave `A -> root` unhighlighted.

## Making traversal deterministic

To make Tarjan's traversal deterministic for vertices with unique names, sort
at both choice points in [tarjan.go](tarjan.go):

1. The outer loop in `StronglyConnected`, which chooses starting vertices.
2. The outgoing-neighbor loop in `stronglyConnected`, which chooses recursive
   traversal routes.

Sorting only the outer loop is insufficient.

The proposed comparator uses `strings.Compare` (requiring a `strings` import;
the file already imports `slices`):

```go
func compareVertexNames(a, b Vertex) int {
	return strings.Compare(a.Name(), b.Name())
}
```

Replace the outer iteration with:

```go
vertices := g.VerticesSeq().Collect()
slices.SortFunc(vertices, compareVertexNames)

for _, v := range vertices {
	if acct.VertexIndex[v] == 0 {
		stronglyConnected(&acct, g, v)
	}
}
```

Replace the recursive neighbor iteration with:

```go
for _, target := range slices.SortedFunc(
	g.edgesFrom[v].All(),
	compareVertexNames,
) {
	// Keep the existing traversal body.
}
```

This is a proposal, not a record that these changes were applied.

### Limitations

- Sorting by name guarantees a stable traversal only if names are unique and
  stable. Equal names require a stable, unique tie-breaker. Pointer addresses
  are not stable across runs; a stable sort alone cannot resolve map-order ties.
- Sorting adds allocation and sorting work to cycle detection.
- Most importantly, sorting makes the rendering bug repeatable, not correct.
  A deterministic component ordering is still not an edge path.

## Preferred rendering fix

Make rendering independent of component ordering:

1. Build a map from each vertex ID to its cyclic component index.
2. Iterate the graph's actual edges.
3. Mark an edge as cyclic only when both endpoints are present in the map and
   belong to the same component.
4. Sort emitted edges separately for deterministic output.

Check map membership explicitly: a missing entry must not be mistaken for
component zero. Handle self-loops separately if highlighting them is intended.

This preserves edge direction, avoids inventing edges, and handles overlapping
cycles. Reversing a component list, or checking consecutive pairs in both
directions, does not solve the general problem.

For the hub example, every circular ordering gives A only two adjacent entries,
but A has three neighbors. A consecutive-pair approach must miss at least one
spoke even if it checks both directions.

## Related Mermaid observations

The Mermaid implementation discussed used concatenated vertex IDs to identify
cycle pairs. Bare concatenation can collide: `AB` + `C` and `A` + `BC` both
produce `ABC`. Component membership avoids needing those pair keys; a typed
pair key is another option when edge keys are needed.

`classDef cycle ...` only defines a style. It does not identify cycles or apply
the style automatically. Appending `:::cycle` to the target in an edge
declaration styles that node, not the edge.

If node highlighting is intended, collect all vertices in cyclic components
and apply the class directly to those nodes. If edge highlighting is intended,
use actual cyclic edges and Mermaid edge styling, such as `linkStyle` with
zero-based indices in emitted edge order.