package gzenv_test

import (
	"reflect"
	"testing"
	"testing/quick"

	upstream "github.com/direnv/direnv/v2/gzenv"
	forge "goforge.dev/gpdirenv/gzenv"
)

// The gzenv wire format must be byte-for-byte compatible with upstream so a
// value written by either side is readable by the other. Environments are
// map[string]string, so that is the payload the laws exercise.

func check(t *testing.T, name string, f any) {
	t.Helper()
	if err := quick.Check(f, &quick.Config{MaxCount: 2000}); err != nil {
		t.Errorf("%s: %v", name, err)
	}
}

// Encoding is byte-identical to upstream (same json+zlib+base64 pipeline).
func TestMarshalEqualsUpstream(t *testing.T) {
	check(t, "Marshal", func(env map[string]string) bool {
		return forge.Marshal(env) == upstream.Marshal(env)
	})
}

// Round-trip through the forge recovers the original value.
func TestForgeRoundTrip(t *testing.T) {
	check(t, "forge->forge", func(env map[string]string) bool {
		var got map[string]string
		if err := forge.Unmarshal(forge.Marshal(env), &got); err != nil {
			return false
		}
		return reflect.DeepEqual(normalize(env), normalize(got))
	})
}

// Cross-compatibility: the forge reads what upstream writes and vice-versa.
func TestCrossCompat(t *testing.T) {
	check(t, "upstream->forge", func(env map[string]string) bool {
		var got map[string]string
		if err := forge.Unmarshal(upstream.Marshal(env), &got); err != nil {
			return false
		}
		return reflect.DeepEqual(normalize(env), normalize(got))
	})
	check(t, "forge->upstream", func(env map[string]string) bool {
		var got map[string]string
		if err := upstream.Unmarshal(forge.Marshal(env), &got); err != nil {
			return false
		}
		return reflect.DeepEqual(normalize(env), normalize(got))
	})
}

// Malformed input is rejected the same way by both.
func TestUnmarshalMalformedEqualsUpstream(t *testing.T) {
	check(t, "Unmarshal/arbitrary", func(s string) bool {
		var a, b map[string]string
		return (forge.Unmarshal(s, &a) == nil) == (upstream.Unmarshal(s, &b) == nil)
	})
}

// json round-trips a nil map to an empty map; treat them as equal.
func normalize(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}
