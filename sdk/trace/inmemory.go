package sdktrace

import (
	"time"

	rbtrace "github.com/go-ruby-opentelemetry/opentelemetry/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// InMemorySpanExporter mirrors
// OpenTelemetry::SDK::Trace::Export::InMemorySpanExporter — a SpanExporter that
// keeps finished spans in memory instead of shipping them anywhere. It is the
// injectable exporter seam used for tests and in-process inspection, so the core
// works with zero network.
type InMemorySpanExporter struct {
	*tracetest.InMemoryExporter
}

// NewInMemorySpanExporter returns a fresh in-memory exporter.
func NewInMemorySpanExporter() *InMemorySpanExporter {
	return &InMemorySpanExporter{InMemoryExporter: tracetest.NewInMemoryExporter()}
}

// FinishedSpans mirrors #finished_spans: the spans exported so far, in export
// order, as Ruby-faithful read-only handles.
func (e *InMemorySpanExporter) FinishedSpans() []FinishedSpan {
	stubs := e.GetSpans()
	out := make([]FinishedSpan, len(stubs))
	for i, s := range stubs {
		out[i] = FinishedSpan{stub: s}
	}
	return out
}

// Event mirrors an OpenTelemetry::SDK::Trace::Event recorded on a span.
type Event struct {
	Name       string
	Attributes map[string]any
	Timestamp  time.Time
}

// FinishedSpan mirrors an OpenTelemetry::SDK::Trace::SpanData — the read-only
// snapshot of a finished span as seen by an exporter.
type FinishedSpan struct {
	stub tracetest.SpanStub
}

// Name mirrors SpanData#name.
func (s FinishedSpan) Name() string { return s.stub.Name }

// Kind mirrors SpanData#kind.
func (s FinishedSpan) Kind() rbtrace.SpanKind { return s.stub.SpanKind }

// SpanContext mirrors SpanData#context.
func (s FinishedSpan) SpanContext() rbtrace.SpanContext {
	return rbtrace.NewSpanContext(s.stub.SpanContext)
}

// ParentSpanContext returns the parent's SpanContext (invalid for a root span).
func (s FinishedSpan) ParentSpanContext() rbtrace.SpanContext {
	return rbtrace.NewSpanContext(s.stub.Parent)
}

// HexParentSpanID mirrors SpanData#parent_span_id in W3C hex form.
func (s FinishedSpan) HexParentSpanID() string {
	return s.stub.Parent.SpanID().String()
}

// HexTraceID mirrors SpanData#trace_id in W3C hex form.
func (s FinishedSpan) HexTraceID() string {
	return s.stub.SpanContext.TraceID().String()
}

// Status mirrors SpanData#status.
func (s FinishedSpan) Status() rbtrace.Status {
	return rbtrace.Status{Code: s.stub.Status.Code, Description: s.stub.Status.Description}
}

// Attributes mirrors SpanData#attributes as a Ruby-style hash.
func (s FinishedSpan) Attributes() map[string]any {
	return rbtrace.AttributesToMap(s.stub.Attributes)
}

// Events mirrors SpanData#events.
func (s FinishedSpan) Events() []Event {
	out := make([]Event, len(s.stub.Events))
	for i, ev := range s.stub.Events {
		out[i] = Event{
			Name:       ev.Name,
			Attributes: rbtrace.AttributesToMap(ev.Attributes),
			Timestamp:  ev.Time,
		}
	}
	return out
}

// StartTimestamp mirrors SpanData#start_timestamp (monotonic UTC).
func (s FinishedSpan) StartTimestamp() time.Time { return s.stub.StartTime }

// EndTimestamp mirrors SpanData#end_timestamp (monotonic UTC).
func (s FinishedSpan) EndTimestamp() time.Time { return s.stub.EndTime }

// Sampled reports whether the span was sampled.
func (s FinishedSpan) Sampled() bool { return s.stub.SpanContext.IsSampled() }

// compile-time guard: the exporter must satisfy the SDK SpanExporter interface.
var _ SpanExporter = (*InMemorySpanExporter)(nil)
