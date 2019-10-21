package set

type Iterator struct {
	vals []interface{}
	idx  int
}

func (it *Iterator) Value() interface{} {
	return it.vals[it.idx]
}

func (it *Iterator) Next() bool {
	it.idx++
	return it.idx < len(it.vals)
}
