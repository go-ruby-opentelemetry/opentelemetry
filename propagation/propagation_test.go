package propagation_test

import (
	"strings"
	"testing"

	rbbaggage "github.com/go-ruby-opentelemetry/opentelemetry/baggage"
	rbcontext "github.com/go-ruby-opentelemetry/opentelemetry/context"
	"github.com/go-ruby-opentelemetry/opentelemetry/propagation"
	sdktrace "github.com/go-ruby-opentelemetry/opentelemetry/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func activeSpanContext(t *testing.T) (*rbcontext.Context, oteltrace.SpanContext) {
	t.Helper()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysOn()))
	_, span := tp.Tracer("p", "1").StartSpan(rbcontext.Background(), "s")
	return rbcontext.FromGo(oteltrace.ContextWithSpan(rbcontext.Background().Go(), span.OTel())),
		span.OTel().SpanContext()
}

func TestTraceContextRoundTrip(t *testing.T) {
	ctx, sc := activeSpanContext(t)
	prop := propagation.TraceContext()

	carrier := propagation.NewCarrier()
	prop.Inject(ctx, carrier)

	tp := carrier["traceparent"]
	if !strings.Contains(tp, sc.TraceID().String()) || !strings.Contains(tp, sc.SpanID().String()) {
		t.Fatalf("traceparent %q missing ids", tp)
	}
	if got := prop.Fields(); len(got) == 0 || got[0] != "traceparent" {
		t.Fatalf("Fields = %v", got)
	}

	// Extract into a fresh context; the recovered span context must match and
	// be flagged remote.
	extracted := prop.Extract(rbcontext.Background(), propagation.CarrierFrom(carrier))
	rsc := oteltrace.SpanContextFromContext(extracted.Go())
	if rsc.TraceID() != sc.TraceID() {
		t.Error("trace_id did not round-trip")
	}
	if rsc.SpanID() != sc.SpanID() {
		t.Error("span_id did not round-trip")
	}
	if rsc.TraceFlags() != sc.TraceFlags() {
		t.Error("trace_flags did not round-trip")
	}
	if !rsc.IsRemote() {
		t.Error("extracted context should be remote")
	}
}

func TestBaggageRoundTrip(t *testing.T) {
	ctx, err := rbbaggage.Set(rbcontext.Background(), "team", "otel")
	if err != nil {
		t.Fatal(err)
	}
	prop := propagation.Baggage()
	carrier := propagation.NewCarrier()
	prop.Inject(ctx, carrier)
	if !strings.Contains(carrier["baggage"], "team=otel") {
		t.Fatalf("baggage header = %q", carrier["baggage"])
	}
	extracted := prop.Extract(rbcontext.Background(), propagation.CarrierFrom(carrier))
	if got := rbbaggage.Value(extracted, "team"); got != "otel" {
		t.Fatalf("extracted baggage = %q", got)
	}
}

func TestComposite(t *testing.T) {
	ctx, sc := activeSpanContext(t)
	ctx, err := rbbaggage.Set(ctx, "k", "v")
	if err != nil {
		t.Fatal(err)
	}
	prop := propagation.NewComposite(propagation.TraceContext(), propagation.Baggage())

	fields := prop.Fields()
	if !contains(fields, "traceparent") || !contains(fields, "baggage") {
		t.Fatalf("composite Fields = %v", fields)
	}

	carrier := propagation.NewCarrier()
	prop.Inject(ctx, carrier)
	extracted := prop.Extract(rbcontext.Background(), propagation.CarrierFrom(carrier))
	if oteltrace.SpanContextFromContext(extracted.Go()).TraceID() != sc.TraceID() {
		t.Error("composite did not propagate the trace context")
	}
	if rbbaggage.Value(extracted, "k") != "v" {
		t.Error("composite did not propagate baggage")
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
