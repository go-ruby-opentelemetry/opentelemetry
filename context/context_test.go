package context

import (
	gocontext "context"
	"testing"
)

func TestBackgroundAndGo(t *testing.T) {
	c := Background()
	if c.Go() == nil {
		t.Fatal("Background().Go() is nil")
	}
}

func TestFromGo(t *testing.T) {
	base := gocontext.WithValue(gocontext.Background(), struct{ k int }{1}, "v")
	c := FromGo(base)
	if c.Go() != base {
		t.Fatal("FromGo did not preserve the underlying context")
	}
}

func TestKeyAndValue(t *testing.T) {
	k := CreateKey("user")
	if k.Name() != "user" {
		t.Fatalf("Name = %q", k.Name())
	}
	// Two keys with the same name must not collide.
	k2 := CreateKey("user")
	c := Background().Set(k, 42)
	if got := c.Value(k); got != 42 {
		t.Fatalf("Value(k) = %v", got)
	}
	if got := c.Value(k2); got != nil {
		t.Fatalf("distinct key collided: %v", got)
	}
}

func TestSetIsImmutable(t *testing.T) {
	k := CreateKey("x")
	base := Background()
	derived := base.Set(k, "y")
	if base.Value(k) != nil {
		t.Fatal("Set mutated the receiver")
	}
	if derived.Value(k) != "y" {
		t.Fatal("Set did not bind on the derived context")
	}
}

func TestCurrentAttachDetach(t *testing.T) {
	Clear()
	if Current() != Background() {
		// Current returns a fresh Background each time; compare emptiness instead.
		if Current().Go() == nil {
			t.Fatal("empty Current should still be usable")
		}
	}
	k := CreateKey("k")
	c1 := Background().Set(k, 1)
	tok1 := Attach(c1)
	if Current().Value(k) != 1 {
		t.Fatal("Current did not reflect attached context")
	}
	c2 := Background().Set(k, 2)
	tok2 := Attach(c2)
	if Current().Value(k) != 2 {
		t.Fatal("nested attach not current")
	}
	// Detaching out of order fails and leaves the stack intact.
	if Detach(tok1) {
		t.Fatal("out-of-order Detach should report false")
	}
	if Current().Value(k) != 2 {
		t.Fatal("failed Detach must not alter the stack")
	}
	if !Detach(tok2) {
		t.Fatal("in-order Detach should report true")
	}
	if Current().Value(k) != 1 {
		t.Fatal("Detach did not restore previous context")
	}
	if !Detach(tok1) {
		t.Fatal("Detach of remaining token failed")
	}
	Clear()
}

func TestWith(t *testing.T) {
	Clear()
	k := CreateKey("w")
	c := Background().Set(k, "inside")
	ran := false
	With(c, func(cur *Context) {
		ran = true
		if Current().Value(k) != "inside" {
			t.Fatal("With did not make context current")
		}
	})
	if !ran {
		t.Fatal("With did not run fn")
	}
	if Current().Value(k) != nil {
		t.Fatal("With did not restore the previous context")
	}
}
