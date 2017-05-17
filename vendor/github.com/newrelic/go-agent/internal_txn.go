package newrelic

import (
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/newrelic/go-agent/internal"
)

type txnInput struct {
	W          http.ResponseWriter
	Request    *http.Request
	Config     Config
	Reply      *internal.ConnectReply
	Consumer   dataConsumer
	attrConfig *internal.AttributeConfig
}

type txn struct {
	txnInput
	// This mutex is required since the consumer may call the public API
	// interface functions from different routines.
	sync.Mutex
	// finished indicates whether or not End() has been called.  After
	// finished has been set to true, no recording should occur.
	finished bool
	queuing  time.Duration
	start    time.Time
	name     string // Work in progress name
	isWeb    bool
	ignore   bool
	errors   internal.TxnErrors // Lazily initialized.
	attrs    *internal.Attributes

	// Fields relating to tracing and breakdown metrics/segments.
	tracer internal.Tracer

	// wroteHeader prevents capturing multiple response code errors if the
	// user erroneously calls WriteHeader multiple times.
	wroteHeader bool

	// Fields assigned at completion
	stop           time.Time
	duration       time.Duration
	finalName      string // Full finalized metric name
	zone           internal.ApdexZone
	apdexThreshold time.Duration
}

func newTxn(input txnInput, name string) *txn {
	txn := &txn{
		txnInput: input,
		start:    time.Now(),
		name:     name,
		isWeb:    nil != input.Request,
		attrs:    internal.NewAttributes(input.attrConfig),
	}
	if nil != txn.Request {
		txn.queuing = internal.QueueDuration(input.Request.Header, txn.start)
		internal.RequestAgentAttributes(txn.attrs, input.Request)
	}
	txn.attrs.Agent.HostDisplayName = txn.Config.HostDisplayName
	txn.tracer.Enabled = txn.txnTracesEnabled()
	txn.tracer.SegmentThreshold = txn.Config.TransactionTracer.SegmentThreshold
	txn.tracer.StackTraceThreshold = txn.Config.TransactionTracer.StackTraceThreshold
	txn.tracer.SlowQueriesEnabled = txn.slowQueriesEnabled()
	txn.tracer.SlowQueryThreshold = txn.Config.DatastoreTracer.SlowQuery.Threshold

	return txn
}

func (txn *txn) slowQueriesEnabled() bool {
	return txn.Config.DatastoreTracer.SlowQuery.Enabled &&
		txn.Reply.CollectTraces
}

func (txn *txn) txnTracesEnabled() bool {
	return txn.Config.TransactionTracer.Enabled &&
		txn.Reply.CollectTraces
}

func (txn *txn) txnEventsEnabled() bool {
	return txn.Config.TransactionEvents.Enabled &&
		txn.Reply.CollectAnalyticsEvents
}

func (txn *txn) errorEventsEnabled() bool {
	return txn.Config.ErrorCollector.CaptureEvents &&
		txn.Reply.CollectErrorEvents
}

func (txn *txn) freezeName() {
	if txn.ignore || ("" != txn.finalName) {
		return
	}

	txn.finalName = internal.CreateFullTxnName(txn.name, txn.Reply, txn.isWeb)
	if "" == txn.finalName {
		txn.ignore = true
	}
}

func (txn *txn) getsApdex() bool {
	return txn.isWeb
}

func (txn *txn) txnTraceThreshold() time.Duration {
	if txn.Config.TransactionTracer.Threshold.IsApdexFailing {
		return internal.ApdexFailingThreshold(txn.apdexThreshold)
	}
	return txn.Config.TransactionTracer.Threshold.Duration
}

func (txn *txn) shouldSaveTrace() bool {
	return txn.txnTracesEnabled() &&
		(txn.duration >= txn.txnTraceThreshold())
}

func (txn *txn) hasErrors() bool {
	return len(txn.errors) > 0
}

