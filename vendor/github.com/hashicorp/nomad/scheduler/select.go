package scheduler

// LimitIterator is a RankIterator used to limit the number of options
// that are returned before we artificially end the stream.
type LimitIterator struct {
	ctx    Context
	source RankIterator
	limit  int
	seen   int
}

// NewLimitIterator is returns a LimitIterator with a fixed limit of returned options
func NewLimitIterator(ctx Context, source RankIterator, limit int) *LimitIterator {
	iter := &LimitIterator{
		ctx:    ctx,
		source: source,
		limit:  limit,
	}
	return iter
}

func (iter *LimitIterator) SetLimit(limit int) {
	iter.limit = limit
}

func (iter *LimitIterator) Next() *RankedNode {
	if iter.seen == iter.limit {
		return nil
	}

	option := iter.source.Next()
	if option == nil {
		return nil
	}

	iter.seen += 1
	return option
}

func (iter *LimitIterator) Reset() {
	iter.source.Reset()
	iter.seen = 0
}

// MaxScoreIterator is a RankIterator used to return only a single result
// of the item with the highest score. This iterator will consume all of the
// possible inputs and only returns the highest ranking result.
type MaxScoreIterator struct {
	ctx    Context
	source RankIterator
	max    *RankedNode
}

// MaxScoreIterator returns a MaxScoreIterator over the given source
func NewMaxScoreIterator(ctx Context, source RankIterator) *MaxScoreIterator {
	iter := &MaxScoreIterator{
		ctx:    ctx,
		source: source,
	}
	return iter
}

func (iter *MaxScoreIterator) Next() *RankedNode {
	// Check if we've found the max, return nil
	if iter.max != nil {
		return nil
	}

	// Consume and determine the max
	for {
		option := iter.source.Next()
		if option == nil {
			return iter.max
		}

		if iter.max == nil || option.Score > iter.max.Score {
			iter.max = option
		}
	}
}

func (iter *MaxScoreIterator) Reset() {
	iter.source.Reset()
	iter.max = nil
}
