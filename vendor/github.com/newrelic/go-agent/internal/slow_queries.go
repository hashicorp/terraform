package internal

import (
	"bytes"
	"container/heap"
	"hash/fnv"
	"time"

	"github.com/newrelic/go-agent/internal/jsonx"
)

type queryParameters map[string]interface{}

func vetQueryParameters(params map[string]interface{}) queryParameters {
	if nil == params {
		return nil
	}
	// Copying the parameters into a new map is safer than modifying the map
	// from the customer.
	vetted := make(map[string]interface{})
	for key, val := range params {
		if err := validAttributeKey(key); nil != err {
			continue
		}
		val = truncateStringValueIfLongInterface(val)
		if err := valueIsValid(val); nil != err {
			continue
		}
		vetted[key] = val
	}
	return queryParameters(vetted)
}

func (q queryParameters) WriteJSON(buf *bytes.Buffer) {
	buf.WriteByte('{')
	w := jsonFieldsWriter{buf: buf}
	for key, val := range q {
		writeAttributeValueJSON(&w, key, val)
	}
	buf.WriteByte('}')
}

// https://source.datanerd.us/agents/agent-specs/blob/master/Slow-SQLs-LEGACY.md

// slowQueryInstance represents a single datastore call.
type slowQueryInstance struct {
	// Fields populated right after the datastore segment finishes:

	Duration           time.Duration
	DatastoreMetric    string
	ParameterizedQuery string
	QueryParameters    queryParameters
	Host               string
	PortPathOrID       string
	DatabaseName       string
	StackTrace         *StackTrace

	// Fields populated when merging into the harvest:

	TxnName string
	TxnURL  string
}

// Aggregation is performed to avoid reporting multiple slow queries with same
// query string.  Since some datastore segments may be below the slow query
// threshold, the aggregation fields Count, Total, and Min should be taken with
// a grain of salt.
type slowQuery struct {
	Count int32         // number of times the query has been observed
	Total time.Duration // cummulative duration
	Min   time.Duration // minimum observed duration

	// When Count > 1, slowQueryInstance contains values from the slowest
	// observation.
	slowQueryInstance
}

type slowQueries struct {
	priorityQueue []*slowQuery
	// lookup maps query strings to indices in the priorityQueue
	lookup map[string]int
}

func (slows *slowQueries) Len() int {
	return len(slows.priorityQueue)
}
func (slows *slowQueries) Less(i, j int) bool {
	pq := slows.priorityQueue
	return pq[i].Duration < pq[j].Duration
}
func (slows *slowQueries) Swap(i, j int) {
	pq := slows.priorityQueue
	si := pq[i]
	sj := pq[j]
	pq[i], pq[j] = pq[j], pq[i]
	slows.lookup[si.ParameterizedQuery] = j
	slows.lookup[sj.ParameterizedQuery] = i
}

// Push and Pop are unused: only heap.Init and heap.Fix are used.
func (slows *slowQueries) Push(x interface{}) {}
func (slows *slowQueries) Pop() interface{}   { return nil }

func newSlowQueries(max int) *slowQueries {
	return &slowQueries{
		lookup:        make(map[string]int, max),
		priorityQueue: make([]*slowQuery, 0, max),
	}
}

// Merge is used to merge slow queries from the transaction into the harvest.
func (slows *slowQueries) Merge(other *slowQueries, txnName, txnURL string) {
	for _, s := range other.priorityQueue {
		cp := *s
		cp.TxnName = txnName
		cp.TxnURL = txnURL
		slows.observe(cp)
	}
}

// merge aggregates the observations from two slow queries with the same Query.
func (slow *slowQuery) merge(other slowQuery) {
	slow.Count += other.Count
	slow.Total += other.Total

	if other.Min < slow.Min {
		slow.Min = other.Min
	}
	if other.Duration > slow.Duration {
		slow.slowQueryInstance = other.slowQueryInstance
	}
}

func (slows *slowQueries) observeInstance(slow slowQueryInstance) {
	slows.observe(slowQuery{
		Count:             1,
		Total:             slow.Duration,
		Min:               slow.Duration,
		slowQueryInstance: slow,
	})
}

