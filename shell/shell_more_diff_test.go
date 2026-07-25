package shell_test

import (
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"testing"
	"testing/quick"

	"goforge.dev/gpdirenv/env"
	ref "goforge.dev/gpdirenv/internal/upstreamref"
	forge "goforge.dev/gpdirenv/shell"
)

// These suites differentially test each ported shell's Export/Dump against the
// verbatim upstream oracle in internal/upstreamref.
//
// Export/Dump iterate a Go map, so line/statement order is nondeterministic.
// We never compare the raw multi-entry string. Instead:
//   - elvish/murex render whole-map JSON (encoding/json sorts keys) → the whole
//     output is deterministic and compared byte-for-byte.
//   - vim statements are newline-terminated and never contain a raw newline, so
//     we split on "\n", sort, and compare (exercises the real multi-entry Dump).
//   - zsh/fish/tcsh/pwsh/systemd concatenate ";"- or "\n"-terminated statements
//     whose values may themselves contain the terminator, so raw splitting is
//     unsafe. We (a) render each key individually — deterministic, byte-exact —
//     and compare the sorted multiset forge-vs-oracle, and (b) verify the real
//     multi-entry output decomposes into exactly those per-key pieces (a genuine
//     order-independent check of the real Export/Dump).
//   - gha uses a crypto-random delimiter, so its output cannot be byte-exact; it
//     is validated structurally (see TestGHAStructural).

// randValue returns a random string that includes control chars, high bytes,
// single quotes, backslashes, colons and PowerShell/tcsh-significant chars.
func randValue(r *rand.Rand) string {
	n := r.Intn(12)
	b := make([]byte, n)
	for i := range b {
		switch r.Intn(4) {
		case 0:
			b[i] = byte(r.Intn(256)) // any byte, incl. invalid UTF-8
		case 1:
			b[i] = ":"[0] // encourage PATH-style splitting
		default:
			// printable-ish plus the interesting punctuation
			set := "abcXYZ_09 '\\\"*?:=[]{}`~/:;"
			b[i] = set[r.Intn(len(set))]
		}
	}
	return string(b)
}

// randKey returns a random key; sometimes "PATH" to exercise fish/tcsh PATH
// handling, sometimes "" to exercise pwsh's empty-key skip.
func randKey(r *rand.Rand) string {
	switch r.Intn(10) {
	case 0:
		return "PATH"
	case 1:
		return ""
	default:
		return randValue(r)
	}
}

// randExport builds a random map with some nil (unset) values.
func randExport(r *rand.Rand) map[string]*string {
	m := make(map[string]*string)
	for i := 0; i < r.Intn(6); i++ {
		k := randKey(r)
		if r.Intn(4) == 0 {
			m[k] = nil
		} else {
			v := randValue(r)
			m[k] = &v
		}
	}
	return m
}

func randDump(r *rand.Rand) map[string]string {
	m := make(map[string]string)
	for i := 0; i < r.Intn(6); i++ {
		m[randKey(r)] = randValue(r)
	}
	return m
}

func toForgeExport(m map[string]*string) env.ShellExport { return env.ShellExport(m) }
func toRefExport(m map[string]*string) ref.ShellExport   { return ref.ShellExport(m) }

