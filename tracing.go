package main

import (
	"fmt"
	"log"
	"strings"

	basictracer "github.com/opentracing/basictracer-go"
	"github.com/opentracing/opentracing-go"
)

func initTracing() {
	recorder := traceLogRecorder{}

	opts := basictracer.DefaultOptions()
	opts.Recorder = recorder
	opts.NewSpanEventListener = func() func(basictracer.SpanEvent) {
		return recorder.HandleEvent
	}

	tracer := basictracer.NewWithOptions(opts)
	opentracing.SetGlobalTracer(tracer)
}

type traceLogRecorder struct{}

var _ basictracer.SpanRecorder = traceLogRecorder{}

func (r traceLogRecorder) RecordSpan(span basictracer.RawSpan) {
	var buf strings.Builder
	fmt.Fprintf(&buf, "%s complete, in %s", span.Operation, span.Duration)
	for _, record := range span.Logs {
		for _, field := range record.Fields {
			fmt.Fprintf(&buf, "\n    %s = %#v", field.Key(), field.Value())
		}
	}
	log.Printf("[TRACE] %s", buf.String())
}

func (r traceLogRecorder) HandleEvent(evt basictracer.SpanEvent) {
	switch evt := evt.(type) {
	case basictracer.EventCreate:
		log.Printf("[TRACE] %s starting", evt.OperationName)
	}
}
