package xdg_test

import (
	"testing"
	"testing/quick"

	upstream "github.com/direnv/direnv/v2/xdg"
	forge "goforge.dev/gpdirenv/xdg"
)

// The XDG directory resolution must match the pinned upstream exactly across
// every combination of set/unset XDG_*_HOME and HOME.

// envGen produces env maps that actually exercise the XDG precedence branches
// (upstream's own random string maps would almost never set the right keys).
type envMap map[string]string

func genEnv(seed []string) map[string]string {
	m := map[string]string{}
	keys := []string{"XDG_DATA_HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "HOME"}
	for i, k := range keys {
		if i < len(seed) && seed[i] != "" {
			m[k] = seed[i]
		}
	}
	return m
}

func TestDirsEqualUpstream(t *testing.T) {
	prog := "direnv"
	f := func(seed []string, useProg bool) bool {
		env := genEnv(seed)
		p := prog
		if !useProg {
			p = ""
		}
		return forge.DataDir(env, p) == upstream.DataDir(env, p) &&
			forge.ConfigDir(env, p) == upstream.ConfigDir(env, p) &&
			forge.CacheDir(env, p) == upstream.CacheDir(env, p)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 3000}); err != nil {
		t.Errorf("xdg differential: %v", err)
	}
}

func TestDirsFixedVectors(t *testing.T) {
	cases := []map[string]string{
		{},
		{"HOME": "/home/x"},
		{"XDG_DATA_HOME": "/d", "HOME": "/home/x"},
		{"XDG_CONFIG_HOME": "/c"},
		{"XDG_CACHE_HOME": "/ca", "XDG_DATA_HOME": "/d", "XDG_CONFIG_HOME": "/c", "HOME": "/h"},
	}
	for _, env := range cases {
		if forge.DataDir(env, "direnv") != upstream.DataDir(env, "direnv") ||
			forge.ConfigDir(env, "direnv") != upstream.ConfigDir(env, "direnv") ||
			forge.CacheDir(env, "direnv") != upstream.CacheDir(env, "direnv") {
			t.Errorf("mismatch for env=%v", env)
		}
	}
}
