package set

type Iterator struct {
	bucketIds []int
	vals      map[int][]interface{}
	bucketIdx int
	valIdx    int
}

func (it *Iterator) Value() interface{} {
	return it.currentBucket()[it.valIdx]
}

func (it *Iterator) Next() bool {
	if it.bucketIdx == -1 {
		// init
		if len(it.bucketIds) == 0 {
			return false
		}

		it.valIdx = 0
		it.bucketIdx = 0
		return true
	}

	it.valIdx++
	if it.valIdx >= len(it.currentBucket()) {
		it.valIdx = 0
		it.bucketIdx++
	}
	return it.bucketIdx < len(it.bucketIds)
}

func (it *Iterator) currentBucket() []interface{} {
	return it.vals[it.bucketIds[it.bucketIdx]]
}
