package sdktrace_test

import (
	"testing"

	rbcontext "github.com/go-ruby-opentelemetry/opentelemetry/context"
	sdktrace "github.com/go-ruby-opentelemetry/opentelemetry/sdk/trace"
	"github.com/go-ruby-opentelemetry/opentelemetry/trace"
)

func TestDefaultProviderRecords(t *testing.T) {
	// No sampler configured -> the SDK default (ParentBased(AlwaysOn)) records
	// local root spans.
	exp := sdktrace.NewInMemorySpanExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)))
	_, span := tp.Tracer("svc", "1").StartSpan(rbcontext.Background(), "s")
	span.Finish()
	if len(exp.FinishedSpans()) != 1 {
		t.Fatal("default provider should record the root span")
	}
	if err := tp.Shutdown(); err != nil {
		t.Fatal(err)
	}
}

func TestAddSpanProcessorAndBatchFlush(t *testing.T) {
	exp := sdktrace.NewInMemorySpanExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysOn()))
	tp.AddSpanProcessor(sdktrace.NewBatchSpanProcessor(exp))

	_, span := tp.Tracer("svc", "1").StartSpan(rbcontext.Background(), "batched")
	span.Finish()
	// A batch processor buffers; nothing is exported until a flush.
	if len(exp.FinishedSpans()) != 0 {
		t.Fatal("batch processor should not export before flush")
	}
	if err := tp.ForceFlush(); err != nil {
		t.Fatal(err)
	}
	if len(exp.FinishedSpans()) != 1 {
		t.Fatal("ForceFlush should export the buffered span")
	}
}

func TestAlwaysOffSampler(t *testing.T) {
	exp := sdktrace.NewInMemorySpanExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)),
		sdktrace.WithSampler(sdktrace.AlwaysOff()),
	)
	_, span := tp.Tracer("svc", "1").StartSpan(rbcontext.Background(), "dropped")
	span.Finish()
	if len(exp.FinishedSpans()) != 0 {
		t.Fatal("AlwaysOff should sample nothing")
	}
}

func TestParentBasedAndRatioSamplers(t *testing.T) {
	for _, sampler := range []sdktrace.Sampler{
		sdktrace.ParentBased(sdktrace.AlwaysOn()),
		sdktrace.TraceIDRatioBased(1.0),
	} {
		exp := sdktrace.NewInMemorySpanExporter()
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)),
			sdktrace.WithSampler(sampler),
		)
		_, span := tp.Tracer("svc", "1").StartSpan(rbcontext.Background(), "kept")
		span.Finish()
		if len(exp.FinishedSpans()) != 1 {
			t.Fatalf("sampler %T should keep the span", sampler)
		}
	}
}

func TestInMemoryReset(t *testing.T) {
	exp := sdktrace.NewInMemorySpanExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)),
		sdktrace.WithSampler(sdktrace.AlwaysOn()),
	)
	_, span := tp.Tracer("svc", "1").StartSpan(rbcontext.Background(), "s")
	span.Finish()
	exp.Reset()
	if len(exp.FinishedSpans()) != 0 {
		t.Fatal("Reset should clear recorded spans")
	}
}

func TestFinishedSpanSnapshot(t *testing.T) {
	exp := sdktrace.NewInMemorySpanExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)),
		sdktrace.WithSampler(sdktrace.AlwaysOn()),
	)
	tr := tp.Tracer("svc", "1")
	pctx, parent := tr.StartSpan(rbcontext.Background(), "parent", trace.WithKind(trace.Client))
	_, child := tr.StartSpan(pctx, "child")
	child.Finish()
	parent.Finish()

	var childSpan, parentSpan sdktrace.FinishedSpan
	for _, s := range exp.FinishedSpans() {
		switch s.Name() {
		case "child":
			childSpan = s
		case "parent":
			parentSpan = s
		}
	}
	if parentSpan.Kind() != trace.Client {
		t.Error("parent kind not recorded")
	}
	if !parentSpan.SpanContext().Valid() {
		t.Error("SpanContext should be valid")
	}
	if !parentSpan.StartTimestamp().Before(parentSpan.EndTimestamp()) &&
		!parentSpan.StartTimestamp().Equal(parentSpan.EndTimestamp()) {
		t.Error("timestamps should be monotonic (start <= end)")
	}
	if !childSpan.ParentSpanContext().Valid() {
		t.Error("child parent context should be valid")
	}
	if childSpan.HexParentSpanID() != parentSpan.SpanContext().HexSpanID() {
		t.Error("child parent-span-id mismatch")
	}
}
