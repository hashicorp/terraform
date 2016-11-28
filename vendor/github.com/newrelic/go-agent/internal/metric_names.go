package internal

const (
	apdexRollup = "Apdex"
	apdexPrefix = "Apdex/"

	webRollup        = "WebTransaction"
	backgroundRollup = "OtherTransaction/all"

	errorsAll        = "Errors/all"
	errorsWeb        = "Errors/allWeb"
	errorsBackground = "Errors/allOther"
	errorsPrefix     = "Errors/"

	// "HttpDispatcher" metric is used for the overview graph, and
	// therefore should only be made for web transactions.
	dispatcherMetric = "HttpDispatcher"

	queueMetric = "WebFrontend/QueueTime"

	webMetricPrefix        = "WebTransaction/Go"
	backgroundMetricPrefix = "OtherTransaction/Go"

	instanceReporting = "Instance/Reporting"

	// https://newrelic.atlassian.net/wiki/display/eng/Custom+Events+in+New+Relic+Agents
	customEventsSeen = "Supportability/Events/Customer/Seen"
	customEventsSent = "Supportability/Events/Customer/Sent"

	// https://source.datanerd.us/agents/agent-specs/blob/master/Transaction-Events-PORTED.md
	txnEventsSeen = "Supportability/AnalyticsEvents/TotalEventsSeen"
	txnEventsSent = "Supportability/AnalyticsEvents/TotalEventsSent"

	// https://source.datanerd.us/agents/agent-specs/blob/master/Error-Events.md
	errorEventsSeen = "Supportability/Events/TransactionError/Seen"
	errorEventsSent = "Supportability/Events/TransactionError/Sent"

	supportabilityDropped = "Supportability/MetricsDropped"

	// source.datanerd.us/agents/agent-specs/blob/master/Datastore-Metrics-PORTED.md
	datastoreAll   = "Datastore/all"
	datastoreWeb   = "Datastore/allWeb"
	datastoreOther = "Datastore/allOther"

	// source.datanerd.us/agents/agent-specs/blob/master/APIs/external_segment.md
	// source.datanerd.us/agents/agent-specs/blob/master/APIs/external_cat.md
	// source.datanerd.us/agents/agent-specs/blob/master/Cross-Application-Tracing-PORTED.md
	externalAll   = "External/all"
	externalWeb   = "External/allWeb"
	externalOther = "External/allOther"

	// Runtime/System Metrics
	memoryPhysical       = "Memory/Physical"
	heapObjectsAllocated = "Memory/Heap/AllocatedObjects"
	cpuUserUtilization   = "CPU/User/Utilization"
	cpuSystemUtilization = "CPU/System/Utilization"
	cpuUserTime          = "CPU/User Time"
	cpuSystemTime        = "CPU/System Time"
	runGoroutine         = "Go/Runtime/Goroutines"
	gcPauseFraction      = "GC/System/Pause Fraction"
	gcPauses             = "GC/System/Pauses"
)

func customSegmentMetric(s string) string {
	return "Custom/" + s
}

// DatastoreMetricKey contains the fields by which datastore metrics are
// aggregated.
type DatastoreMetricKey struct {
	Product      string
	Collection   string
	Operation    string
	Host         string
	PortPathOrID string
}

type externalMetricKey struct {
	Host                    string
	ExternalCrossProcessID  string
	ExternalTransactionName string
}

type datastoreProductMetrics struct {
	All   string // Datastore/{datastore}/all
	Web   string // Datastore/{datastore}/allWeb
	Other string // Datastore/{datastore}/allOther
}

func datastoreScopedMetric(key DatastoreMetricKey) string {
	if "" != key.Collection {
		return datastoreStatementMetric(key)
	}
	return datastoreOperationMetric(key)
}

func datastoreProductMetric(key DatastoreMetricKey) datastoreProductMetrics {
	d, ok := datastoreProductMetricsCache[key.Product]
	if ok {
		return d
	}
	return datastoreProductMetrics{
		All:   "Datastore/" + key.Product + "/all",
		Web:   "Datastore/" + key.Product + "/allWeb",
		Other: "Datastore/" + key.Product + "/allOther",
	}
}

// Datastore/operation/{datastore}/{operation}
func datastoreOperationMetric(key DatastoreMetricKey) string {
	return "Datastore/operation/" + key.Product +
		"/" + key.Operation
}

// Datastore/statement/{datastore}/{table}/{operation}
func datastoreStatementMetric(key DatastoreMetricKey) string {
	return "Datastore/statement/" + key.Product +
		"/" + key.Collection +
		"/" + key.Operation
}

// Datastore/instance/{datastore}/{host}/{port_path_or_id}
func datastoreInstanceMetric(key DatastoreMetricKey) string {
	return "Datastore/instance/" + key.Product +
		"/" + key.Host +
		"/" + key.PortPathOrID
}

// External/{host}/all
func externalHostMetric(key externalMetricKey) string {
	return "External/" + key.Host + "/all"
}

// ExternalApp/{host}/{external_id}/all
func externalAppMetric(key externalMetricKey) string {
	return "ExternalApp/" + key.Host +
		"/" + key.ExternalCrossProcessID + "/all"
}

// ExternalTransaction/{host}/{external_id}/{external_txnname}
func externalTransactionMetric(key externalMetricKey) string {
	return "ExternalTransaction/" + key.Host +
		"/" + key.ExternalCrossProcessID +
		"/" + key.ExternalTransactionName
}
