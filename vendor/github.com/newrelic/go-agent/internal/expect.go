package internal

import (
	"fmt"
	"runtime"
	"time"
)

var (
	// Unfortunately, the resolution of time.Now() on Windows is coarse: Two
	// sequential calls to time.Now() may return the same value, and tests
	// which expect non-zero durations may fail.  To avoid adding sleep
	// statements or mocking time.Now(), those tests are skipped on Windows.
	doDurationTests = runtime.GOOS != `windows`
)

// Validator is used for testing.
type Validator interface {
	Error(...interface{})
}

func validateStringField(v Validator, fieldName, v1, v2 string) {
	if v1 != v2 {
		v.Error(fieldName, v1, v2)
	}
}

type addValidatorField struct {
	field    interface{}
	original Validator
}

func (a addValidatorField) Error(fields ...interface{}) {
	fields = append([]interface{}{a.field}, fields...)
	a.original.Error(fields...)
}

// ExtendValidator is used to add more context to a validator.
func ExtendValidator(v Validator, field interface{}) Validator {
	return addValidatorField{
		field:    field,
		original: v,
	}
}

// WantMetric is a metric expectation.  If Data is nil, then any data values are
// acceptable.
type WantMetric struct {
	Name   string
	Scope  string
	Forced interface{} // true, false, or nil
	Data   []float64
}

// WantCustomEvent is a custom event expectation.
type WantCustomEvent struct {
	Type   string
	Params map[string]interface{}
}

// WantError is a traced error expectation.
type WantError struct {
	TxnName         string
	Msg             string
	Klass           string
	Caller          string
	URL             string
	UserAttributes  map[string]interface{}
	AgentAttributes map[string]interface{}
}

// WantErrorEvent is an error event expectation.
type WantErrorEvent struct {
	TxnName            string
	Msg                string
	Klass              string
	Queuing            bool
	ExternalCallCount  uint64
	DatastoreCallCount uint64
	UserAttributes     map[string]interface{}
	AgentAttributes    map[string]interface{}
}

// WantTxnEvent is a transaction event expectation.
type WantTxnEvent struct {
	Name               string
	Zone               string
	Queuing            bool
	ExternalCallCount  uint64
	DatastoreCallCount uint64
	UserAttributes     map[string]interface{}
	AgentAttributes    map[string]interface{}
}

// WantTxnTrace is a transaction trace expectation.
type WantTxnTrace struct {
	MetricName      string
	CleanURL        string
	NumSegments     int
	UserAttributes  map[string]interface{}
	AgentAttributes map[string]interface{}
}

// WantSlowQuery is a slowQuery expectation.
type WantSlowQuery struct {
	Count        int32
	MetricName   string
	Query        string
	TxnName      string
	TxnURL       string
	DatabaseName string
	Host         string
	PortPathOrID string
	Params       map[string]interface{}
}

// Expect exposes methods that allow for testing whether the correct data was
// captured.
type Expect interface {
	ExpectCustomEvents(t Validator, want []WantCustomEvent)
	ExpectErrors(t Validator, want []WantError)
	ExpectErrorEvents(t Validator, want []WantErrorEvent)
	ExpectTxnEvents(t Validator, want []WantTxnEvent)
	ExpectMetrics(t Validator, want []WantMetric)
	ExpectTxnTraces(t Validator, want []WantTxnTrace)
	ExpectSlowQueries(t Validator, want []WantSlowQuery)
}

func expectMetricField(t Validator, id metricID, v1, v2 float64, fieldName string) {
	if v1 != v2 {
		t.Error("metric fields do not match", id, v1, v2, fieldName)
	}
}

// ExpectMetrics allows testing of metrics.
func ExpectMetrics(t Validator, mt *metricTable, expect []WantMetric) {
	if len(mt.metrics) != len(expect) {
		t.Error("metric counts do not match expectations", len(mt.metrics), len(expect))
	}
	expectedIds := make(map[metricID]struct{})
	for _, e := range expect {
		id := metricID{Name: e.Name, Scope: e.Scope}
		expectedIds[id] = struct{}{}
		m := mt.metrics[id]
		if nil == m {
			t.Error("unable to find metric", id)
			continue
		}

		if b, ok := e.Forced.(bool); ok {
			if b != (forced == m.forced) {
				t.Error("metric forced incorrect", b, m.forced, id)
			}
		}

		if nil != e.Data {
			expectMetricField(t, id, e.Data[0], m.data.countSatisfied, "countSatisfied")
			expectMetricField(t, id, e.Data[1], m.data.totalTolerated, "totalTolerated")
			expectMetricField(t, id, e.Data[2], m.data.exclusiveFailed, "exclusiveFailed")
			expectMetricField(t, id, e.Data[3], m.data.min, "min")
			expectMetricField(t, id, e.Data[4], m.data.max, "max")
			expectMetricField(t, id, e.Data[5], m.data.sumSquares, "sumSquares")
		}
	}
	for id := range mt.metrics {
		if _, ok := expectedIds[id]; !ok {
			t.Error("expected metrics does not contain", id.Name, id.Scope)
		}
	}
}

