package schema

// listSort implements sort.Interface to sort a list of []interface according
// to a schema.
type listSort struct {
	List   []interface{}
	Schema *Schema
}

func (s *listSort) Len() int {
	return len(s.List)
}

func (s *listSort) Less(i, j int) bool {
	return s.Schema.Order(s.List[i], s.List[j])
}

func (s *listSort) Swap(i, j int) {
	s.List[i], s.List[j] = s.List[j], s.List[i]
}
