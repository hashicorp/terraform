package internal

import (
	"strings"
	"sync"
	"time"
)

// Harvestable is something that can be merged into a Harvest.
type Harvestable interface {
	MergeIntoHarvest(h *Harvest)
}

// Harvest contains collected data.
type Harvest struct {
	Metrics      *metricTable
	CustomEvents *customEvents
	TxnEvents    *txnEvents
	ErrorEvents  *errorEvents
	ErrorTraces  *harvestErrors
	TxnTraces    *harvestTraces
	SlowSQLs     *slowQueries
}

// Payloads returns a map from expected collector method name to data type.
func (h *Harvest) Payloads() map[string]PayloadCreator {
	return map[string]PayloadCreator{
		cmdMetrics:      h.Metrics,
		cmdCustomEvents: h.CustomEvents,
		cmdTxnEvents:    h.TxnEvents,
		cmdErrorEvents:  h.ErrorEvents,
		cmdErrorData:    h.ErrorTraces,
		cmdTxnTraces:    h.TxnTraces,
		cmdSlowSQLs:     h.SlowSQLs,
	}
}

// NewHarvest returns a new Harvest.
func NewHarvest(now time.Time) *Harvest {
	return &Harvest{
		Metrics:      newMetricTable(maxMetrics, now),
		CustomEvents: newCustomEvents(maxCustomEvents),
		TxnEvents:    newTxnEvents(maxTxnEvents),
		ErrorEvents:  newErrorEvents(maxErrorEvents),
		ErrorTraces:  newHarvestErrors(maxHarvestErrors),
		TxnTraces:    newHarvestTraces(),
		SlowSQLs:     newSlowQueries(maxHarvestSlowSQLs),
	}
}

var (
	trackMutex   sync.Mutex
	trackMetrics []string
)

// TrackUsage helps track which integration packages are used.
func TrackUsage(s ...string) {
	trackMutex.Lock()
	defer trackMutex.Unlock()

	m := "Supportability/" + strings.Join(s, "/")
	trackMetrics = append(trackMetrics, m)
}

func createTrackUsageMetrics(metrics *metricTable) {
	trackMutex.Lock()
	defer trackMutex.Unlock()

	for _, m := range trackMetrics {
		metrics.addSingleCount(m, forced)
	}
}

// CreateFinalMetrics creates extra metrics at harvest time.
func (h *Harvest) CreateFinalMetrics() {
	h.Metrics.addSingleCount(instanceReporting, forced)

	h.Metrics.addCount(customEventsSeen, h.CustomEvents.numSeen(), forced)
	h.Metrics.addCount(customEventsSent, h.CustomEvents.numSaved(), forced)

	h.Metrics.addCount(txnEventsSeen, h.TxnEvents.numSeen(), forced)
	h.Metrics.addCount(txnEventsSent, h.TxnEvents.numSaved(), forced)

	h.Metrics.addCount(errorEventsSeen, h.ErrorEvents.numSeen(), forced)
	h.Metrics.addCount(errorEventsSent, h.ErrorEvents.numSaved(), forced)

	if h.Metrics.numDropped > 0 {
		h.Metrics.addCount(supportabilityDropped, float64(h.Metrics.numDropped), forced)
	}

	createTrackUsageMetrics(h.Metrics)
}

// PayloadCreator is a data type in the harvest.
type PayloadCreator interface {
	// In the event of a rpm request failure (hopefully simply an
	// intermittent collector issue) the payload may be merged into the next
	// time period's harvest.
	Harvestable
	// Data prepares JSON in the format expected by the collector endpoint.
	// This method should return (nil, nil) if the payload is empty and no
	// rpm request is necessary.
	Data(agentRunID string, harvestStart time.Time) ([]byte, error)
}

// CreateTxnMetricsArgs contains the parameters to CreateTxnMetrics.
type CreateTxnMetricsArgs struct {
	IsWeb          bool
	Duration       time.Duration
	Exclusive      time.Duration
	Name           string
	Zone           ApdexZone
	ApdexThreshold time.Duration
	HasErrors      bool
	Queueing       time.Duration
}

// CreateTxnMetrics creates metrics for a transaction.
func CreateTxnMetrics(args CreateTxnMetricsArgs, metrics *metricTable) {
	// Duration Metrics
	rollup := backgroundRollup
	if args.IsWeb {
		rollup = webRollup
		metrics.addDuration(dispatcherMetric, "", args.Duration, 0, forced)
	}

	metrics.addDuration(args.Name, "", args.Duration, args.Exclusive, forced)
	metrics.addDuration(rollup, "", args.Duration, args.Exclusive, forced)

	// Apdex Metrics
	if args.Zone != ApdexNone {
		metrics.addApdex(apdexRollup, "", args.ApdexThreshold, args.Zone, forced)

		mname := apdexPrefix + removeFirstSegment(args.Name)
		metrics.addApdex(mname, "", args.ApdexThreshold, args.Zone, unforced)
	}

	// Error Metrics
	if args.HasErrors {
		metrics.addSingleCount(errorsAll, forced)
		if args.IsWeb {
			metrics.addSingleCount(errorsWeb, forced)
		} else {
			metrics.addSingleCount(errorsBackground, forced)
		}
		metrics.addSingleCount(errorsPrefix+args.Name, forced)
	}

	// Queueing Metrics
	if args.Queueing > 0 {
		metrics.addDuration(queueMetric, "", args.Queueing, args.Queueing, forced)
	}
}