func (txn *txn) MergeIntoHarvest(h *internal.Harvest) {
	exclusive := time.Duration(0)
	children := internal.TracerRootChildren(&txn.tracer)
	if txn.duration > children {
		exclusive = txn.duration - children
	}

	internal.CreateTxnMetrics(internal.CreateTxnMetricsArgs{
		IsWeb:          txn.isWeb,
		Duration:       txn.duration,
		Exclusive:      exclusive,
		Name:           txn.finalName,
		Zone:           txn.zone,
		ApdexThreshold: txn.apdexThreshold,
		HasErrors:      txn.hasErrors(),
		Queueing:       txn.queuing,
	}, h.Metrics)

	internal.MergeBreakdownMetrics(&txn.tracer, h.Metrics, txn.finalName, txn.isWeb)

	if txn.txnEventsEnabled() {
		h.TxnEvents.AddTxnEvent(&internal.TxnEvent{
			Name:      txn.finalName,
			Timestamp: txn.start,
			Duration:  txn.duration,
			Queuing:   txn.queuing,
			Zone:      txn.zone,
			Attrs:     txn.attrs,
			DatastoreExternalTotals: txn.tracer.DatastoreExternalTotals,
		})
	}

	requestURI := ""
	if nil != txn.Request && nil != txn.Request.URL {
		requestURI = internal.SafeURL(txn.Request.URL)
	}

	internal.MergeTxnErrors(h.ErrorTraces, txn.errors, txn.finalName, requestURI, txn.attrs)

	if txn.errorEventsEnabled() {
		for _, e := range txn.errors {
			h.ErrorEvents.Add(&internal.ErrorEvent{
				Klass:    e.Klass,
				Msg:      e.Msg,
				When:     e.When,
				TxnName:  txn.finalName,
				Duration: txn.duration,
				Queuing:  txn.queuing,
				Attrs:    txn.attrs,
				DatastoreExternalTotals: txn.tracer.DatastoreExternalTotals,
			})
		}
	}

	if txn.shouldSaveTrace() {
		h.TxnTraces.Witness(internal.HarvestTrace{
			Start:                txn.start,
			Duration:             txn.duration,
			MetricName:           txn.finalName,
			CleanURL:             requestURI,
			Trace:                txn.tracer.TxnTrace,
			ForcePersist:         false,
			GUID:                 "",
			SyntheticsResourceID: "",
			Attrs:                txn.attrs,
		})
	}

	if nil != txn.tracer.SlowQueries {
		h.SlowSQLs.Merge(txn.tracer.SlowQueries, txn.finalName, requestURI)
	}
}

func responseCodeIsError(cfg *Config, code int) bool {
	if code < http.StatusBadRequest { // 400
		return false
	}
	for _, ignoreCode := range cfg.ErrorCollector.IgnoreStatusCodes {
		if code == ignoreCode {
			return false
		}
	}
	return true
}

func headersJustWritten(txn *txn, code int) {
	if txn.finished {
		return
	}
	if txn.wroteHeader {
		return
	}
	txn.wroteHeader = true

	internal.ResponseHeaderAttributes(txn.attrs, txn.W.Header())
	internal.ResponseCodeAttribute(txn.attrs, code)

	if responseCodeIsError(&txn.Config, code) {
		e := internal.TxnErrorFromResponseCode(time.Now(), code)
		e.Stack = internal.GetStackTrace(1)
		txn.noticeErrorInternal(e)
	}
}

func (txn *txn) Header() http.Header { return txn.W.Header() }

func (txn *txn) Write(b []byte) (int, error) {
	n, err := txn.W.Write(b)

	txn.Lock()
	defer txn.Unlock()

	headersJustWritten(txn, http.StatusOK)

	return n, err
}

func (txn *txn) WriteHeader(code int) {
	txn.W.WriteHeader(code)

	txn.Lock()
	defer txn.Unlock()

	headersJustWritten(txn, code)
}

func (txn *txn) End() error {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return errAlreadyEnded
	}

	txn.finished = true

	r := recover()
	if nil != r {
		e := internal.TxnErrorFromPanic(time.Now(), r)
		e.Stack = internal.GetStackTrace(0)
		txn.noticeErrorInternal(e)
	}

	txn.stop = time.Now()
	txn.duration = txn.stop.Sub(txn.start)

	txn.freezeName()

	// Assign apdexThreshold regardless of whether or not the transaction
	// gets apdex since it may be used to calculate the trace threshold.
	txn.apdexThreshold = internal.CalculateApdexThreshold(txn.Reply, txn.finalName)

	if txn.getsApdex() {
		if txn.hasErrors() {
			txn.zone = internal.ApdexFailing
		} else {
			txn.zone = internal.CalculateApdexZone(txn.apdexThreshold, txn.duration)
		}
	} else {
		txn.zone = internal.ApdexNone
	}

	if txn.Config.Logger.DebugEnabled() {
		txn.Config.Logger.Debug("transaction ended", map[string]interface{}{
			"name":        txn.finalName,
			"duration_ms": txn.duration.Seconds() * 1000.0,
			"ignored":     txn.ignore,
			"run":         txn.Reply.RunID,
		})
	}

	if !txn.ignore {
		txn.Consumer.Consume(txn.Reply.RunID, txn)
	}

	// Note that if a consumer uses `panic(nil)`, the panic will not
	// propagate.
	if nil != r {
		panic(r)
	}

	return nil
}

