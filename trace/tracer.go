package trace

import (
	"fmt"

	rbcontext "github.com/go-ruby-opentelemetry/opentelemetry/context"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Tracer mirrors OpenTelemetry::Trace::Tracer — the factory that starts spans.
// It wraps the backing Go tracer obtained from a TracerProvider.
type Tracer struct {
	tracer oteltrace.Tracer
}

// NewTracer wraps a Go tracer. It is the seam a TracerProvider uses to surface
// the tracer it created.
func NewTracer(tracer oteltrace.Tracer) *Tracer {
	return &Tracer{tracer: tracer}
}

// spanConfig collects the keyword options accepted by in_span/start_span.
type spanConfig struct {
	attributes map[string]any
	kind       SpanKind
	root       bool
}

// SpanOption configures a span being started, mirroring the keyword arguments
// of OpenTelemetry::Trace::Tracer#in_span (attributes:, kind:).
type SpanOption func(*spanConfig)

// WithAttributes sets the span's initial attributes (attributes:).
func WithAttributes(m map[string]any) SpanOption {
	return func(c *spanConfig) { c.attributes = m }
}

// WithKind sets the span's kind (kind:).
func WithKind(kind SpanKind) SpanOption {
	return func(c *spanConfig) { c.kind = kind }
}

func newSpanConfig(opts []SpanOption) spanConfig {
	cfg := spanConfig{kind: Internal}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

func (t *Tracer) start(parent *rbcontext.Context, name string, cfg spanConfig) (*rbcontext.Context, Span) {
	startOpts := []oteltrace.SpanStartOption{oteltrace.WithSpanKind(cfg.kind)}
	if cfg.root {
		startOpts = append(startOpts, oteltrace.WithNewRoot())
	}
	if len(cfg.attributes) > 0 {
		startOpts = append(startOpts, oteltrace.WithAttributes(attrs(cfg.attributes)...))
	}
	goctx, span := t.tracer.Start(parent.Go(), name, startOpts...)
	return rbcontext.FromGo(goctx), NewSpan(span)
}

// StartSpan mirrors OpenTelemetry::Trace::Tracer#start_span(name, with_parent:,
// attributes:, kind:). It starts a span as a child of parent and returns the
// span together with a Context carrying it as the active span. The caller is
// responsible for finishing the span.
func (t *Tracer) StartSpan(parent *rbcontext.Context, name string, opts ...SpanOption) (*rbcontext.Context, Span) {
	return t.start(parent, name, newSpanConfig(opts))
}

// StartRootSpan mirrors OpenTelemetry::Trace::Tracer#start_root_span: it starts
// a span at the root of a new trace, ignoring any active parent in the context.
func (t *Tracer) StartRootSpan(parent *rbcontext.Context, name string, opts ...SpanOption) (*rbcontext.Context, Span) {
	cfg := newSpanConfig(opts)
	cfg.root = true
	return t.start(parent, name, cfg)
}

// InSpan mirrors OpenTelemetry::Trace::Tracer#in_span(name, attributes:, kind:)
// { |span| ... }. It starts a span, makes it the current context for the
// duration of fn, and finishes it afterwards — even if fn panics. A panic is
// recorded on the span as an exception, the status is set to error, and the
// panic is re-raised, matching how the Ruby block form records and re-raises.
func (t *Tracer) InSpan(parent *rbcontext.Context, name string, fn func(span Span, ctx *rbcontext.Context), opts ...SpanOption) {
	ctx, span := t.StartSpan(parent, name, opts...)
	tok := rbcontext.Attach(ctx)
	defer rbcontext.Detach(tok)
	defer span.Finish()
	defer func() {
		if r := recover(); r != nil {
			span.RecordException(fmt.Errorf("%v", r), nil)
			span.SetStatus(StatusError(fmt.Sprint(r)))
			panic(r)
		}
	}()
	fn(span, ctx)
}
