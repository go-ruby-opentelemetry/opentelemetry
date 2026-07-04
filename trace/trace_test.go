package trace_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	rbcontext "github.com/go-ruby-opentelemetry/opentelemetry/context"
	sdktrace "github.com/go-ruby-opentelemetry/opentelemetry/sdk/trace"
	"github.com/go-ruby-opentelemetry/opentelemetry/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// recorder wires a recording SDK tracer to an in-memory exporter so tests can
// observe what the facade produced.
func recorder(t *testing.T) (*trace.Tracer, *sdktrace.InMemorySpanExporter) {
	t.Helper()
	exp := sdktrace.NewInMemorySpanExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)),
		sdktrace.WithSampler(sdktrace.AlwaysOn()),
	)
	return tp.Tracer("test", "1.0.0"), exp
}

func TestSpanContextAccessors(t *testing.T) {
	tid := oteltrace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	sid := oteltrace.SpanID{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18}
	ts, _ := oteltrace.ParseTraceState("vendor=value")
	sc := trace.NewSpanContext(oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: oteltrace.FlagsSampled,
		TraceState: ts,
		Remote:     true,
	}))
	if sc.TraceID() != tid {
		t.Error("TraceID mismatch")
	}
	if sc.SpanID() != sid {
		t.Error("SpanID mismatch")
	}
	if sc.HexTraceID() != "0102030405060708090a0b0c0d0e0f10" {
		t.Errorf("HexTraceID = %s", sc.HexTraceID())
	}
	if sc.HexSpanID() != "1112131415161718" {
		t.Errorf("HexSpanID = %s", sc.HexSpanID())
	}
	if sc.TraceFlags() != oteltrace.FlagsSampled {
		t.Error("TraceFlags mismatch")
	}
	if !sc.Sampled() {
		t.Error("Sampled should be true")
	}
	if !sc.Remote() {
		t.Error("Remote should be true")
	}
	if !sc.Valid() {
		t.Error("Valid should be true")
	}
	if sc.TraceState() != "vendor=value" {
		t.Errorf("TraceState = %s", sc.TraceState())
	}
	if !sc.OTel().IsValid() {
		t.Error("OTel() should return the backing context")
	}
}

func TestStatusConstructors(t *testing.T) {
	if s := trace.StatusUnset(); s.Code != trace.Unset {
		t.Error("StatusUnset")
	}
	if s := trace.StatusOK(); s.Code != trace.Ok {
		t.Error("StatusOK")
	}
	s := trace.StatusError("boom")
	if s.Code != trace.Error || s.Description != "boom" {
		t.Error("StatusError")
	}
}

func TestAttributeCoercions(t *testing.T) {
	tr, exp := recorder(t)
	_, span := tr.StartSpan(rbcontext.Background(), "attrs")
	span.
		SetAttribute("s", "str").
		SetAttribute("b", true).
		SetAttribute("i", 7).
		SetAttribute("i64", int64(9)).
		SetAttribute("f", 3.5).
		SetAttribute("ss", []string{"a", "b"}).
		SetAttribute("bs", []bool{true, false}).
		SetAttribute("is", []int64{1, 2}).
		SetAttribute("fs", []float64{1.5, 2.5}).
		SetAttribute("other", time.Second) // fallback -> String
	span.Finish()

	spans := exp.FinishedSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans", len(spans))
	}
	got := spans[0].Attributes()
	want := map[string]any{
		"s":     "str",
		"b":     true,
		"i":     int64(7),
		"i64":   int64(9),
		"f":     3.5,
		"ss":    []string{"a", "b"},
		"bs":    []bool{true, false},
		"is":    []int64{1, 2},
		"fs":    []float64{1.5, 2.5},
		"other": "1s",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("attributes mismatch:\n got=%#v\nwant=%#v", got, want)
	}
}

func TestSpanEnrichment(t *testing.T) {
	tr, exp := recorder(t)
	_, span := tr.StartSpan(rbcontext.Background(), "orig", trace.WithKind(trace.Server))
	if !span.Recording() {
		t.Error("recording span should report Recording()==true")
	}
	span.SetName("renamed")
	span.SetAttributes(map[string]any{"k": "v"})
	span.AddEvent("event-1", map[string]any{"e": 1})
	span.RecordException(errors.New("kaboom"), map[string]any{"extra": "x"})
	span.SetStatus(trace.StatusError("failed"))
	if span.OTel() == nil {
		t.Error("OTel() nil")
	}
	if !span.Context().Valid() {
		t.Error("span context should be valid")
	}
	span.Finish()

	s := exp.FinishedSpans()[0]
	if s.Name() != "renamed" {
		t.Errorf("name = %s", s.Name())
	}
	if s.Kind() != trace.Server {
		t.Errorf("kind = %v", s.Kind())
	}
	if s.Status().Code != trace.Error || s.Status().Description != "failed" {
		t.Errorf("status = %+v", s.Status())
	}
	names := make([]string, 0)
	for _, ev := range s.Events() {
		names = append(names, ev.Name)
	}
	// The recorded exception event is named "exception" by the SDK.
	if len(names) != 2 || names[0] != "event-1" || names[1] != "exception" {
		t.Errorf("events = %v", names)
	}
}

