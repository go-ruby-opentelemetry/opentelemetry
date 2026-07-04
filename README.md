<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-opentelemetry/brand/main/social/go-ruby-opentelemetry-opentelemetry.png" alt="go-ruby-opentelemetry/opentelemetry" width="720"></p>

# opentelemetry — go-ruby-opentelemetry

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-opentelemetry.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo), MRI-faithful reimplementation of Ruby's
[`opentelemetry-api`](https://github.com/open-telemetry/opentelemetry-ruby/tree/main/api)
and
[`opentelemetry-sdk`](https://github.com/open-telemetry/opentelemetry-ruby/tree/main/sdk)
gems** — distributed **tracing**: tracers, spans, span context, W3C context
propagation, baggage, span processors, samplers and an exporter seam. It mirrors
opentelemetry-ruby's observable surface (`in_span`, `start_span`, `set_attribute`,
`add_event`, `record_exception`, `status=`, `TraceContext` inject/extract) while
delegating the tracing model to the real, well-tested OpenTelemetry Go SDK.

It is the OpenTelemetry backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but is a
**standalone, reusable** module — a sibling of
[go-ruby-json](https://github.com/go-ruby-json/json) and
[go-ruby-set](https://github.com/go-ruby-set/set).

> **Does not reimplement the tracing model.** Every type is a Ruby-faithful facade
> over [`go.opentelemetry.io/otel`](https://pkg.go.dev/go.opentelemetry.io/otel)
> and [`go.opentelemetry.io/otel/sdk`](https://pkg.go.dev/go.opentelemetry.io/otel/sdk):
> the Go SDK does the recording, sampling, batching and W3C
> traceparent/tracestate/baggage serialization; this module re-expresses it with
> opentelemetry-ruby method names and semantics.

> **In-process, zero network.** The span exporter is an injectable seam. An
> **`InMemorySpanExporter`** is provided for tests and embedding; OTLP/stdout
> exporters are a documented follow-up.

## Install

```sh
go get github.com/go-ruby-opentelemetry/opentelemetry
```

## Usage

```go
package main

import (
	"errors"
	"fmt"

	rbcontext "github.com/go-ruby-opentelemetry/opentelemetry/context"
	"github.com/go-ruby-opentelemetry/opentelemetry/propagation"
	sdktrace "github.com/go-ruby-opentelemetry/opentelemetry/sdk/trace"
	"github.com/go-ruby-opentelemetry/opentelemetry/trace"
)

func main() {
	// SDK::Trace::TracerProvider with a simple processor + in-memory exporter.
	exporter := sdktrace.NewInMemorySpanExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)),
		sdktrace.WithSampler(sdktrace.AlwaysOn()),
	)
	tracer := tp.Tracer("my.service", "1.0.0")

	// Tracer#in_span(name, attributes:, kind:) { |span| ... }
	tracer.InSpan(rbcontext.Background(), "handle-request",
		func(span trace.Span, ctx *rbcontext.Context) {
			span.SetAttribute("http.method", "GET")
			span.AddEvent("cache.miss", map[string]any{"key": "u:42"})
			span.RecordException(errors.New("upstream slow"), nil)
			span.SetStatus(trace.StatusError("degraded"))

			// W3C TraceContext inject → extract round-trips the ids + flags.
			carrier := propagation.NewCarrier()
			propagation.TraceContext().Inject(ctx, carrier)
			fmt.Println(carrier["traceparent"])
		},
		trace.WithKind(trace.Server),
	)

	finished := exporter.FinishedSpans()
	fmt.Println(finished[0].Name(), finished[0].Status().Code) // handle-request Error
}
```

## Ruby surface

| Ruby | Go |
| --- | --- |
| `OpenTelemetry.tracer_provider` / `.tracer(name, version)` | `opentelemetry.TracerProvider()` / `.Tracer(name, version)` |
| `OpenTelemetry.propagation` | `opentelemetry.Propagation()` / `SetPropagation` |
| `Trace::Tracer#in_span` / `#start_span` / `#start_root_span` | `trace.Tracer.InSpan` / `.StartSpan` / `.StartRootSpan` |
| `Trace::Span#set_attribute` / `add_event` / `record_exception` / `status=` / `finish` / `context` / `name=` | `trace.Span.SetAttribute` / `AddEvent` / `RecordException` / `SetStatus` / `Finish` / `Context` / `SetName` |
| `Trace::SpanContext` (trace_id/span_id/trace_flags/remote?) | `trace.SpanContext` |
| `Trace::SpanKind` (INTERNAL/SERVER/CLIENT/PRODUCER/CONSUMER) | `trace.Internal/Server/Client/Producer/Consumer` |
| `Trace::Status` (unset/ok/error) | `trace.StatusUnset/StatusOK/StatusError` |
| `Context` (create_key/value/current/attach/detach/with_current) | `context.Context` package |
| `Context::Propagation::TraceContext` / `Baggage` inject/extract | `propagation.TraceContext()` / `.Baggage()` |
| `Baggage` (set_value/value/values/remove_value/clear) | `baggage.Set/Value/Values/Remove/Clear` |
| `SDK::Trace::TracerProvider` + `SpanProcessor` (Simple/Batch) + `SpanExporter` | `sdktrace.NewTracerProvider` / `NewSimpleSpanProcessor` / `NewBatchSpanProcessor` / `SpanExporter` |
| `SDK::Trace::Export::InMemorySpanExporter` | `sdktrace.InMemorySpanExporter` |
| `SDK::Trace::Samplers` (ALWAYS_ON/OFF/parent_based/ratio) | `sdktrace.AlwaysOn/AlwaysOff/ParentBased/TraceIDRatioBased` |

## What it consumes

- [`go.opentelemetry.io/otel/trace`](https://pkg.go.dev/go.opentelemetry.io/otel/trace) — the tracing API model (spans, span context, kind).
- [`go.opentelemetry.io/otel/sdk/trace`](https://pkg.go.dev/go.opentelemetry.io/otel/sdk/trace) — the tracer provider, processors, samplers.
- [`go.opentelemetry.io/otel/propagation`](https://pkg.go.dev/go.opentelemetry.io/otel/propagation) — W3C TraceContext + Baggage propagators.
- [`go.opentelemetry.io/otel/baggage`](https://pkg.go.dev/go.opentelemetry.io/otel/baggage) — the baggage container.
- [`tracetest.InMemoryExporter`](https://pkg.go.dev/go.opentelemetry.io/otel/sdk/trace/tracetest) — backs the in-memory exporter seam.

## Correctness

Verified against the backing Go SDK semantics with the in-memory exporter, fully
in-process (no network):

- spans record the correct **name, attributes, events, status, kind** and
  **parent/child linkage** (child shares the parent's trace_id and carries the
  parent's span_id);
- W3C **TraceContext** inject → extract round-trips `trace_id`, `span_id` and
  `trace_flags`, and marks the extracted context `remote?`;
- **baggage** round-trips through the W3C baggage header;
- **sampling** decisions are honored (`AlwaysOn`, `AlwaysOff`, `ParentBased`,
  ratio-based);
- both the **simple** and **batch** processors flush to the exporter;
- `in_span` records a panic as an exception event, sets the status to error, and
  re-raises.

## Tests & coverage

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

CGO-free, `gofmt` + `go vet` clean, race-clean, and green across the six 64-bit
Go targets (amd64, arm64, riscv64, loong64, ppc64le, **s390x** big-endian) and
three OSes (Linux, macOS, Windows). Span timestamps are the SDK's monotonic UTC
clocks — no timezone assumptions.

## Follow-ups

- **Metrics** (`OpenTelemetry::Metrics` — meters/instruments) — a documented
  follow-up; this module does the trace API + SDK faithfully.
- **OTLP / stdout exporters** wired onto the existing `SpanExporter` seam.
- **rbgo binding** into [go-embedded-ruby](https://github.com/go-embedded-ruby/ruby).
- Full **org conformance** (Hugo landing, MkDocs/mike docs, brand assets).

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-opentelemetry/opentelemetry authors.
