package sri_test

import (
	"bytes"
	"testing"
	"testing/quick"

	upstream "github.com/direnv/direnv/v2/pkg/sri"
	forge "goforge.dev/gpdirenv/sri"
)

// The SRI reimplementation must be byte-for-byte identical to the pinned
// upstream: same digest, same string/hex rendering, same parse behavior on
// both valid and malformed input. Each law below is a differential property
// checked over quick-generated inputs.

// forge.Algo is now a Go+ enum: variants are constructed as struct literals
// (forge.SHA256{}) from plain Go. upstream.Algo remains a string type.
var algos = []struct {
	name string
	f    forge.Algo
	u    upstream.Algo
}{
	{"sha256", forge.SHA256{}, upstream.SHA256},
	{"sha384", forge.SHA384{}, upstream.SHA384},
	{"sha512", forge.SHA512{}, upstream.SHA512},
}

func check(t *testing.T, name string, f any) {
	t.Helper()
	if err := quick.Check(f, &quick.Config{MaxCount: 2000}); err != nil {
		t.Errorf("%s: %v", name, err)
	}
}

// Digest + rendering equal upstream for every algorithm and payload.
func TestWriterSumEqualsUpstream(t *testing.T) {
	for _, a := range algos {
		a := a
		check(t, "Sum/"+a.name, func(data []byte) bool {
			var fb, ub bytes.Buffer
			fw := forge.NewWriter(&fb, a.f)
			uw := upstream.NewWriter(&ub, a.u)
			_, _ = fw.Write(data)
			_, _ = uw.Write(data)
			fh, uh := fw.Sum(), uw.Sum()
			// data must be forwarded verbatim, and the hashes must agree.
			return bytes.Equal(fb.Bytes(), ub.Bytes()) &&
				bytes.Equal(fb.Bytes(), data) &&
				fh.String() == uh.String() &&
				fh.Hex() == uh.Hex()
		})
	}
}

// Round-trip: a hash's String() parses back to an equal hash, matching upstream.
func TestParseRoundTripEqualsUpstream(t *testing.T) {
	for _, a := range algos {
		a := a
		check(t, "Parse/"+a.name, func(data []byte) bool {
			var fb, ub bytes.Buffer
			s := func() string {
				w := forge.NewWriter(&fb, a.f)
				_, _ = w.Write(data)
				return w.Sum().String()
			}()
			us := func() string {
				w := upstream.NewWriter(&ub, a.u)
				_, _ = w.Write(data)
				return w.Sum().String()
			}()
			fh, ferr := forge.Parse(s)
			uh, uerr := upstream.Parse(us)
			if ferr != nil || uerr != nil {
				return false
			}
			return fh.String() == uh.String() && fh.Hex() == uh.Hex()
		})
	}
}

// Parse agrees with upstream on arbitrary (mostly malformed) input: same
// success/failure classification and, on success, same rendering.
func TestParseMalformedEqualsUpstream(t *testing.T) {
	check(t, "Parse/arbitrary", func(s string) bool {
		fh, ferr := forge.Parse(s)
		uh, uerr := upstream.Parse(s)
		if (ferr == nil) != (uerr == nil) {
			return false
		}
		if ferr != nil {
			return true // both failed; error text is not part of the contract
		}
		return fh.String() == uh.String() && fh.Hex() == uh.Hex()
	})
}

// A few fixed, known-answer vectors (guards against both sides sharing a bug a
// generator might not surface).
func TestParseKnownVectors(t *testing.T) {
	const known = "sha256-gQ/y+yQqXe5CIPLLDmpRmJH7Z/L4KKbKtO+IlGM7H1A="
	fh, err := forge.Parse(known)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fh.String() != known {
		t.Fatalf("round-trip: got %s want %s", fh.String(), known)
	}
	for _, bad := range []string{"", "sha256", "md5-abc", "sha256-!!!!"} {
		if _, err := forge.Parse(bad); err == nil {
			t.Errorf("Parse(%q) should have failed", bad)
		}
	}
}
