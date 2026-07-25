// Ported-engine unit tests. Hermetic (t.TempDir()), plain Go. These pin the
// runtime-boundary behavior that the pure env/shell cores cannot cover: the
// on-disk allow/deny CAS filenames, the allow/deny/whitelist precedence in
// RC.Allowed, and the FileTimes gzenv round-trip.
package cmd

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// sha256hex mirrors direnv's hashing: hex(sha256(payload)).
func sha256hex(payload string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(payload))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// newTestRC writes an .envrc with the given content in a temp dir and returns
// an RC plus a Config whose DataDir is a separate temp dir.
func newTestRC(t *testing.T, content string) (*RC, *Config, string) {
	t.Helper()
	work := t.TempDir()
	data := t.TempDir()
	rcPath := filepath.Join(work, ".envrc")
	if err := os.WriteFile(rcPath, []byte(content), 0644); err != nil {
		t.Fatalf("write .envrc: %v", err)
	}
	config := &Config{
		Env:            Env{},
		DataDir:        data,
		WhitelistExact: map[string]bool{},
	}
	rc, err := RCFromPath(rcPath, config)
	if err != nil {
		t.Fatalf("RCFromPath: %v", err)
	}
	return rc, config, rcPath
}

// TestAllowHashFilename verifies `allow` writes a file named
// sha256(abspath + "\n" + contents) containing abspath + "\n".
func TestAllowHashFilename(t *testing.T) {
	content := "export MYVAR=hello\n"
	rc, config, rcPath := newTestRC(t, content)

	if err := rc.Allow(); err != nil {
		t.Fatalf("Allow: %v", err)
	}

	abs, _ := filepath.Abs(rcPath)
	wantName := sha256hex(abs + "\n" + content)
	wantFile := filepath.Join(config.AllowDir(), wantName)

	got, err := os.ReadFile(wantFile)
	if err != nil {
		t.Fatalf("expected allow file %q: %v", wantFile, err)
	}
	if string(got) != abs+"\n" {
		t.Errorf("allow file content = %q, want %q", got, abs+"\n")
	}
}

// TestDenyHashFilename verifies `deny` writes a file named
// sha256(abspath + "\n") (contents excluded) containing abspath + "\n".
func TestDenyHashFilename(t *testing.T) {
	rc, config, rcPath := newTestRC(t, "export MYVAR=hello\n")

	if err := rc.Deny(); err != nil {
		t.Fatalf("Deny: %v", err)
	}

	abs, _ := filepath.Abs(rcPath)
	wantName := sha256hex(abs + "\n")
	wantFile := filepath.Join(config.DenyDir(), wantName)

	got, err := os.ReadFile(wantFile)
	if err != nil {
		t.Fatalf("expected deny file %q: %v", wantFile, err)
	}
	if string(got) != abs+"\n" {
		t.Errorf("deny file content = %q, want %q", got, abs+"\n")
	}
}

// TestAllowedPrecedence walks the allow -> deny -> whitelist decision ladder.
func TestAllowedPrecedence(t *testing.T) {
	// Fresh RC is NotAllowed.
	rc, _, _ := newTestRC(t, "export X=1\n")
	if got := rc.Allowed(); got != NotAllowed {
		t.Fatalf("fresh: got %v, want NotAllowed", got)
	}

	// After Allow: Allowed.
	if err := rc.Allow(); err != nil {
		t.Fatalf("Allow: %v", err)
	}
	if got := rc.Allowed(); got != Allowed {
		t.Fatalf("after allow: got %v, want Allowed", got)
	}

	// After Deny: Denied (deny is checked first and Allow's file was removed).
	if err := rc.Deny(); err != nil {
		t.Fatalf("Deny: %v", err)
	}
	if got := rc.Allowed(); got != Denied {
		t.Fatalf("after deny: got %v, want Denied", got)
	}

	// Whitelist (exact) grants Allowed on a fresh, un-denied RC.
	rc2, config2, rcPath2 := newTestRC(t, "export Y=2\n")
	abs2, _ := filepath.Abs(rcPath2)
	config2.WhitelistExact[abs2] = true
	if got := rc2.Allowed(); got != Allowed {
		t.Fatalf("exact whitelist: got %v, want Allowed", got)
	}

	// Whitelist (prefix) also grants Allowed.
	rc3, config3, rcPath3 := newTestRC(t, "export Z=3\n")
	config3.WhitelistPrefix = []string{filepath.Dir(rcPath3)}
	if got := rc3.Allowed(); got != Allowed {
		t.Fatalf("prefix whitelist: got %v, want Allowed", got)
	}

	// Deny beats whitelist: a whitelisted path that is also denied is Denied.
	if err := rc2.Deny(); err != nil {
		t.Fatalf("Deny rc2: %v", err)
	}
	if got := rc2.Allowed(); got != Denied {
		t.Fatalf("denied+whitelisted: got %v, want Denied", got)
	}
}

// TestFileTimesRoundTrip verifies FileTimes survives a gzenv Marshal/Unmarshal.
func TestFileTimesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	if err := os.WriteFile(a, []byte("aa"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("bb"), 0644); err != nil {
		t.Fatal(err)
	}

	orig := NewFileTimes()
	if err := orig.Update(a); err != nil {
		t.Fatalf("Update a: %v", err)
	}
	if err := orig.Update(b); err != nil {
		t.Fatalf("Update b: %v", err)
	}
	// Record a known-missing path too (Exists=false must round-trip).
	if err := orig.Update(filepath.Join(dir, "missing")); err != nil {
		t.Fatalf("Update missing: %v", err)
	}

	marshalled := orig.Marshal()

	restored := NewFileTimes()
	if err := restored.Unmarshal(marshalled); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(*restored.list) != len(*orig.list) {
		t.Fatalf("len = %d, want %d", len(*restored.list), len(*orig.list))
	}
	for i := range *orig.list {
		o := (*orig.list)[i]
		r := (*restored.list)[i]
		if o != r {
			t.Errorf("entry %d: got %+v, want %+v", i, r, o)
		}
	}

	// A round-trip of the restored times must reproduce the same wire string.
	if again := restored.Marshal(); again != marshalled {
		t.Errorf("re-marshal mismatch:\n got %q\nwant %q", again, marshalled)
	}
}
