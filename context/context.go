// Package context mirrors Ruby's OpenTelemetry::Context — the immutable,
// key/value execution context that carries the current span, baggage and any
// user values across API boundaries.
//
// It is a thin, Ruby-faithful facade over Go's standard [context.Context]: a
// [Context] wraps a Go context so that the OpenTelemetry Go SDK
// (go.opentelemetry.io/otel) can be driven underneath while callers use the
// method names and semantics of the opentelemetry-ruby gem.
//
// Like MRI, a Context is immutable: Set returns a new Context and never mutates
// the receiver. The package also reproduces Ruby's process-wide "current"
// context managed through Attach/Detach tokens (OpenTelemetry::Context.current,
// .attach, .detach and .with_current).
package context

import (
	gocontext "context"
	"sync"
)

// Context is the Ruby-faithful OpenTelemetry::Context. It wraps a Go context so
// values, spans and baggage stored by the backing SDK travel with it.
type Context struct {
	ctx gocontext.Context
}

// Background returns the empty root Context (OpenTelemetry::Context::ROOT).
func Background() *Context {
	return &Context{ctx: gocontext.Background()}
}

// FromGo adapts a standard Go context into a Ruby-faithful Context. It is the
// seam through which the backing SDK's context (holding the active span and
// baggage) is surfaced to callers.
func FromGo(ctx gocontext.Context) *Context {
	return &Context{ctx: ctx}
}

// Go returns the underlying Go context, for use with the backing SDK.
func (c *Context) Go() gocontext.Context {
	return c.ctx
}

// Key identifies a value stored in a Context. Distinct Key values never collide
// even when they share a name, matching MRI's Context.create_key: identity is by
// the underlying pointer, not the descriptive name.
type Key struct {
	p *keyName
}

type keyName struct {
	name string
}

// CreateKey mirrors OpenTelemetry::Context.create_key(name).
func CreateKey(name string) Key {
	return Key{p: &keyName{name: name}}
}

// Name returns the descriptive name the Key was created with.
func (k Key) Name() string {
	return k.p.name
}

// Set mirrors OpenTelemetry::Context#set_value: it returns a new Context with
// key bound to value, leaving the receiver unchanged.
func (c *Context) Set(key Key, value any) *Context {
	return &Context{ctx: gocontext.WithValue(c.ctx, key, value)}
}

// Value mirrors OpenTelemetry::Context#value: it returns the value bound to key,
// or nil when the key is absent.
func (c *Context) Value(key Key) any {
	return c.ctx.Value(key)
}

// Token is returned by Attach and consumed by Detach to restore the previous
// current Context (OpenTelemetry::Context.attach's return value).
type Token struct {
	index int
}

var (
	mu    sync.Mutex
	stack []*Context
)

// Current mirrors OpenTelemetry::Context.current. It returns the most recently
// attached Context, or the empty root Context when none is attached.
func Current() *Context {
	mu.Lock()
	defer mu.Unlock()
	if len(stack) == 0 {
		return Background()
	}
	return stack[len(stack)-1]
}

// Attach mirrors OpenTelemetry::Context.attach(context): it makes ctx current
// and returns a Token that Detach uses to restore the prior current context.
func Attach(ctx *Context) *Token {
	mu.Lock()
	defer mu.Unlock()
	tok := &Token{index: len(stack)}
	stack = append(stack, ctx)
	return tok
}

// Detach mirrors OpenTelemetry::Context.detach(token). It pops back to the
// context that was current when tok was produced and reports whether the token
// matched the top of the stack (Ruby logs a warning on mismatch; here the
// mismatch is reported to the caller and the stack is left untouched).
func Detach(tok *Token) bool {
	mu.Lock()
	defer mu.Unlock()
	if tok.index != len(stack)-1 {
		return false
	}
	stack = stack[:tok.index]
	return true
}

// With mirrors OpenTelemetry::Context.with_current(context) { ... }: it makes
// ctx current for the duration of fn and restores the previous current context
// afterwards, even if fn panics.
func With(ctx *Context, fn func(*Context)) {
	tok := Attach(ctx)
	defer Detach(tok)
	fn(ctx)
}

// Clear resets the current-context stack. It exists so tests (and long-lived
// hosts embedding the runtime) can return to a pristine state.
func Clear() {
	mu.Lock()
	defer mu.Unlock()
	stack = nil
}