func TestFinishWithExplicitTime(t *testing.T) {
	tr, exp := recorder(t)
	_, span := tr.StartSpan(rbcontext.Background(), "timed")
	end := time.Now().Add(time.Hour)
	span.Finish(end)
	if !exp.FinishedSpans()[0].EndTimestamp().Equal(end) {
		t.Error("explicit end timestamp not honoured")
	}
}

func TestStartRootSpanIgnoresParent(t *testing.T) {
	tr, exp := recorder(t)
	parentCtx, parent := tr.StartSpan(rbcontext.Background(), "parent")
	// A root span started from within the parent's context must not be a child.
	_, root := tr.StartRootSpan(parentCtx, "root")
	root.Finish()
	parent.Finish()

	spans := exp.FinishedSpans()
	var rootSpan sdktrace.FinishedSpan
	for _, s := range spans {
		if s.Name() == "root" {
			rootSpan = s
		}
	}
	if rootSpan.HexTraceID() == parent.Context().HexTraceID() {
		t.Error("root span should start a new trace, not share the parent's")
	}
	if rootSpan.ParentSpanContext().Valid() {
		t.Error("root span must have no valid parent")
	}
}

func TestParentChildLinkage(t *testing.T) {
	tr, exp := recorder(t)
	pctx, parent := tr.StartSpan(rbcontext.Background(), "parent")
	_, child := tr.StartSpan(pctx, "child")
	child.Finish()
	parent.Finish()

	spans := exp.FinishedSpans()
	var childSpan sdktrace.FinishedSpan
	for _, s := range spans {
		if s.Name() == "child" {
			childSpan = s
		}
	}
	if childSpan.HexTraceID() != parent.Context().HexTraceID() {
		t.Error("child should share the parent's trace id")
	}
	if childSpan.HexParentSpanID() != parent.Context().HexSpanID() {
		t.Error("child parent-span-id should equal the parent's span id")
	}
	if !childSpan.Sampled() {
		t.Error("child should be sampled")
	}
}

func TestInSpan(t *testing.T) {
	tr, exp := recorder(t)
	ran := false
	tr.InSpan(rbcontext.Background(), "work", func(span trace.Span, ctx *rbcontext.Context) {
		ran = true
		// Inside the block the span is the current context.
		if rbcontext.Current().Go() != ctx.Go() {
			t.Error("in_span did not make the span context current")
		}
		span.SetAttribute("did", "work")
	}, trace.WithAttributes(map[string]any{"start": "attr"}))
	if !ran {
		t.Fatal("block did not run")
	}
	if rbcontext.Current().Value(rbcontext.CreateKey("x")) != nil {
		// restored to background
	}
	s := exp.FinishedSpans()[0]
	if s.Attributes()["start"] != "attr" || s.Attributes()["did"] != "work" {
		t.Errorf("attrs = %v", s.Attributes())
	}
}

func TestInSpanRecordsPanic(t *testing.T) {
	tr, exp := recorder(t)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("in_span should re-raise the panic")
		}
		s := exp.FinishedSpans()[0]
		if s.Status().Code != trace.Error {
			t.Errorf("panic should set error status, got %v", s.Status())
		}
		found := false
		for _, ev := range s.Events() {
			if ev.Name == "exception" {
				found = true
			}
		}
		if !found {
			t.Error("panic should be recorded as an exception event")
		}
	}()
	tr.InSpan(rbcontext.Background(), "boom", func(span trace.Span, ctx *rbcontext.Context) {
		panic("explode")
	})
}

func TestNoopProvider(t *testing.T) {
	tp := trace.NoopTracerProvider()
	tr := tp.Tracer("noop", "0.1")
	_, span := tr.StartSpan(rbcontext.Background(), "novel")
	if span.Recording() {
		t.Error("noop span should not be recording")
	}
	span.SetAttribute("k", "v") // must be a no-op, not panic
	span.Finish()
}

func TestAttributesToMapDirect(t *testing.T) {
	if got := trace.AttributesToMap(nil); len(got) != 0 {
		t.Errorf("empty attributes should map to empty, got %v", got)
	}
}
