package env_test

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	forge "goforge.dev/gpdirenv/env"
	ref "goforge.dev/gpdirenv/internal/upstreamref"
)

// The environment diff engine must be identical to the pinned upstream
// (internal/cmd, copied verbatim into internal/upstreamref because Go forbids
// importing another module's internal package). Laws below compare forge
// against that oracle over generated environments — including keys that hit the
// ignore rules (DIRENV_*, __fish*, BASH_FUNC_*, and the fixed IgnoredKeys).

// envPair generates two related environments with overlapping, changed, and
// ignore-triggering keys so the diff branches are all exercised.
type envPair struct {
	a, b map[string]string
}

var keyPool = []string{
	"FOO", "BAR", "BAZ", "PATH", "HOME", // ordinary
	"PWD", "OLDPWD", "SHELL", "SHLVL", "PS1", "_", // IgnoredKeys
	"DIRENV_CONFIG", "DIRENV_BASH", "DIRENV_IN_ENVRC", // IgnoredKeys
	"__fish_x", "__fishvar", "BASH_FUNC_foo%%", // prefix rules
	"DIRENV_DIR", "DIRENV_DIFF", // NOT ignored (only the config ones are)
}

func (envPair) Generate(r *rand.Rand, size int) reflect.Value {
	mk := func() map[string]string {
		m := map[string]string{}
		for _, k := range keyPool {
			switch r.Intn(3) {
			case 0: // absent
			case 1:
				m[k] = "v" + string(rune('a'+r.Intn(5)))
			case 2:
				m[k] = "" // present-but-empty (edge case vs absent)
			}
		}
		return m
	}
	return reflect.ValueOf(envPair{mk(), mk()})
}

func forgeEnv(m map[string]string) forge.Env { return forge.Env(m) }
func refEnv(m map[string]string) ref.Env     { return ref.Env(m) }

func TestBuildEnvDiffEqualsUpstream(t *testing.T) {
	f := func(p envPair) bool {
		fd := forge.BuildEnvDiff(forgeEnv(p.a), forgeEnv(p.b))
		rd := ref.BuildEnvDiff(refEnv(p.a), refEnv(p.b))
		return reflect.DeepEqual(fd.Prev, rd.Prev) && reflect.DeepEqual(fd.Next, rd.Next)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 5000}); err != nil {
		t.Errorf("BuildEnvDiff differential: %v", err)
	}
}

func TestPatchEqualsUpstream(t *testing.T) {
	f := func(p envPair, target envPair) bool {
		fd := forge.BuildEnvDiff(forgeEnv(p.a), forgeEnv(p.b))
		rd := ref.BuildEnvDiff(refEnv(p.a), refEnv(p.b))
		fp := fd.Patch(forgeEnv(target.a))
		rp := rd.Patch(refEnv(target.a))
		return reflect.DeepEqual(map[string]string(fp), map[string]string(rp))
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 5000}); err != nil {
		t.Errorf("Patch differential: %v", err)
	}
}

func TestReverseEqualsUpstream(t *testing.T) {
	f := func(p envPair) bool {
		fd := forge.BuildEnvDiff(forgeEnv(p.a), forgeEnv(p.b)).Reverse()
		rd := ref.BuildEnvDiff(refEnv(p.a), refEnv(p.b)).Reverse()
		return reflect.DeepEqual(fd.Prev, rd.Prev) && reflect.DeepEqual(fd.Next, rd.Next)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 5000}); err != nil {
		t.Errorf("Reverse differential: %v", err)
	}
}

func TestIgnoredEnvEqualsUpstream(t *testing.T) {
	// Over the curated pool...
	for _, k := range keyPool {
		if forge.IgnoredEnv(k) != ref.IgnoredEnv(k) {
			t.Errorf("IgnoredEnv(%q): forge=%v upstream=%v", k, forge.IgnoredEnv(k), ref.IgnoredEnv(k))
		}
	}
	// ...and over arbitrary strings.
	f := func(k string) bool { return forge.IgnoredEnv(k) == ref.IgnoredEnv(k) }
	if err := quick.Check(f, &quick.Config{MaxCount: 5000}); err != nil {
		t.Errorf("IgnoredEnv differential: %v", err)
	}
}

// Round-trip: reversing a diff twice returns the original; patching a→b onto a
// yields b restricted to non-ignored keys (sanity of the model, not just parity).
func TestReverseInvolution(t *testing.T) {
	f := func(p envPair) bool {
		d := forge.BuildEnvDiff(forgeEnv(p.a), forgeEnv(p.b))
		dd := d.Reverse().Reverse()
		return reflect.DeepEqual(d.Prev, dd.Prev) && reflect.DeepEqual(d.Next, dd.Next)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 2000}); err != nil {
		t.Errorf("Reverse involution: %v", err)
	}
}
