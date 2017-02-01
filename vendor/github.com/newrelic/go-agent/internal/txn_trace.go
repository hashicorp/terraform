package internal

import (
	"bytes"
	"container/heap"
	"encoding/json"
	"sort"
	"time"

	"github.com/newrelic/go-agent/internal/jsonx"
)

// See https://source.datanerd.us/agents/agent-specs/blob/master/Transaction-Trace-LEGACY.md

type traceNodeHeap []traceNode

// traceNodeParams is used for trace node parameters.  A struct is used in place
// of a map[string]interface{} to facilitate testing and reduce JSON Marshal
// overhead.  If too many fields get added here, it probably makes sense to
// start using a map.  This struct is not embedded into traceNode to minimize
// the size of traceNode:  Not all nodes will have parameters.
type traceNodeParams struct {
	StackTrace      *StackTrace
	CleanURL        string
	Database        string
	Host            string
	PortPathOrID    string
	Query           string
	queryParameters queryParameters
}

func (p *traceNodeParams) WriteJSON(buf *bytes.Buffer) {
	w := jsonFieldsWriter{buf: buf}
	buf.WriteByte('{')
	if nil != p.StackTrace {
		w.writerField("backtrace", p.StackTrace)
	}
	if "" != p.CleanURL {
		w.stringField("uri", p.CleanURL)
	}
	if "" != p.Database {
		w.stringField("database_name", p.Database)
	}
	if "" != p.Host {
		w.stringField("host", p.Host)
	}
	if "" != p.PortPathOrID {
		w.stringField("port_path_or_id", p.PortPathOrID)
	}
	if "" != p.Query {
		w.stringField("query", p.Query)
	}
	if nil != p.queryParameters {
		w.writerField("query_parameters", p.queryParameters)
	}
	buf.WriteByte('}')
}

// MarshalJSON is used for testing.
func (p *traceNodeParams) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	p.WriteJSON(buf)
	return buf.Bytes(), nil
}

type traceNode struct {
	start    segmentTime
	stop     segmentTime
	duration time.Duration
	params   *traceNodeParams
	name     string
}

