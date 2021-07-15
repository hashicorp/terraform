package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	basictracer "github.com/opentracing/basictracer-go"
	"github.com/opentracing/opentracing-go"
)

var traceRecorder traceLogRecorder

func initTracing() {
	if fn := os.Getenv("TF_TRACE_FILE"); fn != "" {
		traceFile, err := os.Create(fn)
		if err == nil {
			log.Printf("[TRACE] Trace spans will go into %s", fn)
			traceRecorder.outputFile = traceFile
			traceRecorder.spans = make([]traceSpan, 0, 26)
		} else {
			log.Printf("[WARN] Failed to open trace file: %s", err)
		}
	}

	opts := basictracer.DefaultOptions()
	opts.Recorder = &traceRecorder
	opts.NewSpanEventListener = func() func(basictracer.SpanEvent) {
		return traceRecorder.HandleEvent
	}

	tracer := basictracer.NewWithOptions(opts)
	opentracing.SetGlobalTracer(tracer)
}

func closeTracing() {
	if traceRecorder.outputFile != nil {
		sort.Slice(traceRecorder.spans, func(i, j int) bool {
			return traceRecorder.spans[i].StartTime.Before(traceRecorder.spans[j].StartTime)
		})
		traceJSON, err := json.MarshalIndent(traceRecorder.spans, "", "  ")
		if err == nil {
			traceRecorder.outputFile.Write(traceJSON)
			traceRecorder.outputFile.Close()
		}
	}
}

type traceLogRecorder struct {
	outputFile *os.File

	mu    sync.Mutex
	spans []traceSpan
}

var _ basictracer.SpanRecorder = (*traceLogRecorder)(nil)

func (r *traceLogRecorder) RecordSpan(span basictracer.RawSpan) {
	var buf strings.Builder
	fmt.Fprintf(&buf, "%s complete, in %s", span.Operation, span.Duration)
	for _, record := range span.Logs {
		for _, field := range record.Fields {
			fmt.Fprintf(&buf, "\n    %s = %#v", field.Key(), field.Value())
		}
	}
	log.Printf("[TRACE] %s", buf.String())

	if r.outputFile != nil {
		r.mu.Lock()
		defer r.mu.Unlock()
		var jsonSpan traceSpan
		jsonSpan.ID = strconv.FormatUint(span.Context.SpanID, 16)
		jsonSpan.Operation = span.Operation
		jsonSpan.StartTime = span.Start
		jsonSpan.Duration = span.Duration
		if span.ParentSpanID != 0 {
			jsonSpan.ParentID = strconv.FormatUint(span.ParentSpanID, 16)
		}
		for _, record := range span.Logs {
			jsonLog := traceSpanLog{
				Time:   record.Timestamp,
				Values: map[string]interface{}{},
			}
			for _, field := range record.Fields {
				jsonLog.Values[field.Key()] = field.Value()
			}
			jsonSpan.Log = append(jsonSpan.Log, jsonLog)
		}
		r.spans = append(r.spans, jsonSpan)
	}
}

func (r *traceLogRecorder) HandleEvent(evt basictracer.SpanEvent) {
	switch evt := evt.(type) {
	case basictracer.EventCreate:
		log.Printf("[TRACE] %s starting", evt.OperationName)
	}
}

type traceSpan struct {
	ID        string         `json:"id"`
	Operation string         `json:"operation"`
	StartTime time.Time      `json:"startTime"`
	Duration  time.Duration  `json:"duration"`
	ParentID  string         `json:"parentId,omitempty"`
	Log       []traceSpanLog `json:"log,omitempty"`
}

type traceSpanLog struct {
	Time   time.Time              `json:"time"`
	Values map[string]interface{} `json:"values"`
}
