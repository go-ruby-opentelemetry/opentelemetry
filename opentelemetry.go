// Package opentelemetry is the root of a pure-Go (CGO=0), MRI-faithful
// reimplementation of Ruby's opentelemetry-api and opentelemetry-sdk gems,
// focused on distributed tracing.
//
// It mirrors the top-level OpenTelemetry Ruby module — the global tracer
// provider and text-map propagator — while the sub-packages mirror the rest of
// the surface:
//
//   - trace: OpenTelemetry::Trace — tracers, spans, span context, kind, status.
//   - context: OpenTelemetry::Context — the immutable execution context.
//   - baggage: OpenTelemetry::Baggage — propagated key/value pairs.
//   - propagation: OpenTelemetry::Context::Propagation — W3C TraceContext and
//     Baggage inject/extract.
//   - sdk/trace: OpenTelemetry::SDK::Trace — the configurable TracerProvider,
//     span processors (simple + batch), samplers and the in-memory exporter seam.
//
// The tracing model itself is not reimplemented: every type is a Ruby-faithful
// facade over the OpenTelemetry Go SDK (go.opentelemetry.io/otel and
// go.opentelemetry.io/otel/sdk), so the real, well-tested Go implementation does
// the recording, sampling, batching and W3C serialization while callers use the
// method names and semantics of opentelemetry-ruby. Everything runs in-process
// with no network: the exporter is an injectable seam and an InMemorySpanExporter
// is provided for tests and embedding.
package opentelemetry

import (
	"sync"

	"github.com/go-ruby-opentelemetry/opentelemetry/propagation"
	"github.com/go-ruby-opentelemetry/opentelemetry/trace"
)

var (
	mu sync.Mutex

	tracerProvider trace.TracerProvider = trace.NoopTracerProvider()

	// The default global propagator matches opentelemetry-ruby's default:
	// W3C TraceContext plus Baggage.
	propagator propagation.TextMapPropagator = propagation.NewComposite(
		propagation.TraceContext(),
		propagation.Baggage(),
	)
)

// TracerProvider mirrors OpenTelemetry.tracer_provider — the registered global
// provider, defaulting to a no-op provider until the SDK is installed.
func TracerProvider() trace.TracerProvider {
	mu.Lock()
	defer mu.Unlock()
	return tracerProvider
}

// SetTracerProvider mirrors OpenTelemetry.tracer_provider= — it installs the
// global provider (typically an SDK TracerProvider).
func SetTracerProvider(tp trace.TracerProvider) {
	mu.Lock()
	defer mu.Unlock()
	tracerProvider = tp
}

// Tracer mirrors OpenTelemetry.tracer_provider.tracer(name, version) — a
// convenience that fetches a tracer from the global provider.
func Tracer(name, version string) *trace.Tracer {
	return TracerProvider().Tracer(name, version)
}

// Propagation mirrors OpenTelemetry.propagation — the registered global
// text-map propagator.
func Propagation() propagation.TextMapPropagator {
	mu.Lock()
	defer mu.Unlock()
	return propagator
}

// SetPropagation mirrors OpenTelemetry.propagation= — it installs the global
// propagator.
func SetPropagation(p propagation.TextMapPropagator) {
	mu.Lock()
	defer mu.Unlock()
	propagator = p
}
