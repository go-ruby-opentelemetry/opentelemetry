package baggage_test

import (
	"testing"

	"github.com/go-ruby-opentelemetry/opentelemetry/baggage"
	rbcontext "github.com/go-ruby-opentelemetry/opentelemetry/context"
)

func TestSetValueAndRoundTrip(t *testing.T) {
	ctx, err := baggage.Set(rbcontext.Background(), "user", "alice")
	if err != nil {
		t.Fatal(err)
	}
	ctx, err = baggage.Set(ctx, "role", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if got := baggage.Value(ctx, "user"); got != "alice" {
		t.Errorf("Value(user) = %q", got)
	}
	if got := baggage.Value(ctx, "missing"); got != "" {
		t.Errorf("missing key should be empty, got %q", got)
	}
	values := baggage.Values(ctx)
	if len(values) != 2 || values["user"] != "alice" || values["role"] != "admin" {
		t.Errorf("Values = %v", values)
	}
}

func TestSetIsImmutable(t *testing.T) {
	base := rbcontext.Background()
	derived, err := baggage.Set(base, "k", "v")
	if err != nil {
		t.Fatal(err)
	}
	if baggage.Value(base, "k") != "" {
		t.Error("Set mutated the original context")
	}
	if baggage.Value(derived, "k") != "v" {
		t.Error("Set did not bind on the derived context")
	}
}

func TestRemove(t *testing.T) {
	ctx, _ := baggage.Set(rbcontext.Background(), "a", "1")
	ctx, _ = baggage.Set(ctx, "b", "2")
	ctx = baggage.Remove(ctx, "a")
	if baggage.Value(ctx, "a") != "" {
		t.Error("Remove did not delete the key")
	}
	if baggage.Value(ctx, "b") != "2" {
		t.Error("Remove deleted the wrong key")
	}
}

func TestClear(t *testing.T) {
	ctx, _ := baggage.Set(rbcontext.Background(), "a", "1")
	ctx = baggage.Clear(ctx)
	if len(baggage.Values(ctx)) != 0 {
		t.Error("Clear did not remove all baggage")
	}
}

func TestSetInvalidKey(t *testing.T) {
	// A key containing a space is not a valid W3C baggage token.
	ctx := rbcontext.Background()
	got, err := baggage.Set(ctx, "bad key", "v")
	if err == nil {
		t.Fatal("expected an error for an invalid key")
	}
	if got != ctx {
		t.Error("on error the original context must be returned unchanged")
	}
}
