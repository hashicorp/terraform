package internal

import "time"

const (
	// app behavior

	// ConnectBackoff is the wait time between unsuccessful connect
	// attempts.
	ConnectBackoff = 20 * time.Second
	// HarvestPeriod is the period that collected data is sent to New Relic.
	HarvestPeriod = 60 * time.Second
	// CollectorTimeout is the timeout used in the client for communication
	// with New Relic's servers.
	CollectorTimeout = 20 * time.Second
	// AppDataChanSize is the size of the channel that contains data sent
	// the app processor.
	AppDataChanSize           = 200
	failedMetricAttemptsLimit = 5
	failedEventsAttemptsLimit = 10

	// transaction behavior
	maxStackTraceFrames = 100
	// MaxTxnErrors is the maximum number of errors captured per
	// transaction.
	MaxTxnErrors      = 5
	maxTxnTraceNodes  = 256
	maxTxnSlowQueries = 10

	// harvest data
	maxMetrics         = 2 * 1000
	maxCustomEvents    = 10 * 1000
	maxTxnEvents       = 10 * 1000
	maxErrorEvents     = 100
	maxHarvestErrors   = 20
	maxHarvestSlowSQLs = 10

	// attributes
	attributeKeyLengthLimit   = 255
	attributeValueLengthLimit = 255
	attributeUserLimit        = 64
	attributeAgentLimit       = 255 - attributeUserLimit
	customEventAttributeLimit = 64

	// Limits affecting Config validation are found in the config package.

	// RuntimeSamplerPeriod is the period of the runtime sampler.  Runtime
	// metrics should not depend on the sampler period, but the period must
	// be the same across instances.  For that reason, this value should not
	// be changed without notifying customers that they must update all
	// instance simultaneously for valid runtime metrics.
	RuntimeSamplerPeriod = 60 * time.Second
)
