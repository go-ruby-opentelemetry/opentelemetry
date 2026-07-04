// Package trace mirrors Ruby's OpenTelemetry::Trace API — the tracing surface
// (tracers, spans, span context, kind, status) exposed by the
// opentelemetry-api gem.
//
// It is a Ruby-faithful facade over the OpenTelemetry Go tracing model
// (go.opentelemetry.io/otel/trace): the types here wrap the Go API/SDK spans so
// callers use opentelemetry-ruby method names (in_span, start_span,
// set_attribute, add_event, record_exception, status=, finish, context) while
// the real, battle-tested Go SDK does the recording. The tracing model is not
// reimplemented.
package trace

import (
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// TraceID is a 16-byte trace identifier (OpenTelemetry::Trace's trace_id).
type TraceID = oteltrace.TraceID

// SpanID is an 8-byte span identifier (OpenTelemetry::Trace's span_id).
type SpanID = oteltrace.SpanID

// TraceFlags is the W3C trace-flags byte (the sampled bit lives here).
type TraceFlags = oteltrace.TraceFlags

// SpanKind mirrors OpenTelemetry::Trace::SpanKind.
type SpanKind = oteltrace.SpanKind

// SpanKind constants mirror OpenTelemetry::Trace::SpanKind's INTERNAL, SERVER,
// CLIENT, PRODUCER and CONSUMER.
const (
	Internal = oteltrace.SpanKindInternal
	Server   = oteltrace.SpanKindServer
	Client   = oteltrace.SpanKindClient
	Producer = oteltrace.SpanKindProducer
	Consumer = oteltrace.SpanKindConsumer
)

// StatusCode mirrors the OpenTelemetry::Trace::Status codes.
type StatusCode = codes.Code

// Status codes mirror OpenTelemetry::Trace::Status.unset/ok/error.
const (
	Unset = codes.Unset
	Ok    = codes.Ok
	Error = codes.Error
)

// Status mirrors OpenTelemetry::Trace::Status — a code plus an optional,
// error-only human-readable description.
type Status struct {
	Code        StatusCode
	Description string
}

// StatusUnset mirrors OpenTelemetry::Trace::Status.unset.
func StatusUnset() Status { return Status{Code: Unset} }

// StatusOK mirrors OpenTelemetry::Trace::Status.ok.
func StatusOK() Status { return Status{Code: Ok} }

// StatusError mirrors OpenTelemetry::Trace::Status.error(description).
func StatusError(description string) Status {
	return Status{Code: Error, Description: description}
}

// SpanContext mirrors OpenTelemetry::Trace::SpanContext — the immutable,
// serializable identity of a span (trace_id, span_id, trace_flags, tracestate,
// remote?).
type SpanContext struct {
	sc oteltrace.SpanContext
}

// NewSpanContext wraps a Go SpanContext. It is the seam used by propagation to
// surface an extracted remote context.
func NewSpanContext(sc oteltrace.SpanContext) SpanContext {
	return SpanContext{sc: sc}
}

// TraceID returns the 16-byte trace identifier.
func (c SpanContext) TraceID() TraceID { return c.sc.TraceID() }

// SpanID returns the 8-byte span identifier.
func (c SpanContext) SpanID() SpanID { return c.sc.SpanID() }

// HexTraceID returns the lowercase-hex trace_id (the W3C wire form).
func (c SpanContext) HexTraceID() string { return c.sc.TraceID().String() }

// HexSpanID returns the lowercase-hex span_id (the W3C wire form).
func (c SpanContext) HexSpanID() string { return c.sc.SpanID().String() }

// TraceFlags returns the W3C trace-flags byte.
func (c SpanContext) TraceFlags() TraceFlags { return c.sc.TraceFlags() }

// Sampled reports whether the sampled trace-flag is set.
func (c SpanContext) Sampled() bool { return c.sc.IsSampled() }

// Remote mirrors OpenTelemetry::Trace::SpanContext#remote? — whether the
// context was extracted from a remote parent.
func (c SpanContext) Remote() bool { return c.sc.IsRemote() }

// Valid mirrors OpenTelemetry::Trace::SpanContext#valid? — a non-zero trace_id
// and span_id.
func (c SpanContext) Valid() bool { return c.sc.IsValid() }

// TraceState returns the W3C tracestate as its header string form.
func (c SpanContext) TraceState() string { return c.sc.TraceState().String() }

// OTel returns the underlying Go SpanContext, for use with the backing SDK.
func (c SpanContext) OTel() oteltrace.SpanContext { return c.sc }