func expectAttributes(v Validator, exists map[string]interface{}, expect map[string]interface{}) {
	// TODO: This params comparison can be made smarter: Alert differences
	// based on sub/super set behavior.
	if len(exists) != len(expect) {
		v.Error("attributes length difference", exists, expect)
		return
	}
	for key, val := range expect {
		found, ok := exists[key]
		if !ok {
			v.Error("missing key", key)
			continue
		}
		v1 := fmt.Sprint(found)
		v2 := fmt.Sprint(val)
		if v1 != v2 {
			v.Error("value difference", fmt.Sprintf("key=%s", key),
				v1, v2)
		}
	}
}

func expectCustomEvent(v Validator, event *CustomEvent, expect WantCustomEvent) {
	if event.eventType != expect.Type {
		v.Error("type mismatch", event.eventType, expect.Type)
	}
	now := time.Now()
	diff := absTimeDiff(now, event.timestamp)
	if diff > time.Hour {
		v.Error("large timestamp difference", event.eventType, now, event.timestamp)
	}
	expectAttributes(v, event.truncatedParams, expect.Params)
}

// ExpectCustomEvents allows testing of custom events.
func ExpectCustomEvents(v Validator, cs *customEvents, expect []WantCustomEvent) {
	if len(cs.events.events) != len(expect) {
		v.Error("number of custom events does not match", len(cs.events.events),
			len(expect))
		return
	}
	for i, e := range expect {
		event, ok := cs.events.events[i].jsonWriter.(*CustomEvent)
		if !ok {
			v.Error("wrong custom event")
		} else {
			expectCustomEvent(v, event, e)
		}
	}
}

func expectErrorEvent(v Validator, err *ErrorEvent, expect WantErrorEvent) {
	validateStringField(v, "txnName", expect.TxnName, err.TxnName)
	validateStringField(v, "klass", expect.Klass, err.Klass)
	validateStringField(v, "msg", expect.Msg, err.Msg)
	if (0 != err.Queuing) != expect.Queuing {
		v.Error("queuing", err.Queuing)
	}
	if nil != expect.UserAttributes {
		expectAttributes(v, getUserAttributes(err.Attrs, destError), expect.UserAttributes)
	}
	if nil != expect.AgentAttributes {
		expectAttributes(v, getAgentAttributes(err.Attrs, destError), expect.AgentAttributes)
	}
	if expect.ExternalCallCount != err.externalCallCount {
		v.Error("external call count", expect.ExternalCallCount, err.externalCallCount)
	}
	if doDurationTests && (0 == expect.ExternalCallCount) != (err.externalDuration == 0) {
		v.Error("external duration", err.externalDuration)
	}
	if expect.DatastoreCallCount != err.datastoreCallCount {
		v.Error("datastore call count", expect.DatastoreCallCount, err.datastoreCallCount)
	}
	if doDurationTests && (0 == expect.DatastoreCallCount) != (err.datastoreDuration == 0) {
		v.Error("datastore duration", err.datastoreDuration)
	}
}

// ExpectErrorEvents allows testing of error events.
func ExpectErrorEvents(v Validator, events *errorEvents, expect []WantErrorEvent) {
	if len(events.events.events) != len(expect) {
		v.Error("number of custom events does not match",
			len(events.events.events), len(expect))
		return
	}
	for i, e := range expect {
		event, ok := events.events.events[i].jsonWriter.(*ErrorEvent)
		if !ok {
			v.Error("wrong error event")
		} else {
			expectErrorEvent(v, event, e)
		}
	}
}

func expectTxnEvent(v Validator, e *TxnEvent, expect WantTxnEvent) {
	validateStringField(v, "apdex zone", expect.Zone, e.Zone.label())
	validateStringField(v, "name", expect.Name, e.Name)
	if doDurationTests && 0 == e.Duration {
		v.Error("zero duration", e.Duration)
	}
	if (0 != e.Queuing) != expect.Queuing {
		v.Error("queuing", e.Queuing)
	}
	if nil != expect.UserAttributes {
		expectAttributes(v, getUserAttributes(e.Attrs, destTxnEvent), expect.UserAttributes)
	}
	if nil != expect.AgentAttributes {
		expectAttributes(v, getAgentAttributes(e.Attrs, destTxnEvent), expect.AgentAttributes)
	}
	if expect.ExternalCallCount != e.externalCallCount {
		v.Error("external call count", expect.ExternalCallCount, e.externalCallCount)
	}
	if doDurationTests && (0 == expect.ExternalCallCount) != (e.externalDuration == 0) {
		v.Error("external duration", e.externalDuration)
	}
	if expect.DatastoreCallCount != e.datastoreCallCount {
		v.Error("datastore call count", expect.DatastoreCallCount, e.datastoreCallCount)
	}
	if doDurationTests && (0 == expect.DatastoreCallCount) != (e.datastoreDuration == 0) {
		v.Error("datastore duration", e.datastoreDuration)
	}
}

