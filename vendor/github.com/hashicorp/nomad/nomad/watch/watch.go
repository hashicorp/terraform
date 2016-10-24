package watch

// The watch package provides a means of describing a watch for a blocking
// query. It is exported so it may be shared between Nomad's RPC layer and
// the underlying state store.

// Item describes the scope of a watch. It is used to provide a uniform
// input for subscribe/unsubscribe and notification firing. Specifying
// multiple fields does not place a watch on multiple items. Each Item
// describes exactly one scoped watch.
type Item struct {
	Alloc      string
	AllocEval  string
	AllocJob   string
	AllocNode  string
	Eval       string
	Job        string
	JobSummary string
	Node       string
	Table      string
}

// Items is a helper used to construct a set of watchItems. It deduplicates
// the items as they are added using map keys.
type Items map[Item]struct{}

// NewItems creates a new Items set and adds the given items.
func NewItems(items ...Item) Items {
	wi := make(Items)
	for _, item := range items {
		wi.Add(item)
	}
	return wi
}

// Add adds an item to the watch set.
func (wi Items) Add(i Item) {
	wi[i] = struct{}{}
}