func (txn *txn) AddAttribute(name string, value interface{}) error {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return errAlreadyEnded
	}

	return internal.AddUserAttribute(txn.attrs, name, value, internal.DestAll)
}

var (
	errorsLocallyDisabled  = errors.New("errors locally disabled")
	errorsRemotelyDisabled = errors.New("errors remotely disabled")
	errNilError            = errors.New("nil error")
	errAlreadyEnded        = errors.New("transaction has already ended")
)

const (
	highSecurityErrorMsg = "message removed by high security setting"
)

func (txn *txn) noticeErrorInternal(err internal.TxnError) error {
	if !txn.Config.ErrorCollector.Enabled {
		return errorsLocallyDisabled
	}

	if !txn.Reply.CollectErrors {
		return errorsRemotelyDisabled
	}

	if nil == txn.errors {
		txn.errors = internal.NewTxnErrors(internal.MaxTxnErrors)
	}

	if txn.Config.HighSecurity {
		err.Msg = highSecurityErrorMsg
	}

	txn.errors.Add(err)

	return nil
}

func (txn *txn) NoticeError(err error) error {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return errAlreadyEnded
	}

	if nil == err {
		return errNilError
	}

	e := internal.TxnErrorFromError(time.Now(), err)
	e.Stack = internal.GetStackTrace(2)
	return txn.noticeErrorInternal(e)
}

func (txn *txn) SetName(name string) error {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return errAlreadyEnded
	}

	txn.name = name
	return nil
}

func (txn *txn) Ignore() error {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return errAlreadyEnded
	}
	txn.ignore = true
	return nil
}

func (txn *txn) StartSegmentNow() SegmentStartTime {
	var s internal.SegmentStartTime
	txn.Lock()
	if !txn.finished {
		s = internal.StartSegment(&txn.tracer, time.Now())
	}
	txn.Unlock()
	return SegmentStartTime{
		segment: segment{
			start: s,
			txn:   txn,
		},
	}
}

type segment struct {
	start internal.SegmentStartTime
	txn   *txn
}

func endSegment(s Segment) {
	txn := s.StartTime.txn
	if nil == txn {
		return
	}
	txn.Lock()
	if !txn.finished {
		internal.EndBasicSegment(&txn.tracer, s.StartTime.start, time.Now(), s.Name)
	}
	txn.Unlock()
}

func endDatastore(s DatastoreSegment) {
	txn := s.StartTime.txn
	if nil == txn {
		return
	}
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return
	}
	if txn.Config.HighSecurity {
		s.QueryParameters = nil
	}
	if !txn.Config.DatastoreTracer.QueryParameters.Enabled {
		s.QueryParameters = nil
	}
	if !txn.Config.DatastoreTracer.DatabaseNameReporting.Enabled {
		s.DatabaseName = ""
	}
	if !txn.Config.DatastoreTracer.InstanceReporting.Enabled {
		s.Host = ""
		s.PortPathOrID = ""
	}
	internal.EndDatastoreSegment(internal.EndDatastoreParams{
		Tracer:             &txn.tracer,
		Start:              s.StartTime.start,
		Now:                time.Now(),
		Product:            string(s.Product),
		Collection:         s.Collection,
		Operation:          s.Operation,
		ParameterizedQuery: s.ParameterizedQuery,
		QueryParameters:    s.QueryParameters,
		Host:               s.Host,
		PortPathOrID:       s.PortPathOrID,
		Database:           s.DatabaseName,
	})
}

func externalSegmentURL(s ExternalSegment) *url.URL {
	if "" != s.URL {
		u, _ := url.Parse(s.URL)
		return u
	}
	r := s.Request
	if nil != s.Response && nil != s.Response.Request {
		r = s.Response.Request
	}
	if r != nil {
		return r.URL
	}
	return nil
}

func endExternal(s ExternalSegment) {
	txn := s.StartTime.txn
	if nil == txn {
		return
	}
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return
	}
	internal.EndExternalSegment(&txn.tracer, s.StartTime.start, time.Now(), externalSegmentURL(s))
}