// perKeyForge renders each entry of m individually through the forge shell.
func perKeyForge(t *testing.T, sh env.Shell, m map[string]*string) []string {
	t.Helper()
	var out []string
	for k, v := range m {
		s, err := sh.Export(env.ShellExport{k: v})
		if err != nil {
			t.Fatalf("forge single Export(%q): %v", k, err)
		}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func perKeyRef(t *testing.T, exp func(ref.ShellExport) (string, error), m map[string]*string) []string {
	t.Helper()
	var out []string
	for k, v := range m {
		s, err := exp(ref.ShellExport{k: v})
		if err != nil {
			t.Fatalf("oracle single Export(%q): %v", k, err)
		}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// assertPermutation checks that whole is exactly a concatenation of pieces in
// some order (each piece used once). Longest-first greedy avoids prefix
// ambiguity between statements.
func assertPermutation(t *testing.T, label, whole string, pieces []string) {
	t.Helper()
	idx := make([]int, len(pieces))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return len(pieces[idx[a]]) > len(pieces[idx[b]]) })
	used := make([]bool, len(pieces))
	rem := whole
	for rem != "" {
		matched := false
		for _, i := range idx {
			if !used[i] && pieces[i] != "" && strings.HasPrefix(rem, pieces[i]) {
				rem = rem[len(pieces[i]):]
				used[i] = true
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("%s: could not decompose remaining %q into per-key pieces %q", label, rem, pieces)
		}
	}
	for i, u := range used {
		if !u && pieces[i] != "" {
			t.Fatalf("%s: piece %q not present in whole output %q", label, pieces[i], whole)
		}
	}
}

// diffConcatShell runs the per-key multiset + real-output-decomposition checks
// for a concatenating shell (Export with nils, and Dump).
func diffConcatShell(t *testing.T, name string, sh env.Shell, refExport func(ref.ShellExport) (string, error)) {
	t.Helper()
	r := rand.New(rand.NewSource(1))
	for iter := 0; iter < 4000; iter++ {
		m := randExport(r)
		fk := perKeyForge(t, sh, m)
		rk := perKeyRef(t, refExport, m)
		if !equalStrings(fk, rk) {
			t.Fatalf("%s Export per-key mismatch:\nforge=%q\noracle=%q\ninput=%v", name, fk, rk, keysOf(m))
		}
		whole, err := sh.Export(toForgeExport(m))
		if err != nil {
			t.Fatalf("%s Export: %v", name, err)
		}
		assertPermutation(t, name+" Export", whole, fk)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func keysOf(m map[string]*string) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func TestZshExportDiff(t *testing.T)  { diffConcatShell(t, "zsh", forge.Zsh, ref.Zsh.Export) }
func TestFishExportDiff(t *testing.T) { diffConcatShell(t, "fish", forge.Fish, ref.Fish.Export) }
func TestTcshExportDiff(t *testing.T) { diffConcatShell(t, "tcsh", forge.Tcsh, ref.Tcsh.Export) }
func TestPwshExportDiff(t *testing.T) { diffConcatShell(t, "pwsh", forge.Pwsh, ref.Pwsh.Export) }
func TestSystemdExportDiff(t *testing.T) {
	diffConcatShell(t, "systemd", forge.Systemd, ref.Systemd.Export)
}

// diffConcatShellDump renders each key through Dump individually (deterministic)
// and decomposes the real multi-entry Dump into those pieces. Dump is used for
// per-key rendering — not Export — because pwsh's Dump and Export differ for the
// empty key (Export skips it; Dump emits the __DiReNv_UnReAcHaBlE__ block).
func diffConcatShellDump(t *testing.T, name string, sh env.Shell, refDump func(ref.Env) (string, error)) {
	t.Helper()
	r := rand.New(rand.NewSource(7))
	for iter := 0; iter < 3000; iter++ {
		d := randDump(r)
		var fk, rk []string
		for k, v := range d {
			fs, err := sh.Dump(env.Env{k: v})
			if err != nil {
				t.Fatalf("%s forge single Dump: %v", name, err)
			}
			rs, err := refDump(ref.Env{k: v})
			if err != nil {
				t.Fatalf("%s oracle single Dump: %v", name, err)
			}
			fk = append(fk, fs)
			rk = append(rk, rs)
		}
		sort.Strings(fk)
		sort.Strings(rk)
		if !equalStrings(fk, rk) {
			t.Fatalf("%s Dump per-key mismatch:\nforge=%q\noracle=%q", name, fk, rk)
		}
		whole, err := sh.Dump(env.Env(d))
		if err != nil {
			t.Fatalf("%s Dump: %v", name, err)
		}
		assertPermutation(t, name+" Dump", whole, fk)
		rwhole, err := refDump(ref.Env(d))
		if err != nil {
			t.Fatalf("%s oracle Dump: %v", name, err)
		}
		assertPermutation(t, name+" oracle Dump", rwhole, rk)
	}
}

func TestZshDumpDiff(t *testing.T)  { diffConcatShellDump(t, "zsh", forge.Zsh, ref.Zsh.Dump) }
func TestFishDumpDiff(t *testing.T) { diffConcatShellDump(t, "fish", forge.Fish, ref.Fish.Dump) }
func TestTcshDumpDiff(t *testing.T) { diffConcatShellDump(t, "tcsh", forge.Tcsh, ref.Tcsh.Dump) }
func TestPwshDumpDiff(t *testing.T) { diffConcatShellDump(t, "pwsh", forge.Pwsh, ref.Pwsh.Dump) }
func TestSystemdDumpDiff(t *testing.T) {
	diffConcatShellDump(t, "systemd", forge.Systemd, ref.Systemd.Dump)
}

// elvish and murex render deterministic whole-map JSON: compare byte-for-byte.
func TestElvishJSONDiff(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	for iter := 0; iter < 3000; iter++ {
		m := randExport(r)
		fe, err1 := forge.Elvish.Export(toForgeExport(m))
		re, err2 := ref.Elvish.Export(toRefExport(m))
		if err1 != nil || err2 != nil {
			t.Fatalf("elvish Export err: %v %v", err1, err2)
		}
		if fe != re {
			t.Fatalf("elvish Export mismatch:\nforge=%q\noracle=%q", fe, re)
		}
		d := randDump(r)
		fd, _ := forge.Elvish.Dump(env.Env(d))
		rd, _ := ref.Elvish.Dump(ref.Env(d))
		if fd != rd {
			t.Fatalf("elvish Dump mismatch:\nforge=%q\noracle=%q", fd, rd)
		}
	}
}

func TestMurexJSONDiff(t *testing.T) {
	r := rand.New(rand.NewSource(4))
	for iter := 0; iter < 3000; iter++ {
		m := randExport(r)
		fe, _ := forge.Murex.Export(toForgeExport(m))
		re, _ := ref.Murex.Export(toRefExport(m))
		if fe != re {
			t.Fatalf("murex Export mismatch:\nforge=%q\noracle=%q", fe, re)
		}
		d := randDump(r)
		fd, _ := forge.Murex.Dump(env.Env(d))
		rd, _ := ref.Murex.Dump(ref.Env(d))
		if fd != rd {
			t.Fatalf("murex Dump mismatch:\nforge=%q\noracle=%q", fd, rd)
		}
	}
}

// vim statements are "\n"-terminated and contain no raw newline (escapeValue
// maps "\n" → "\\n"), so we can split the real multi-entry output on "\n" and
// compare sorted lines directly against the oracle.
func TestVimDiff(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	splitSort := func(s string) []string {
		lines := strings.Split(s, "\n")
		sort.Strings(lines)
		return lines
	}
	for iter := 0; iter < 4000; iter++ {
		m := randExport(r)
		fe, _ := forge.Vim.Export(toForgeExport(m))
		re, _ := ref.Vim.Export(toRefExport(m))
		if !equalStrings(splitSort(fe), splitSort(re)) {
			t.Fatalf("vim Export mismatch:\nforge=%q\noracle=%q", fe, re)
		}
		d := randDump(r)
		fd, _ := forge.Vim.Dump(env.Env(d))
		rd, _ := ref.Vim.Dump(ref.Env(d))
		if !equalStrings(splitSort(fd), splitSort(rd)) {
			t.Fatalf("vim Dump mismatch:\nforge=%q\noracle=%q", fd, rd)
		}
	}
}

// gha uses a crypto-random delimiter so its output is not byte-deterministic.
// Validate structurally: each valid key yields a KEY<<delim\nVALUE\ndelim\n
// block with matching delimiters, and invalid keys are skipped.
var ghaValidKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func TestGHAStructural(t *testing.T) {
	one := func(key, value string) {
		v := value
		out, err := forge.GitHubActions.Export(env.ShellExport{key: &v})
		if err != nil {
			t.Fatalf("gha Export(%q): %v", key, err)
		}
		if !ghaValidKey.MatchString(key) {
			if out != "" {
				t.Fatalf("gha should skip invalid key %q, got %q", key, out)
			}
			return
		}
		// Expect: KEY<<DELIM\nVALUE\nDELIM\n
		head := key + "<<"
		if !strings.HasPrefix(out, head) {
			t.Fatalf("gha block missing header for %q: %q", key, out)
		}
		rest := out[len(head):]
		nl := strings.IndexByte(rest, '\n')
		if nl < 0 {
			t.Fatalf("gha block missing delimiter newline: %q", out)
		}
		delim := rest[:nl]
		if !strings.HasPrefix(delim, "ghadelimiter_") {
			t.Fatalf("gha delimiter format: %q", delim)
		}
		want := key + "<<" + delim + "\n" + value + "\n" + delim + "\n"
		if out != want {
			t.Fatalf("gha block mismatch:\n got=%q\nwant=%q", out, want)
		}
	}
	one("FOO", "bar")
	one("A_B_C", "line1\nline2")
	one("_x9", "")
	one("with space", "v")     // invalid → skipped
	one("1leading", "v")       // invalid → skipped
	one("has=eq", "v")         // invalid → skipped
	// nil value is a no-op.
	if out, _ := forge.GitHubActions.Export(env.ShellExport{"FOO": nil}); out != "" {
		t.Fatalf("gha unset should be empty, got %q", out)
	}
}

// --- pwsh public escaper property tests (byte-exact) ---

func pwshEscaperByte(t *testing.T, name string, f, g func(string) string) {
	t.Helper()
	for i := 0; i < 256; i++ {
		s := string([]byte{byte(i)})
		if f(s) != g(s) {
			t.Errorf("%s byte %d: forge=%q oracle=%q", name, i, f(s), g(s))
		}
	}
	if f("") != g("") {
		t.Errorf("%s empty mismatch", name)
	}
	fn := func(s string) bool { return f(s) == g(s) }
	if err := quick.Check(fn, &quick.Config{MaxCount: 20000}); err != nil {
		t.Errorf("%s (string) differential: %v", name, err)
	}
	fb := func(b rawBytes) bool { s := string(b); return f(s) == g(s) }
	if err := quick.Check(fb, &quick.Config{MaxCount: 20000}); err != nil {
		t.Errorf("%s (raw bytes) differential: %v", name, err)
	}
}

func TestPowerShellEscapeEnvKey(t *testing.T) {
	pwshEscaperByte(t, "PowerShellEscapeEnvKey", forge.PowerShellEscapeEnvKey, ref.PowerShellEscapeEnvKey)
}

func TestPowerShellEscapeVerbatimEnvKey(t *testing.T) {
	pwshEscaperByte(t, "PowerShellEscapeVerbatimEnvKey", forge.PowerShellEscapeVerbatimEnvKey, ref.PowerShellEscapeVerbatimEnvKey)
}

func TestPowerShellEscapeVerbatimString(t *testing.T) {
	pwshEscaperByte(t, "PowerShellEscapeVerbatimString", forge.PowerShellEscapeVerbatimString, ref.PowerShellEscapeVerbatimString)
}

// DetectShell must recognise every registered upstream key.
func TestDetectShellRegistry(t *testing.T) {
	for _, name := range []string{"bash", "elvish", "fish", "gha", "gzenv", "json", "murex", "tcsh", "vim", "zsh", "pwsh", "systemd"} {
		if forge.DetectShell(name) == nil {
			t.Errorf("DetectShell(%q) = nil, want registered shell", name)
		}
		if forge.DetectShell("-"+name) == nil {
			t.Errorf("DetectShell(%q) = nil, want registered shell", "-"+name)
		}
	}
	if forge.DetectShell("nope") != nil {
		t.Errorf("DetectShell(nope) should be nil")
	}
}
