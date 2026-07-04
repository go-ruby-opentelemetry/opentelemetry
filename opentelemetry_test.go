package opentelemetry_test

import (
	"testing"

	otel "github.com/go-ruby-opentelemetry/opentelemetry"
	rbcontext "github.com/go-ruby-opentelemetry/opentelemetry/context"
	"github.com/go-ruby-opentelemetry/opentelemetry/propagation"
	sdktrace "github.com/go-ruby-opentelemetry/opentelemetry/sdk/trace"
	"github.com/go-ruby-opentelemetry/opentelemetry/trace"
)

func TestGlobalTracerProviderDefaultsToNoop(t *testing.T) {
	tp := otel.TracerProvider()
	_, span := tp.Tracer("g", "1").StartSpan(rbcontext.Background(), "s")
	if span.Recording() {
		t.Error("default global provider should be a no-op")
	}
	span.Finish()
}

func TestSetGlobalTracerProvider(t *testing.T) {
	exp := sdktrace.NewInMemorySpanExporter()
	sdk := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)),
		sdktrace.WithSampler(sdktrace.AlwaysOn()),
	)
	otel.SetTracerProvider(sdk)
	defer otel.SetTracerProvider(trace.NoopTracerProvider())
	// Tracer() is the convenience over the global provider.
	_, span := otel.Tracer("g", "1").StartSpan(rbcontext.Background(), "s")
	span.Finish()
	if len(exp.FinishedSpans()) != 1 {
		t.Fatal("global Tracer should route through the installed SDK provider")
	}
}

func TestGlobalPropagation(t *testing.T) {
	def := otel.Propagation()
	fields := def.Fields()
	if len(fields) == 0 {
		t.Fatal("default propagator should expose fields")
	}
	custom := propagation.TraceContext()
	otel.SetPropagation(custom)
	if got := otel.Propagation(); got.Fields()[0] != "traceparent" {
		t.Errorf("SetPropagation not applied: %v", got.Fields())
	}
	otel.SetPropagation(def)
}
