// Package propagation mirrors Ruby's OpenTelemetry::Context::Propagation — the
// injection and extraction of cross-process context over carriers such as HTTP
// headers.
//
// It is a Ruby-faithful facade over go.opentelemetry.io/otel/propagation. The
// W3C TraceContext (traceparent/tracestate) and Baggage propagators are the real
// Go implementations; this package only re-expresses them in terms of the
// Ruby-faithful Context and the opentelemetry-ruby inject/extract vocabulary.
package propagation

import (
	rbcontext "github.com/go-ruby-opentelemetry/opentelemetry/context"
	otelprop "go.opentelemetry.io/otel/propagation"
)

// Carrier mirrors a Ruby text-map carrier: a mutable string map that a
// propagator reads from (extract) and writes to (inject).
type Carrier = otelprop.MapCarrier

// NewCarrier returns an empty Carrier ready to be injected into.
func NewCarrier() Carrier {
	return otelprop.MapCarrier{}
}

// CarrierFrom adapts an existing header map into a Carrier for extraction.
func CarrierFrom(headers map[string]string) Carrier {
	return otelprop.MapCarrier(headers)
}

// TextMapPropagator mirrors OpenTelemetry::Context::Propagation::TextMapPropagator.
// It injects the active context into a carrier and extracts a context from one.
type TextMapPropagator interface {
	// Inject mirrors #inject(carrier, context:): it writes the context into carrier.
	Inject(ctx *rbcontext.Context, carrier otelprop.TextMapCarrier)
	// Extract mirrors #extract(carrier, context:): it returns a Context enriched
	// from carrier.
	Extract(ctx *rbcontext.Context, carrier otelprop.TextMapCarrier) *rbcontext.Context
	// Fields mirrors #fields: the carrier keys the propagator sets.
	Fields() []string
	// otel exposes the backing Go propagator so propagators can be composed.
	otel() otelprop.TextMapPropagator
}

type wrapper struct {
	p otelprop.TextMapPropagator
}

func (w wrapper) Inject(ctx *rbcontext.Context, carrier otelprop.TextMapCarrier) {
	w.p.Inject(ctx.Go(), carrier)
}

func (w wrapper) Extract(ctx *rbcontext.Context, carrier otelprop.TextMapCarrier) *rbcontext.Context {
	return rbcontext.FromGo(w.p.Extract(ctx.Go(), carrier))
}

func (w wrapper) Fields() []string {
	return w.p.Fields()
}

func (w wrapper) otel() otelprop.TextMapPropagator {
	return w.p
}

// TraceContext mirrors OpenTelemetry::Trace::Propagation::TraceContext — the
// W3C traceparent/tracestate propagator.
func TraceContext() TextMapPropagator {
	return wrapper{p: otelprop.TraceContext{}}
}

// Baggage mirrors OpenTelemetry::Baggage::Propagation — the W3C baggage
// propagator.
func Baggage() TextMapPropagator {
	return wrapper{p: otelprop.Baggage{}}
}

// NewComposite mirrors
// OpenTelemetry::Context::Propagation::CompositeTextMapPropagator — a single
// propagator that runs each of its members in turn.
func NewComposite(propagators ...TextMapPropagator) TextMapPropagator {
	inner := make([]otelprop.TextMapPropagator, len(propagators))
	for i, p := range propagators {
		inner[i] = p.otel()
	}
	return wrapper{p: otelprop.NewCompositeTextMapPropagator(inner...)}
}
