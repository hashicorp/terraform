package internal

import (
	"bytes"
	"math/rand"
	"time"
)

// ErrorEvent is an error event.
type ErrorEvent struct {
	Klass    string
	Msg      string
	When     time.Time
	TxnName  string
	Duration time.Duration
	Queuing  time.Duration
	Attrs    *Attributes
	DatastoreExternalTotals
}

// MarshalJSON is used for testing.
func (e *ErrorEvent) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 256))

	e.WriteJSON(buf)

	return buf.Bytes(), nil
}

// WriteJSON prepares JSON in the format expected by the collector.
// https://source.datanerd.us/agents/agent-specs/blob/master/Error-Events.md
func (e *ErrorEvent) WriteJSON(buf *bytes.Buffer) {
	w := jsonFieldsWriter{buf: buf}
	buf.WriteByte('[')
	buf.WriteByte('{')
	w.stringField("type", "TransactionError")
	w.stringField("error.class", e.Klass)
	w.stringField("error.message", e.Msg)
	w.floatField("timestamp", timeToFloatSeconds(e.When))
	w.stringField("transactionName", e.TxnName)
	w.floatField("duration", e.Duration.Seconds())
	if e.Queuing > 0 {
		w.floatField("queueDuration", e.Queuing.Seconds())
	}
	if e.externalCallCount > 0 {
		w.intField("externalCallCount", int64(e.externalCallCount))
		w.floatField("externalDuration", e.externalDuration.Seconds())
	}
	if e.datastoreCallCount > 0 {
		// Note that "database" is used for the keys here instead of
		// "datastore" for historical reasons.
		w.intField("databaseCallCount", int64(e.datastoreCallCount))
		w.floatField("databaseDuration", e.datastoreDuration.Seconds())
	}
	buf.WriteByte('}')
	buf.WriteByte(',')
	userAttributesJSON(e.Attrs, buf, destError)
	buf.WriteByte(',')
	agentAttributesJSON(e.Attrs, buf, destError)
	buf.WriteByte(']')
}

type errorEvents struct {
	events *analyticsEvents
}

func newErrorEvents(max int) *errorEvents {
	return &errorEvents{
		events: newAnalyticsEvents(max),
	}
}

func (events *errorEvents) Add(e *ErrorEvent) {
	stamp := eventStamp(rand.Float32())
	events.events.addEvent(analyticsEvent{stamp, e})
}

func (events *errorEvents) MergeIntoHarvest(h *Harvest) {
	h.ErrorEvents.events.mergeFailed(events.events)
}

func (events *errorEvents) Data(agentRunID string, harvestStart time.Time) ([]byte, error) {
	return events.events.CollectorJSON(agentRunID)
}

func (events *errorEvents) numSeen() float64  { return events.events.NumSeen() }
func (events *errorEvents) numSaved() float64 { return events.events.NumSaved() }
