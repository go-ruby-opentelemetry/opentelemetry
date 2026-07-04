// Package sdktrace mirrors Ruby's OpenTelemetry::SDK::Trace — the configurable
// tracing implementation from the opentelemetry-sdk gem.
//
// It is a Ruby-faithful facade over go.opentelemetry.io/otel/sdk/trace: the
// TracerProvider wires span processors, samplers and exporters exactly as the
// Ruby SDK does, while delegating all recording, sampling and batching to the
// real Go SDK. The exporter is an injectable seam; an in-memory exporter is
// provided for tests and in-process use (OTLP/stdout wiring is a follow-up).
package sdktrace

import (
	gocontext "context"

	rbtrace "github.com/go-ruby-opentelemetry/opentelemetry/trace"
	otelsdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// SpanProcessor mirrors OpenTelemetry::SDK::Trace::SpanProcessor — the hook that
// receives spans as they start and finish. Simple and Batch implementations are
// provided by NewSimpleSpanProcessor and NewBatchSpanProcessor.
type SpanProcessor = otelsdktrace.SpanProcessor

// SpanExporter mirrors OpenTelemetry::SDK::Trace::Export::SpanExporter — the
// injectable sink a batch/simple processor flushes finished spans to.
type SpanExporter = otelsdktrace.SpanExporter

// Sampler mirrors OpenTelemetry::SDK::Trace::Samplers — the per-span decision to
// record and sample.
type Sampler = otelsdktrace.Sampler

// config accumulates TracerProvider options.
type config struct {
	processors []SpanProcessor
	sampler    Sampler
}

// Option configures a TracerProvider, mirroring the keyword configuration of
// OpenTelemetry::SDK.configure.
type Option func(*config)

// WithSpanProcessor registers a span processor (add_span_processor).
func WithSpanProcessor(p SpanProcessor) Option {
	return func(c *config) { c.processors = append(c.processors, p) }
}

// WithSampler sets the sampler (the SDK's :sampler config).
func WithSampler(s Sampler) Option {
	return func(c *config) { c.sampler = s }
}

// TracerProvider mirrors OpenTelemetry::SDK::Trace::TracerProvider.
type TracerProvider struct {
	tp *otelsdktrace.TracerProvider
}

// NewTracerProvider mirrors OpenTelemetry::SDK::Trace::TracerProvider.new: it
// builds a provider from the given processors and sampler.
func NewTracerProvider(opts ...Option) *TracerProvider {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}
	tpOpts := make([]otelsdktrace.TracerProviderOption, 0, len(cfg.processors)+1)
	if cfg.sampler != nil {
		tpOpts = append(tpOpts, otelsdktrace.WithSampler(cfg.sampler))
	}
	for _, p := range cfg.processors {
		tpOpts = append(tpOpts, otelsdktrace.WithSpanProcessor(p))
	}
	return &TracerProvider{tp: otelsdktrace.NewTracerProvider(tpOpts...)}
}

// Tracer mirrors OpenTelemetry::SDK::Trace::TracerProvider#tracer(name, version).
func (p *TracerProvider) Tracer(name, version string) *rbtrace.Tracer {
	return rbtrace.NewTracer(p.tp.Tracer(name, oteltrace.WithInstrumentationVersion(version)))
}

// AddSpanProcessor mirrors #add_span_processor.
func (p *TracerProvider) AddSpanProcessor(sp SpanProcessor) {
	p.tp.RegisterSpanProcessor(sp)
}

// ForceFlush mirrors #force_flush: it flushes all registered processors.
func (p *TracerProvider) ForceFlush() error {
	return p.tp.ForceFlush(gocontext.Background())
}

// Shutdown mirrors #shutdown: it flushes and shuts down all processors.
func (p *TracerProvider) Shutdown() error {
	return p.tp.Shutdown(gocontext.Background())
}

// NewSimpleSpanProcessor mirrors
// OpenTelemetry::SDK::Trace::Export::SimpleSpanProcessor — it exports each span
// synchronously as it finishes.
func NewSimpleSpanProcessor(exporter SpanExporter) SpanProcessor {
	return otelsdktrace.NewSimpleSpanProcessor(exporter)
}

// NewBatchSpanProcessor mirrors
// OpenTelemetry::SDK::Trace::Export::BatchSpanProcessor — it buffers spans and
// exports them in batches.
func NewBatchSpanProcessor(exporter SpanExporter) SpanProcessor {
	return otelsdktrace.NewBatchSpanProcessor(exporter)
}

// AlwaysOn mirrors OpenTelemetry::SDK::Trace::Samplers::ALWAYS_ON.
func AlwaysOn() Sampler {
	return otelsdktrace.AlwaysSample()
}

// AlwaysOff mirrors OpenTelemetry::SDK::Trace::Samplers::ALWAYS_OFF.
func AlwaysOff() Sampler {
	return otelsdktrace.NeverSample()
}

// TraceIDRatioBased mirrors
// OpenTelemetry::SDK::Trace::Samplers.trace_id_ratio_based(ratio).
func TraceIDRatioBased(fraction float64) Sampler {
	return otelsdktrace.TraceIDRatioBased(fraction)
}

// ParentBased mirrors OpenTelemetry::SDK::Trace::Samplers.parent_based(root:):
// it honours the parent's sampling decision and falls back to root for local
// roots.
func ParentBased(root Sampler) Sampler {
	return otelsdktrace.ParentBased(root)
}
