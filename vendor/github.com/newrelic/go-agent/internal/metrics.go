package internal

import (
	"bytes"
	"time"

	"github.com/newrelic/go-agent/internal/jsonx"
)

type metricForce int

const (
	forced metricForce = iota
	unforced
)

type metricID struct {
	Name  string `json:"name"`
	Scope string `json:"scope,omitempty"`
}

type metricData struct {
	// These values are in the units expected by the collector.
	countSatisfied  float64 // Seconds, or count for Apdex
	totalTolerated  float64 // Seconds, or count for Apdex
	exclusiveFailed float64 // Seconds, or count for Apdex
	min             float64 // Seconds
	max             float64 // Seconds
	sumSquares      float64 // Seconds**2, or 0 for Apdex
}

func metricDataFromDuration(duration, exclusive time.Duration) metricData {
	ds := duration.Seconds()
	return metricData{
		countSatisfied:  1,
		totalTolerated:  ds,
		exclusiveFailed: exclusive.Seconds(),
		min:             ds,
		max:             ds,
		sumSquares:      ds * ds,
	}
}

type metric struct {
	forced metricForce
	data   metricData
}

type metricTable struct {
	metricPeriodStart time.Time
	failedHarvests    int
	maxTableSize      int // After this max is reached, only forced metrics are added
	numDropped        int // Number of unforced metrics dropped due to full table
	metrics           map[metricID]*metric
}

func newMetricTable(maxTableSize int, now time.Time) *metricTable {
	return &metricTable{
		metricPeriodStart: now,
		metrics:           make(map[metricID]*metric),
		maxTableSize:      maxTableSize,
		failedHarvests:    0,
	}
}

func (mt *metricTable) full() bool {
	return len(mt.metrics) >= mt.maxTableSize
}

func (data *metricData) aggregate(src metricData) {
	data.countSatisfied += src.countSatisfied
	data.totalTolerated += src.totalTolerated
	data.exclusiveFailed += src.exclusiveFailed

	if src.min < data.min {
		data.min = src.min
	}
	if src.max > data.max {
		data.max = src.max
	}

	data.sumSquares += src.sumSquares
}

func (mt *metricTable) mergeMetric(id metricID, m metric) {
	if to := mt.metrics[id]; nil != to {
		to.data.aggregate(m.data)
		return
	}

	if mt.full() && (unforced == m.forced) {
		mt.numDropped++
		return
	}
	// NOTE: `new` is used in place of `&m` since the latter will make `m`
	// get heap allocated regardless of whether or not this line gets
	// reached (running go version go1.5 darwin/amd64).  See
	// BenchmarkAddingSameMetrics.
	alloc := new(metric)
	*alloc = m
	mt.metrics[id] = alloc
}

func (mt *metricTable) mergeFailed(from *metricTable) {
	fails := from.failedHarvests + 1
	if fails >= failedMetricAttemptsLimit {
		return
	}
	if from.metricPeriodStart.Before(mt.metricPeriodStart) {
		mt.metricPeriodStart = from.metricPeriodStart
	}
	mt.failedHarvests = fails
	mt.merge(from, "")
}

func (mt *metricTable) merge(from *metricTable, newScope string) {
	if "" == newScope {
		for id, m := range from.metrics {
			mt.mergeMetric(id, *m)
		}
	} else {
		for id, m := range from.metrics {
			mt.mergeMetric(metricID{Name: id.Name, Scope: newScope}, *m)
		}
	}
}

func (mt *metricTable) add(name, scope string, data metricData, force metricForce) {
	mt.mergeMetric(metricID{Name: name, Scope: scope}, metric{data: data, forced: force})
}

func (mt *metricTable) addCount(name string, count float64, force metricForce) {
	mt.add(name, "", metricData{countSatisfied: count}, force)
}

func (mt *metricTable) addSingleCount(name string, force metricForce) {
	mt.addCount(name, float64(1), force)
}

func (mt *metricTable) addDuration(name, scope string, duration, exclusive time.Duration, force metricForce) {
	mt.add(name, scope, metricDataFromDuration(duration, exclusive), force)
}

func (mt *metricTable) addValueExclusive(name, scope string, total, exclusive float64, force metricForce) {
	data := metricData{
		countSatisfied:  1,
		totalTolerated:  total,
		exclusiveFailed: exclusive,
		min:             total,
		max:             total,
		sumSquares:      total * total,
	}
	mt.add(name, scope, data, force)
}

func (mt *metricTable) addValue(name, scope string, total float64, force metricForce) {
	mt.addValueExclusive(name, scope, total, total, force)
}

func (mt *metricTable) addApdex(name, scope string, apdexThreshold time.Duration, zone ApdexZone, force metricForce) {
	apdexSeconds := apdexThreshold.Seconds()
	data := metricData{min: apdexSeconds, max: apdexSeconds}

	switch zone {
	case ApdexSatisfying:
		data.countSatisfied = 1
	case ApdexTolerating:
		data.totalTolerated = 1
	case ApdexFailing:
		data.exclusiveFailed = 1
	}

	mt.add(name, scope, data, force)
}

func (mt *metricTable) CollectorJSON(agentRunID string, now time.Time) ([]byte, error) {
	if 0 == len(mt.metrics) {
		return nil, nil
	}
	estimatedBytesPerMetric := 128
	estimatedLen := len(mt.metrics) * estimatedBytesPerMetric
	buf := bytes.NewBuffer(make([]byte, 0, estimatedLen))
	buf.WriteByte('[')

	jsonx.AppendString(buf, agentRunID)
	buf.WriteByte(',')
	jsonx.AppendInt(buf, mt.metricPeriodStart.Unix())
	buf.WriteByte(',')
	jsonx.AppendInt(buf, now.Unix())
	buf.WriteByte(',')

	buf.WriteByte('[')
	first := true
	for id, metric := range mt.metrics {
		if first {
			first = false
		} else {
			buf.WriteByte(',')
		}
		buf.WriteByte('[')
		buf.WriteByte('{')
		buf.WriteString(`"name":`)
		jsonx.AppendString(buf, id.Name)
		if id.Scope != "" {
			buf.WriteString(`,"scope":`)
			jsonx.AppendString(buf, id.Scope)
		}
		buf.WriteByte('}')
		buf.WriteByte(',')

		jsonx.AppendFloatArray(buf,
			metric.data.countSatisfied,
			metric.data.totalTolerated,
			metric.data.exclusiveFailed,
			metric.data.min,
			metric.data.max,
			metric.data.sumSquares)

		buf.WriteByte(']')
	}
	buf.WriteByte(']')

	buf.WriteByte(']')
	return buf.Bytes(), nil
}

func (mt *metricTable) Data(agentRunID string, harvestStart time.Time) ([]byte, error) {
	return mt.CollectorJSON(agentRunID, harvestStart)
}
func (mt *metricTable) MergeIntoHarvest(h *Harvest) {
	h.Metrics.mergeFailed(mt)
}

func (mt *metricTable) ApplyRules(rules metricRules) *metricTable {
	if nil == rules {
		return mt
	}
	if len(rules) == 0 {
		return mt
	}

	applied := newMetricTable(mt.maxTableSize, mt.metricPeriodStart)
	cache := make(map[string]string)

	for id, m := range mt.metrics {
		out, ok := cache[id.Name]
		if !ok {
			out = rules.Apply(id.Name)
			cache[id.Name] = out
		}

		if "" != out {
			applied.mergeMetric(metricID{Name: out, Scope: id.Scope}, *m)
		}
	}

	return applied
}
