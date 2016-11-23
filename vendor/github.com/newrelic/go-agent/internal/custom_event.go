package internal

import (
	"bytes"
	"fmt"
	"regexp"
	"time"
)

// https://newrelic.atlassian.net/wiki/display/eng/Custom+Events+in+New+Relic+Agents

var (
	eventTypeRegexRaw = `^[a-zA-Z0-9:_ ]+$`
	eventTypeRegex    = regexp.MustCompile(eventTypeRegexRaw)

	errEventTypeLength = fmt.Errorf("event type exceeds length limit of %d",
		attributeKeyLengthLimit)
	// ErrEventTypeRegex will be returned to caller of app.RecordCustomEvent
	// if the event type is not valid.
	ErrEventTypeRegex = fmt.Errorf("event type must match %s", eventTypeRegexRaw)
	errNumAttributes  = fmt.Errorf("maximum of %d attributes exceeded",
		customEventAttributeLimit)
)

// CustomEvent is a custom event.
type CustomEvent struct {
	eventType       string
	timestamp       time.Time
	truncatedParams map[string]interface{}
}

// WriteJSON prepares JSON in the format expected by the collector.
func (e *CustomEvent) WriteJSON(buf *bytes.Buffer) {
	w := jsonFieldsWriter{buf: buf}
	buf.WriteByte('[')
	buf.WriteByte('{')
	w.stringField("type", e.eventType)
	w.floatField("timestamp", timeToFloatSeconds(e.timestamp))
	buf.WriteByte('}')

	buf.WriteByte(',')
	buf.WriteByte('{')
	w = jsonFieldsWriter{buf: buf}
	for key, val := range e.truncatedParams {
		writeAttributeValueJSON(&w, key, val)
	}
	buf.WriteByte('}')

	buf.WriteByte(',')
	buf.WriteByte('{')
	buf.WriteByte('}')
	buf.WriteByte(']')
}

// MarshalJSON is used for testing.
func (e *CustomEvent) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 256))

	e.WriteJSON(buf)

	return buf.Bytes(), nil
}

func eventTypeValidate(eventType string) error {
	if len(eventType) > attributeKeyLengthLimit {
		return errEventTypeLength
	}
	if !eventTypeRegex.MatchString(eventType) {
		return ErrEventTypeRegex
	}
	return nil
}

// CreateCustomEvent creates a custom event.
func CreateCustomEvent(eventType string, params map[string]interface{}, now time.Time) (*CustomEvent, error) {
	if err := eventTypeValidate(eventType); nil != err {
		return nil, err
	}

	if len(params) > customEventAttributeLimit {
		return nil, errNumAttributes
	}

	truncatedParams := make(map[string]interface{})
	for key, val := range params {
		if err := validAttributeKey(key); nil != err {
			return nil, err
		}

		val = truncateStringValueIfLongInterface(val)

		if err := valueIsValid(val); nil != err {
			return nil, err
		}
		truncatedParams[key] = val
	}

	return &CustomEvent{
		eventType:       eventType,
		timestamp:       now,
		truncatedParams: truncatedParams,
	}, nil
}

// MergeIntoHarvest implements Harvestable.
func (e *CustomEvent) MergeIntoHarvest(h *Harvest) {
	h.CustomEvents.Add(e)
}
