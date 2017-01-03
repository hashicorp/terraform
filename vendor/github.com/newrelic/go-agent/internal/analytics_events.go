package internal

import (
	"bytes"
	"container/heap"

	"github.com/newrelic/go-agent/internal/jsonx"
)

// eventStamp allows for uniform random sampling of events.  When an event is
// created it is given an eventStamp.  Whenever an event pool is full and events
// need to be dropped, the events with the lowest stamps are dropped.
type eventStamp float32

func eventStampCmp(a, b eventStamp) bool {
	return a < b
}

type analyticsEvent struct {
	stamp eventStamp
	jsonWriter
}

type analyticsEventHeap []analyticsEvent

type analyticsEvents struct {
	numSeen        int
	events         analyticsEventHeap
	failedHarvests int
}

func (events *analyticsEvents) NumSeen() float64  { return float64(events.numSeen) }
func (events *analyticsEvents) NumSaved() float64 { return float64(len(events.events)) }

func (h analyticsEventHeap) Len() int           { return len(h) }
func (h analyticsEventHeap) Less(i, j int) bool { return eventStampCmp(h[i].stamp, h[j].stamp) }
func (h analyticsEventHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

// Push and Pop are unused: only heap.Init and heap.Fix are used.
func (h analyticsEventHeap) Push(x interface{}) {}
func (h analyticsEventHeap) Pop() interface{}   { return nil }

func newAnalyticsEvents(max int) *analyticsEvents {
	return &analyticsEvents{
		numSeen:        0,
		events:         make(analyticsEventHeap, 0, max),
		failedHarvests: 0,
	}
}

func (events *analyticsEvents) addEvent(e analyticsEvent) {
	events.numSeen++

	if len(events.events) < cap(events.events) {
		events.events = append(events.events, e)
		if len(events.events) == cap(events.events) {
			// Delay heap initialization so that we can have
			// deterministic ordering for integration tests (the max
			// is not being reached).
			heap.Init(events.events)
		}
		return
	}

	if eventStampCmp(e.stamp, events.events[0].stamp) {
		return
	}

	events.events[0] = e
	heap.Fix(events.events, 0)
}

func (events *analyticsEvents) mergeFailed(other *analyticsEvents) {
	fails := other.failedHarvests + 1
	if fails >= failedEventsAttemptsLimit {
		return
	}
	events.failedHarvests = fails
	events.Merge(other)
}

func (events *analyticsEvents) Merge(other *analyticsEvents) {
	allSeen := events.numSeen + other.numSeen

	for _, e := range other.events {
		events.addEvent(e)
	}
	events.numSeen = allSeen
}

func (events *analyticsEvents) CollectorJSON(agentRunID string) ([]byte, error) {
	if 0 == events.numSeen {
		return nil, nil
	}

	estimate := 256 * len(events.events)
	buf := bytes.NewBuffer(make([]byte, 0, estimate))

	buf.WriteByte('[')
	jsonx.AppendString(buf, agentRunID)
	buf.WriteByte(',')
	buf.WriteByte('{')
	buf.WriteString(`"reservoir_size":`)
	jsonx.AppendUint(buf, uint64(cap(events.events)))
	buf.WriteByte(',')
	buf.WriteString(`"events_seen":`)
	jsonx.AppendUint(buf, uint64(events.numSeen))
	buf.WriteByte('}')
	buf.WriteByte(',')
	buf.WriteByte('[')
	for i, e := range events.events {
		if i > 0 {
			buf.WriteByte(',')
		}
		e.WriteJSON(buf)
	}
	buf.WriteByte(']')
	buf.WriteByte(']')

	return buf.Bytes(), nil

}
