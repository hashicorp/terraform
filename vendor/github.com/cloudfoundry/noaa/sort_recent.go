package noaa

import (
	"sort"

	"github.com/cloudfoundry/sonde-go/events"
)

// SortRecent sorts a slice of LogMessages by timestamp. The sort is stable, so messages with the same timestamp are sorted
// in the order that they are received.
//
// The input slice is sorted; the return value is simply a pointer to the same slice.
func SortRecent(messages []*events.LogMessage) []*events.LogMessage {
	sort.Stable(logMessageSlice(messages))
	return messages
}

type logMessageSlice []*events.LogMessage

func (lms logMessageSlice) Len() int {
	return len(lms)
}

func (lms logMessageSlice) Less(i, j int) bool {
	return *(lms[i]).Timestamp < *(lms[j]).Timestamp
}

func (lms logMessageSlice) Swap(i, j int) {
	lms[i], lms[j] = lms[j], lms[i]
}
