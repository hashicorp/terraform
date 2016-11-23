## ChangeLog

## 1.5.0

* Added support for Windows.  Thanks to @ianomad and @lvxv for the contributions.

* The number of heap objects allocated is recorded in the
  `Memory/Heap/AllocatedObjects` metric.  This will soon be displayed on the "Go
  runtime" page.

* If the [DatastoreSegment](https://godoc.org/github.com/newrelic/go-agent#DatastoreSegment)
  fields `Host` and `PortPathOrID` are not provided, they will no longer appear
  as `"unknown"` in transaction traces and slow query traces.

* Stack traces will now be nicely aligned in the APM UI.

## 1.4.0

* Added support for slow query traces.  Slow datastore segments will now
 generate slow query traces viewable on the datastore tab.  These traces include
 a stack trace and help you to debug slow datastore activity.
 [Slow Query Documentation](https://docs.newrelic.com/docs/apm/applications-menu/monitoring/viewing-slow-query-details)

* Added new
[DatastoreSegment](https://godoc.org/github.com/newrelic/go-agent#DatastoreSegment)
fields `ParameterizedQuery`, `QueryParameters`, `Host`, `PortPathOrID`, and
`DatabaseName`.  These fields will be shown in transaction traces and in slow
query traces.

## 1.3.0

* Breaking Change: Added a timeout parameter to the `Application.Shutdown` method.

## 1.2.0

* Added support for instrumenting short-lived processes:
  * The new `Application.Shutdown` method allows applications to report
    data to New Relic without waiting a full minute.
  * The new `Application.WaitForConnection` method allows your process to
    defer instrumentation until the application is connected and ready to
    gather data.
  * Full documentation here: [application.go](application.go)
  * Example short-lived process: [examples/short-lived-process/main.go](examples/short-lived-process/main.go)

* Error metrics are no longer created when `ErrorCollector.Enabled = false`.

* Added support for [github.com/mgutz/logxi](github.com/mgutz/logxi).  See
  [_integrations/nrlogxi/v1/nrlogxi.go](_integrations/nrlogxi/v1/nrlogxi.go).

* Fixed bug where Transaction Trace thresholds based upon Apdex were not being
  applied to background transactions.

## 1.1.0

* Added support for Transaction Traces.

* Stack trace filenames have been shortened: Any thing preceding the first
  `/src/` is now removed.

## 1.0.0

* Removed `BetaToken` from the `Config` structure.

* Breaking Datastore Change:  `datastore` package contents moved to top level
  `newrelic` package.  `datastore.MySQL` has become `newrelic.DatastoreMySQL`.

* Breaking Attributes Change:  `attributes` package contents moved to top
  level `newrelic` package.  `attributes.ResponseCode` has become
  `newrelic.AttributeResponseCode`.  Some attribute name constants have been
  shortened.

* Added "runtime.NumCPU" to the environment tab.  Thanks sergeylanzman for the
  contribution.

* Prefixed the environment tab values "Compiler", "GOARCH", "GOOS", and
  "Version" with "runtime.".

## 0.8.0

* Breaking Segments API Changes:  The segments API has been rewritten with the
  goal of being easier to use and to avoid nil Transaction checks.  See:

  * [segments.go](segments.go)
  * [examples/server/main.go](examples/server/main.go)
  * [GUIDE.md#segments](GUIDE.md#segments)

* Updated LICENSE.txt with contribution information.

## 0.7.1

* Fixed a bug causing the `Config` to fail to serialize into JSON when the
  `Transport` field was populated.

## 0.7.0

* Eliminated `api`, `version`, and `log` packages.  `Version`, `Config`,
  `Application`, and `Transaction` now live in the top level `newrelic` package.
  If you imported the  `attributes` or `datastore` packages then you will need
  to remove `api` from the import path.

* Breaking Logging Changes

Logging is no longer controlled though a single global.  Instead, logging is
configured on a per-application basis with the new `Config.Logger` field.  The
logger is an interface described in [log.go](log.go).  See
[GUIDE.md#logging](GUIDE.md#logging).

## 0.6.1

* No longer create "GC/System/Pauses" metric if no GC pauses happened.

## 0.6.0

* Introduced beta token to support our beta program.

* Rename `Config.Development` to `Config.Enabled` (and change boolean
  direction).

* Fixed a bug where exclusive time could be incorrect if segments were not
  ended.

* Fix unit tests broken in 1.6.

* In `Config.Enabled = false` mode, the license must be the proper length or empty.

* Added runtime statistics for CPU/memory usage, garbage collection, and number
  of goroutines.

## 0.5.0

* Added segment timing methods to `Transaction`.  These methods must only be
  used in a single goroutine.

* The license length check will not be performed in `Development` mode.

* Rename `SetLogFile` to `SetFile` to reduce redundancy.

* Added `DebugEnabled` logging guard to reduce overhead.

* `Transaction` now implements an `Ignore` method which will prevent
  any of the transaction's data from being recorded.

* `Transaction` now implements a subset of the interfaces
  `http.CloseNotifier`, `http.Flusher`, `http.Hijacker`, and `io.ReaderFrom`
  to match the behavior of its wrapped `http.ResponseWriter`.

* Changed project name from `go-sdk` to `go-agent`.

## 0.4.0

* Queue time support added: if the inbound request contains an
`"X-Request-Start"` or `"X-Queue-Start"` header with a unix timestamp, the
agent will report queue time metrics.  Queue time will appear on the
application overview chart.  The timestamp may fractional seconds,
milliseconds, or microseconds: the agent will deduce the correct units.
