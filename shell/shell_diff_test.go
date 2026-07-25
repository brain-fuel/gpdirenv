package shell_test

import (
	"encoding/json"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	"goforge.dev/gpdirenv/env"
	forge "goforge.dev/gpdirenv/shell"
	ref "goforge.dev/gpdirenv/internal/upstreamref"
)

// BashEscape must be byte-for-byte identical to the pinned upstream over every
// input, including control characters, single quotes, high bytes, and invalid
// UTF-8. The oracle is the verbatim upstream copy in internal/upstreamref.

func TestBashEscapeEqualsUpstream_String(t *testing.T) {
	f := func(s string) bool { return forge.BashEscape(s) == ref.BashEscape(s) }
	if err := quick.Check(f, &quick.Config{MaxCount: 20000}); err != nil {
		t.Errorf("BashEscape (string) differential: %v", err)
	}
}

// rawBytes generates arbitrary byte slices (incl. invalid UTF-8 / high bytes)
// that plain string generation under-samples.
type rawBytes []byte

func (rawBytes) Generate(r *rand.Rand, size int) reflect.Value {
	n := r.Intn(size%24 + 1)
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(r.Intn(256))
	}
	return reflect.ValueOf(rawBytes(b))
}

func TestBashEscapeEqualsUpstream_RawBytes(t *testing.T) {
	f := func(b rawBytes) bool {
		s := string(b)
		return forge.BashEscape(s) == ref.BashEscape(s)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 20000}); err != nil {
		t.Errorf("BashEscape (raw bytes) differential: %v", err)
	}
}

// Exhaustive single-byte coverage: every byte 0..255 escapes identically.
func TestBashEscapeEverySingleByte(t *testing.T) {
	for i := 0; i < 256; i++ {
		s := string([]byte{byte(i)})
		if forge.BashEscape(s) != ref.BashEscape(s) {
			t.Errorf("byte %d: forge=%q upstream=%q", i, forge.BashEscape(s), ref.BashEscape(s))
		}
	}
	if forge.BashEscape("") != ref.BashEscape("") {
		t.Errorf("empty string mismatch")
	}
}

// The JSON shell is deterministic (Go sorts map keys); its Dump must be valid,
// stable JSON that round-trips back to the same environment.
func TestJSONShellRoundTrip(t *testing.T) {
	f := func(m map[string]string) bool {
		out, err := forge.JSON.Dump(env.Env(m))
		if err != nil {
			return false
		}
		var got map[string]string
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			return false
		}
		if m == nil {
			m = map[string]string{}
		}
		if got == nil {
			got = map[string]string{}
		}
		return reflect.DeepEqual(m, got)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 2000}); err != nil {
		t.Errorf("JSON shell round-trip: %v", err)
	}
}

// The gzenv shell must round-trip an environment through the gzenv token.
func TestGzEnvShellRoundTrip(t *testing.T) {
	f := func(m map[string]string) bool {
		out, err := forge.GzEnv.Dump(env.Env(m))
		if err != nil {
			return false
		}
		got, err := env.LoadEnv(out)
		if err != nil {
			return false
		}
		if m == nil {
			m = map[string]string{}
		}
		return reflect.DeepEqual(map[string]string(got), m)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 2000}); err != nil {
		t.Errorf("gzenv shell round-trip: %v", err)
	}
}
