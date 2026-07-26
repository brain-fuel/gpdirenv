package cmd

import (
	"os"
	"testing"
	"time"

	stdconfig "goforge.dev/goplus/std/config"
)

// direnv's allow/deny → std/config.Capability, and the .envrc mtime →
// std/config.Fingerprint. These laws mirror what viper's config file relies on,
// making direnv the second independent consumer of std/config source loading.

func TestRCCapabilityMapsAllowStatus(t *testing.T) {
	rc, _, _ := newTestRC(t, "export X=1\n")
	// fresh → NotAllowed → Denied
	if stdconfig.IsGranted(rcCapability(rc)) {
		t.Fatal("fresh .envrc must be a Denied capability")
	}
	if err := rc.Allow(); err != nil {
		t.Fatal(err)
	}
	// allowed → Granted
	if !stdconfig.IsGranted(rcCapability(rc)) {
		t.Fatal("allowed .envrc must be a Granted capability")
	}
	if err := rc.Deny(); err != nil {
		t.Fatal(err)
	}
	if stdconfig.IsGranted(rcCapability(rc)) {
		t.Fatal("blocked .envrc must be a Denied capability")
	}
}

func TestResolveRCConfigGatedByAllow(t *testing.T) {
	rc, _, _ := newTestRC(t, "export FOO=bar\n")
	env := Env{"FOO": "bar", "NUM": "42"}
	key := stdconfig.NewKey[string](rcConfigSchema, "FOO",
		func(v any) (string, bool) { s, ok := v.(string); return s, ok })

	// Denied (fresh): the .envrc contributes nothing → empty snapshot.
	snap := ResolveRCConfig(rc, env)
	if _, ok := stdconfig.Get(snap, key).(stdconfig.Found[string]); ok {
		t.Fatal("denied .envrc must not contribute config values")
	}

	// Allowed: the .envrc environment resolves into the snapshot.
	if err := rc.Allow(); err != nil {
		t.Fatal(err)
	}
	snap = ResolveRCConfig(rc, env)
	found, ok := stdconfig.Get(snap, key).(stdconfig.Found[string])
	if !ok || found.Value != "bar" {
		t.Fatalf("allowed .envrc must resolve FOO=bar; got %#v", stdconfig.Get(snap, key))
	}
}

func TestRCSourceWatchReload(t *testing.T) {
	rc, _, path := newTestRC(t, "export X=1\n")
	src := rcSource{rc: rc, env: Env{"X": "1"}}

	fp0, err := src.Probe()
	if err != nil {
		t.Fatal(err)
	}
	prev := map[stdconfig.Source]stdconfig.Fingerprint{src.Provenance(): fp0}

	// unchanged → no reload
	changed, now, _ := stdconfig.Reload(prev, src)
	if len(changed) != 0 {
		t.Fatalf("unchanged .envrc must not reload; got %v", changed)
	}

	// bump the mtime → reload of exactly the file source
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
	changed2, _, _ := stdconfig.Reload(now, src)
	if len(changed2) != 1 || !stdconfig.SourceEqual(changed2[0], stdconfig.FileSource{}) {
		t.Fatalf("changed .envrc must reload; got %v", changed2)
	}
}
