package internal

import (
	"math/rand"
	"time"
)

type customEvents struct {
	events *analyticsEvents
}

func newCustomEvents(max int) *customEvents {
	return &customEvents{
		events: newAnalyticsEvents(max),
	}
}

func (cs *customEvents) Add(e *CustomEvent) {
	stamp := eventStamp(rand.Float32())
	cs.events.addEvent(analyticsEvent{stamp, e})
}

func (cs *customEvents) MergeIntoHarvest(h *Harvest) {
	h.CustomEvents.events.mergeFailed(cs.events)
}

func (cs *customEvents) Data(agentRunID string, harvestStart time.Time) ([]byte, error) {
	return cs.events.CollectorJSON(agentRunID)
}

func (cs *customEvents) numSeen() float64  { return cs.events.NumSeen() }
func (cs *customEvents) numSaved() float64 { return cs.events.NumSaved() }