func (slows *slowQueries) insertAtIndex(slow slowQuery, idx int) {
	cpy := new(slowQuery)
	*cpy = slow
	slows.priorityQueue[idx] = cpy
	slows.lookup[slow.ParameterizedQuery] = idx
	heap.Fix(slows, idx)
}

func (slows *slowQueries) observe(slow slowQuery) {
	// Has the query has previously been observed?
	if idx, ok := slows.lookup[slow.ParameterizedQuery]; ok {
		slows.priorityQueue[idx].merge(slow)
		heap.Fix(slows, idx)
		return
	}
	// Has the collection reached max capacity?
	if len(slows.priorityQueue) < cap(slows.priorityQueue) {
		idx := len(slows.priorityQueue)
		slows.priorityQueue = slows.priorityQueue[0 : idx+1]
		slows.insertAtIndex(slow, idx)
		return
	}
	// Is this query slower than the existing fastest?
	fastest := slows.priorityQueue[0]
	if slow.Duration > fastest.Duration {
		delete(slows.lookup, fastest.ParameterizedQuery)
		slows.insertAtIndex(slow, 0)
		return
	}
}

// The third element of the slow query JSON should be a hash of the query
// string.  This hash may be used by backend services to aggregate queries which
// have the have the same query string.  It is unknown if this actually used.
func makeSlowQueryID(query string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(query))
	return h.Sum32()
}

func (slow *slowQuery) WriteJSON(buf *bytes.Buffer) {
	buf.WriteByte('[')
	jsonx.AppendString(buf, slow.TxnName)
	buf.WriteByte(',')
	jsonx.AppendString(buf, slow.TxnURL)
	buf.WriteByte(',')
	jsonx.AppendInt(buf, int64(makeSlowQueryID(slow.ParameterizedQuery)))
	buf.WriteByte(',')
	jsonx.AppendString(buf, slow.ParameterizedQuery)
	buf.WriteByte(',')
	jsonx.AppendString(buf, slow.DatastoreMetric)
	buf.WriteByte(',')
	jsonx.AppendInt(buf, int64(slow.Count))
	buf.WriteByte(',')
	jsonx.AppendFloat(buf, slow.Total.Seconds()*1000.0)
	buf.WriteByte(',')
	jsonx.AppendFloat(buf, slow.Min.Seconds()*1000.0)
	buf.WriteByte(',')
	jsonx.AppendFloat(buf, slow.Duration.Seconds()*1000.0)
	buf.WriteByte(',')
	w := jsonFieldsWriter{buf: buf}
	buf.WriteByte('{')
	if "" != slow.Host {
		w.stringField("host", slow.Host)
	}
	if "" != slow.PortPathOrID {
		w.stringField("port_path_or_id", slow.PortPathOrID)
	}
	if "" != slow.DatabaseName {
		w.stringField("database_name", slow.DatabaseName)
	}
	if nil != slow.StackTrace {
		w.writerField("backtrace", slow.StackTrace)
	}
	if nil != slow.QueryParameters {
		w.writerField("query_parameters", slow.QueryParameters)
	}
	buf.WriteByte('}')
	buf.WriteByte(']')
}

// WriteJSON marshals the collection of slow queries into JSON according to the
// schema expected by the collector.
//
// Note: This JSON does not contain the agentRunID.  This is for unknown
// historical reasons. Since the agentRunID is included in the url,
// its use in the other commands' JSON is redundant (although required).
func (slows *slowQueries) WriteJSON(buf *bytes.Buffer) {
	buf.WriteByte('[')
	buf.WriteByte('[')
	for idx, s := range slows.priorityQueue {
		if idx > 0 {
			buf.WriteByte(',')
		}
		s.WriteJSON(buf)
	}
	buf.WriteByte(']')
	buf.WriteByte(']')
}

func (slows *slowQueries) Data(agentRunID string, harvestStart time.Time) ([]byte, error) {
	if 0 == len(slows.priorityQueue) {
		return nil, nil
	}
	estimate := 1024 * len(slows.priorityQueue)
	buf := bytes.NewBuffer(make([]byte, 0, estimate))
	slows.WriteJSON(buf)
	return buf.Bytes(), nil
}

func (slows *slowQueries) MergeIntoHarvest(newHarvest *Harvest) {
}
