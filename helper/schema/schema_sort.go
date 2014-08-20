package schema

// listSort implements sort.Interface to sort a list of []interface according
// to a schema.
type listSort struct {
	List   []interface{}
	Map    map[int]int
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

	// Build the mapping. We have to make sure we get to the proper
	// place where the final target is, not the current value.
	if s.Map == nil {
		s.Map = make(map[int]int)
	}
	i2 := i
	j2 := j
	if v, ok := s.Map[i]; ok {
		i2 = v
	}
	if v, ok := s.Map[j]; ok {
		j2 = v
	}
	s.Map[i], s.Map[j] = j2, i2
}
