package shell

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	ref "goforge.dev/gpdirenv/internal/upstreamref"
)

// fish and tcsh escapers are unexported (matching upstream, which does not
// export them). This internal test reaches them directly and proves they are
// byte-for-byte identical to the verbatim upstream oracle over every byte,
// random strings, and random raw byte slices — the same rigor applied to
// BashEscape.

type rawBytes []byte

func (rawBytes) Generate(r *rand.Rand, size int) reflect.Value {
	n := r.Intn(size%24 + 1)
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(r.Intn(256))
	}
	return reflect.ValueOf(rawBytes(b))
}

func escaperByteExact(t *testing.T, name string, f, g func(string) string) {
	t.Helper()
	for i := 0; i < 256; i++ {
		s := string([]byte{byte(i)})
		if f(s) != g(s) {
			t.Errorf("%s byte %d: forge=%q oracle=%q", name, i, f(s), g(s))
		}
	}
	if f("") != g("") {
		t.Errorf("%s empty mismatch: forge=%q oracle=%q", name, f(""), g(""))
	}
	fs := func(s string) bool { return f(s) == g(s) }
	if err := quick.Check(fs, &quick.Config{MaxCount: 20000}); err != nil {
		t.Errorf("%s (string) differential: %v", name, err)
	}
	fb := func(b rawBytes) bool { s := string(b); return f(s) == g(s) }
	if err := quick.Check(fb, &quick.Config{MaxCount: 20000}); err != nil {
		t.Errorf("%s (raw bytes) differential: %v", name, err)
	}
}

func TestFishEscapeByteExact(t *testing.T) {
	escaperByteExact(t, "fishEscape", func(s string) string { return fish{}.escape(s) }, ref.FishEscape)
}

func TestTcshEscapeByteExact(t *testing.T) {
	escaperByteExact(t, "tcshEscape", func(s string) string { return tcsh{}.escape(s) }, ref.TcshEscape)
}