func (h traceNodeHeap) Len() int           { return len(h) }
func (h traceNodeHeap) Less(i, j int) bool { return h[i].duration < h[j].duration }
func (h traceNodeHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

// Push and Pop are unused: only heap.Init and heap.Fix are used.
func (h traceNodeHeap) Push(x interface{}) {}
func (h traceNodeHeap) Pop() interface{}   { return nil }

// TxnTrace contains the work in progress transaction trace.
type TxnTrace struct {
	Enabled             bool
	SegmentThreshold    time.Duration
	StackTraceThreshold time.Duration
	nodes               traceNodeHeap
	maxNodes            int
}

// considerNode exists to prevent unnecessary calls to witnessNode: constructing
// the metric name and params map requires allocations.
func (trace *TxnTrace) considerNode(end segmentEnd) bool {
	return trace.Enabled && (end.duration >= trace.SegmentThreshold)
}

func (trace *TxnTrace) witnessNode(end segmentEnd, name string, params *traceNodeParams) {
	node := traceNode{
		start:    end.start,
		stop:     end.stop,
		duration: end.duration,
		name:     name,
		params:   params,
	}
	if !trace.considerNode(end) {
		return
	}
	if trace.nodes == nil {
		max := trace.maxNodes
		if 0 == max {
			max = maxTxnTraceNodes
		}
		trace.nodes = make(traceNodeHeap, 0, max)
	}
	if end.exclusive >= trace.StackTraceThreshold {
		if node.params == nil {
			p := new(traceNodeParams)
			node.params = p
		}
		// skip the following stack frames:
		//   this method
		//   function in tracing.go      (EndBasicSegment, EndExternalSegment, EndDatastoreSegment)
		//   function in internal_txn.go (endSegment, endExternal, endDatastore)
		//   segment end method
		skip := 4
		node.params.StackTrace = GetStackTrace(skip)
	}
	if len(trace.nodes) < cap(trace.nodes) {
		trace.nodes = append(trace.nodes, node)
		if len(trace.nodes) == cap(trace.nodes) {
			heap.Init(trace.nodes)
		}
		return
	}
	if node.duration <= trace.nodes[0].duration {
		return
	}
	trace.nodes[0] = node
	heap.Fix(trace.nodes, 0)
}

// HarvestTrace contains a finished transaction trace ready for serialization to
// the collector.
type HarvestTrace struct {
	Start                time.Time
	Duration             time.Duration
	MetricName           string
	CleanURL             string
	Trace                TxnTrace
	ForcePersist         bool
	GUID                 string
	SyntheticsResourceID string
	Attrs                *Attributes
}

type nodeDetails struct {
	name          string
	relativeStart time.Duration
	relativeStop  time.Duration
	params        *traceNodeParams
}

func printNodeStart(buf *bytes.Buffer, n nodeDetails) {
	// time.Seconds() is intentionally not used here.  Millisecond
	// precision is enough.
	relativeStartMillis := n.relativeStart.Nanoseconds() / (1000 * 1000)
	relativeStopMillis := n.relativeStop.Nanoseconds() / (1000 * 1000)

	buf.WriteByte('[')
	jsonx.AppendInt(buf, relativeStartMillis)
	buf.WriteByte(',')
	jsonx.AppendInt(buf, relativeStopMillis)
	buf.WriteByte(',')
	jsonx.AppendString(buf, n.name)
	buf.WriteByte(',')
	if nil == n.params {
		buf.WriteString("{}")
	} else {
		n.params.WriteJSON(buf)
	}
	buf.WriteByte(',')
	buf.WriteByte('[')
}

func printChildren(buf *bytes.Buffer, traceStart time.Time, nodes sortedTraceNodes, next int, stop segmentStamp) int {
	firstChild := true
	for next < len(nodes) && nodes[next].start.Stamp < stop {
		if firstChild {
			firstChild = false
		} else {
			buf.WriteByte(',')
		}
		printNodeStart(buf, nodeDetails{
			name:          nodes[next].name,
			relativeStart: nodes[next].start.Time.Sub(traceStart),
			relativeStop:  nodes[next].stop.Time.Sub(traceStart),
			params:        nodes[next].params,
		})
		next = printChildren(buf, traceStart, nodes, next+1, nodes[next].stop.Stamp)
		buf.WriteString("]]")

	}
	return next
}

type sortedTraceNodes []*traceNode

func (s sortedTraceNodes) Len() int           { return len(s) }
func (s sortedTraceNodes) Less(i, j int) bool { return s[i].start.Stamp < s[j].start.Stamp }
func (s sortedTraceNodes) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func traceDataJSON(trace *HarvestTrace) []byte {
	estimate := 100 * len(trace.Trace.nodes)
	buf := bytes.NewBuffer(make([]byte, 0, estimate))

	nodes := make(sortedTraceNodes, len(trace.Trace.nodes))
	for i := 0; i < len(nodes); i++ {
		nodes[i] = &trace.Trace.nodes[i]
	}
	sort.Sort(nodes)

	buf.WriteByte('[') // begin trace data

	// If the trace string pool is used, insert another array here.

	jsonx.AppendFloat(buf, 0.0) // unused timestamp
	buf.WriteByte(',')          //
	buf.WriteString("{}")       // unused: formerly request parameters
	buf.WriteByte(',')          //
	buf.WriteString("{}")       // unused: formerly custom parameters
	buf.WriteByte(',')          //

	printNodeStart(buf, nodeDetails{ // begin outer root
		name:          "ROOT",
		relativeStart: 0,
		relativeStop:  trace.Duration,
	})

	printNodeStart(buf, nodeDetails{ // begin inner root
		name:          trace.MetricName,
		relativeStart: 0,
		relativeStop:  trace.Duration,
	})

	if len(nodes) > 0 {
		lastStopStamp := nodes[len(nodes)-1].stop.Stamp + 1
		printChildren(buf, trace.Start, nodes, 0, lastStopStamp)
	}

	buf.WriteString("]]") // end outer root
	buf.WriteString("]]") // end inner root

	buf.WriteByte(',')
	buf.WriteByte('{')
	buf.WriteString(`"agentAttributes":`)
	agentAttributesJSON(trace.Attrs, buf, destTxnTrace)
	buf.WriteByte(',')
	buf.WriteString(`"userAttributes":`)
	userAttributesJSON(trace.Attrs, buf, destTxnTrace)
	buf.WriteByte(',')
	buf.WriteString(`"intrinsics":{}`) // TODO intrinsics
	buf.WriteByte('}')

	// If the trace string pool is used, end another array here.

	buf.WriteByte(']') // end trace data

	return buf.Bytes()
}

// MarshalJSON prepares the trace in the JSON expected by the collector.
func (trace *HarvestTrace) MarshalJSON() ([]byte, error) {
	return json.Marshal([]interface{}{
		trace.Start.UnixNano() / 1000,
		trace.Duration.Seconds() * 1000.0,
		trace.MetricName,
		trace.CleanURL,
		JSONString(traceDataJSON(trace)),
		trace.GUID,
		nil, // reserved for future use
		trace.ForcePersist,
		nil, // X-Ray sessions not supported
		trace.SyntheticsResourceID,
	})
}

type harvestTraces struct {
	trace *HarvestTrace
}

func newHarvestTraces() *harvestTraces {
	return &harvestTraces{}
}

func (traces *harvestTraces) Witness(trace HarvestTrace) {
	if nil == traces.trace || traces.trace.Duration < trace.Duration {
		cpy := new(HarvestTrace)
		*cpy = trace
		traces.trace = cpy
	}
}

func (traces *harvestTraces) Data(agentRunID string, harvestStart time.Time) ([]byte, error) {
	if nil == traces.trace {
		return nil, nil
	}
	return json.Marshal([]interface{}{
		agentRunID,
		[]interface{}{
			traces.trace,
		},
	})
}

func (traces *harvestTraces) MergeIntoHarvest(h *Harvest) {}
