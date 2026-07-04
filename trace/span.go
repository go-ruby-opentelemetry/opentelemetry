package trace

import (
	"time"

	"go.opentelemetry.io/otel/trace"
)

// Span mirrors OpenTelemetry::Trace::Span — the handle used to enrich and finish
// a span. It wraps the backing Go span (recording or non-recording alike), so
// the same facade serves both the SDK's real spans and the API's no-op span.
//
// Its mutating methods return the receiver so calls can be chained, echoing how
// the Ruby API returns self.
type Span struct {
	span trace.Span
}

// NewSpan wraps a Go span. It is the seam through which a Tracer surfaces the
// span it started.
func NewSpan(span trace.Span) Span {
	return Span{span: span}
}

// SetAttribute mirrors OpenTelemetry::Trace::Span#set_attribute(key, value).
func (s Span) SetAttribute(key string, value any) Span {
	s.span.SetAttributes(attr(key, value))
	return s
}

// SetAttributes sets several attributes at once (span.add_attributes in Ruby).
func (s Span) SetAttributes(m map[string]any) Span {
	s.span.SetAttributes(attrs(m)...)
	return s
}

// AddEvent mirrors OpenTelemetry::Trace::Span#add_event(name, attributes:).
func (s Span) AddEvent(name string, attributes map[string]any) Span {
	s.span.AddEvent(name, trace.WithAttributes(attrs(attributes)...))
	return s
}

// RecordException mirrors OpenTelemetry::Trace::Span#record_exception(error):
// it records the error as an exception event, with optional extra attributes.
func (s Span) RecordException(err error, attributes map[string]any) Span {
	s.span.RecordError(err, trace.WithAttributes(attrs(attributes)...))
	return s
}

// SetStatus mirrors OpenTelemetry::Trace::Span#status=.
func (s Span) SetStatus(status Status) Span {
	s.span.SetStatus(status.Code, status.Description)
	return s
}

// SetName mirrors OpenTelemetry::Trace::Span#name=.
func (s Span) SetName(name string) Span {
	s.span.SetName(name)
	return s
}

// Finish mirrors OpenTelemetry::Trace::Span#finish. An explicit end timestamp
// may be supplied; when omitted the span ends now.
func (s Span) Finish(end ...time.Time) {
	if len(end) > 0 {
		s.span.End(trace.WithTimestamp(end[0]))
		return
	}
	s.span.End()
}

// Context mirrors OpenTelemetry::Trace::Span#context.
func (s Span) Context() SpanContext {
	return NewSpanContext(s.span.SpanContext())
}

// Recording mirrors OpenTelemetry::Trace::Span#recording?.
func (s Span) Recording() bool {
	return s.span.IsRecording()
}

// OTel returns the underlying Go span, for use with the backing SDK.
func (s Span) OTel() trace.Span {
	return s.span
}
