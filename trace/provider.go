package trace

import (
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// TracerProvider mirrors OpenTelemetry::Trace::TracerProvider — the factory that
// vends named, versioned tracers. The SDK provides a configurable
// implementation; the no-op provider below is the API default returned by
// OpenTelemetry.tracer_provider before the SDK is configured.
type TracerProvider interface {
	// Tracer mirrors OpenTelemetry.tracer_provider.tracer(name, version).
	Tracer(name, version string) *Tracer
}

// noopTracerProvider is the API-level provider whose spans never record,
// mirroring opentelemetry-ruby's default proxy provider.
type noopTracerProvider struct {
	tp oteltrace.TracerProvider
}

// NoopTracerProvider returns the default, non-recording TracerProvider.
func NoopTracerProvider() TracerProvider {
	return noopTracerProvider{tp: noop.NewTracerProvider()}
}

// Tracer returns a non-recording tracer for the given instrumentation scope.
func (p noopTracerProvider) Tracer(name, version string) *Tracer {
	return NewTracer(p.tp.Tracer(name, oteltrace.WithInstrumentationVersion(version)))
}
