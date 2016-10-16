package nomad

import (
	"sort"
	"sync"
	"time"

	"github.com/ugorji/go/codec"
)

// TimeTable is used to associate a Raft index with a timestamp.
// This is used so that we can quickly go from a timestamp to an
// index or visa versa.
type TimeTable struct {
	granularity time.Duration
	limit       time.Duration
	table       []TimeTableEntry
	l           sync.RWMutex
}

// TimeTableEntry is used to track a time and index
type TimeTableEntry struct {
	Index uint64
	Time  time.Time
}

// NewTimeTable creates a new time table which stores entries
// at a given granularity for a maximum limit. The storage space
// required is (limit/granularity)
func NewTimeTable(granularity time.Duration, limit time.Duration) *TimeTable {
	size := limit / granularity
	if size < 1 {
		size = 1
	}
	t := &TimeTable{
		granularity: granularity,
		limit:       limit,
		table:       make([]TimeTableEntry, 1, size),
	}
	return t
}

// Serialize is used to serialize the time table
func (t *TimeTable) Serialize(enc *codec.Encoder) error {
	t.l.RLock()
	defer t.l.RUnlock()
	return enc.Encode(t.table)
}

// Deserialize is used to deserialize the time table
// and restore the state
func (t *TimeTable) Deserialize(dec *codec.Decoder) error {
	// Decode the table
	var table []TimeTableEntry
	if err := dec.Decode(&table); err != nil {
		return err
	}

	// Witness from oldest to newest
	n := len(table)
	for i := n - 1; i >= 0; i-- {
		t.Witness(table[i].Index, table[i].Time)
	}
	return nil
}

// Witness is used to witness a new index and time.
func (t *TimeTable) Witness(index uint64, when time.Time) {
	t.l.Lock()
	defer t.l.Unlock()

	// Ensure monotonic indexes
	if t.table[0].Index > index {
		return
	}

	// Skip if we already have a recent enough entry
	if when.Sub(t.table[0].Time) < t.granularity {
		return
	}

	// Grow the table if we haven't reached the size
	if len(t.table) < cap(t.table) {
		t.table = append(t.table, TimeTableEntry{})
	}

	// Add this entry
	copy(t.table[1:], t.table[:len(t.table)-1])
	t.table[0].Index = index
	t.table[0].Time = when
}

// NearestIndex returns the nearest index older than the given time
func (t *TimeTable) NearestIndex(when time.Time) uint64 {
	t.l.RLock()
	defer t.l.RUnlock()

	n := len(t.table)
	idx := sort.Search(n, func(i int) bool {
		return !t.table[i].Time.After(when)
	})
	if idx < n && idx >= 0 {
		return t.table[idx].Index
	}
	return 0
}

// NearestTime returns the nearest time older than the given index
func (t *TimeTable) NearestTime(index uint64) time.Time {
	t.l.RLock()
	defer t.l.RUnlock()

	n := len(t.table)
	idx := sort.Search(n, func(i int) bool {
		return t.table[i].Index <= index
	})
	if idx < n && idx >= 0 {
		return t.table[idx].Time
	}
	return time.Time{}
}