// ExpectTxnEvents allows testing of txn events.
func ExpectTxnEvents(v Validator, events *txnEvents, expect []WantTxnEvent) {
	if len(events.events.events) != len(expect) {
		v.Error("number of txn events does not match",
			len(events.events.events), len(expect))
		return
	}
	for i, e := range expect {
		event, ok := events.events.events[i].jsonWriter.(*TxnEvent)
		if !ok {
			v.Error("wrong txn event")
		} else {
			expectTxnEvent(v, event, e)
		}
	}
}

func expectError(v Validator, err *harvestError, expect WantError) {
	caller := topCallerNameBase(err.TxnError.Stack)
	validateStringField(v, "caller", expect.Caller, caller)
	validateStringField(v, "txnName", expect.TxnName, err.txnName)
	validateStringField(v, "klass", expect.Klass, err.TxnError.Klass)
	validateStringField(v, "msg", expect.Msg, err.TxnError.Msg)
	validateStringField(v, "URL", expect.URL, err.requestURI)
	if nil != expect.UserAttributes {
		expectAttributes(v, getUserAttributes(err.attrs, destError), expect.UserAttributes)
	}
	if nil != expect.AgentAttributes {
		expectAttributes(v, getAgentAttributes(err.attrs, destError), expect.AgentAttributes)
	}
}

// ExpectErrors allows testing of errors.
func ExpectErrors(v Validator, errors *harvestErrors, expect []WantError) {
	if len(errors.errors) != len(expect) {
		v.Error("number of errors mismatch", len(errors.errors), len(expect))
		return
	}
	for i, e := range expect {
		expectError(v, errors.errors[i], e)
	}
}

func expectTxnTrace(v Validator, trace *HarvestTrace, expect WantTxnTrace) {
	if doDurationTests && 0 == trace.Duration {
		v.Error("zero trace duration")
	}
	validateStringField(v, "metric name", expect.MetricName, trace.MetricName)
	validateStringField(v, "request url", expect.CleanURL, trace.CleanURL)
	if nil != expect.UserAttributes {
		expectAttributes(v, getUserAttributes(trace.Attrs, destTxnTrace), expect.UserAttributes)
	}
	if nil != expect.AgentAttributes {
		expectAttributes(v, getAgentAttributes(trace.Attrs, destTxnTrace), expect.AgentAttributes)
	}
	if expect.NumSegments != len(trace.Trace.nodes) {
		v.Error("wrong number of segments", expect.NumSegments, len(trace.Trace.nodes))
	}
}

// ExpectTxnTraces allows testing of transaction traces.
func ExpectTxnTraces(v Validator, traces *harvestTraces, want []WantTxnTrace) {
	if len(want) == 0 {
		if nil != traces.trace {
			v.Error("trace exists when not expected")
		}
	} else if len(want) > 1 {
		v.Error("too many traces expected")
	} else {
		if nil == traces.trace {
			v.Error("missing expected trace")
		} else {
			expectTxnTrace(v, traces.trace, want[0])
		}
	}
}

func expectSlowQuery(t Validator, slowQuery *slowQuery, want WantSlowQuery) {
	if slowQuery.Count != want.Count {
		t.Error("wrong Count field", slowQuery.Count, want.Count)
	}
	validateStringField(t, "MetricName", slowQuery.DatastoreMetric, want.MetricName)
	validateStringField(t, "Query", slowQuery.ParameterizedQuery, want.Query)
	validateStringField(t, "TxnName", slowQuery.TxnName, want.TxnName)
	validateStringField(t, "TxnURL", slowQuery.TxnURL, want.TxnURL)
	validateStringField(t, "DatabaseName", slowQuery.DatabaseName, want.DatabaseName)
	validateStringField(t, "Host", slowQuery.Host, want.Host)
	validateStringField(t, "PortPathOrID", slowQuery.PortPathOrID, want.PortPathOrID)
	expectAttributes(t, map[string]interface{}(slowQuery.QueryParameters), want.Params)
}

// ExpectSlowQueries allows testing of slow queries.
func ExpectSlowQueries(t Validator, slowQueries *slowQueries, want []WantSlowQuery) {
	if len(want) != len(slowQueries.priorityQueue) {
		t.Error("wrong number of slow queries",
			"expected", len(want), "got", len(slowQueries.priorityQueue))
		return
	}
	for _, s := range want {
		idx, ok := slowQueries.lookup[s.Query]
		if !ok {
			t.Error("unable to find slow query", s.Query)
			continue
		}
		expectSlowQuery(t, slowQueries.priorityQueue[idx], s)
	}
}
