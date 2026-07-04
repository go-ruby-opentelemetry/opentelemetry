// Package baggage mirrors Ruby's OpenTelemetry::Baggage — the propagated set of
// key/value pairs that ride alongside a trace.
//
// It is a Ruby-faithful facade over go.opentelemetry.io/otel/baggage: every
// operation takes and returns an immutable Context (matching MRI, where
// OpenTelemetry::Baggage.set_value returns a new context), storing the baggage
// where the backing SDK and W3C baggage propagator expect it.
package baggage

import (
	rbcontext "github.com/go-ruby-opentelemetry/opentelemetry/context"
	otelbaggage "go.opentelemetry.io/otel/baggage"
)

// Set mirrors OpenTelemetry::Baggage.set_value(key, value, context:). It returns
// a new Context with the pair added; an invalid key or value (per the W3C
// baggage grammar) is reported as an error and the context is returned
// unchanged.
func Set(ctx *rbcontext.Context, key, value string) (*rbcontext.Context, error) {
	member, err := otelbaggage.NewMember(key, value)
	if err != nil {
		return ctx, err
	}
	// SetMember only fails for a member carrying no data; the member returned by
	// NewMember above is always valid, so that error is unreachable here.
	bag, _ := otelbaggage.FromContext(ctx.Go()).SetMember(member)
	return rbcontext.FromGo(otelbaggage.ContextWithBaggage(ctx.Go(), bag)), nil
}

// Value mirrors OpenTelemetry::Baggage.value(key, context:). It returns the
// value bound to key, or the empty string when absent.
func Value(ctx *rbcontext.Context, key string) string {
	return otelbaggage.FromContext(ctx.Go()).Member(key).Value()
}

// Values mirrors OpenTelemetry::Baggage.values(context:). It returns all pairs
// as a plain map.
func Values(ctx *rbcontext.Context) map[string]string {
	members := otelbaggage.FromContext(ctx.Go()).Members()
	out := make(map[string]string, len(members))
	for _, m := range members {
		out[m.Key()] = m.Value()
	}
	return out
}

// Remove mirrors OpenTelemetry::Baggage.remove_value(key, context:). It returns
// a new Context with key deleted.
func Remove(ctx *rbcontext.Context, key string) *rbcontext.Context {
	bag := otelbaggage.FromContext(ctx.Go()).DeleteMember(key)
	return rbcontext.FromGo(otelbaggage.ContextWithBaggage(ctx.Go(), bag))
}

// Clear mirrors OpenTelemetry::Baggage.clear(context:). It returns a new Context
// with all baggage removed.
func Clear(ctx *rbcontext.Context) *rbcontext.Context {
	return rbcontext.FromGo(otelbaggage.ContextWithoutBaggage(ctx.Go()))
}
