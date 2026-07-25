package dotenv_test

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	upstream "github.com/direnv/direnv/v2/pkg/dotenv"
	forge "goforge.dev/gpdirenv/dotenv"
)

// dotenv parsing must agree with the pinned upstream on every input: same
// success/failure classification and, on success, the identical key=value map
// (including value expansion, quoting, and unescaping). A random string almost
// always fails the line regexp, so the generator below builds *structured*
// .env documents that exercise the real value-parsing branches, and we also
// pin the exact upstream test corpus as fixed vectors.

// document is a quick.Generator that renders a plausible .env text.
type document string

const keyHead = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_"
const keyTail = keyHead + "0123456789."
const bare = "ABCabc012_-/.:@" // safe unquoted value chars (no space/#/newline)
const inner = bare + " \t=#"    // richer chars, only legal inside quotes

func pick(r *rand.Rand, s string) byte { return s[r.Intn(len(s))] }

func genKey(r *rand.Rand) string {
	n := 1 + r.Intn(8)
	b := []byte{pick(r, keyHead)}
	for i := 1; i < n; i++ {
		b = append(b, pick(r, keyTail))
	}
	return string(b)
}

func genRun(r *rand.Rand, alpha string, max int) string {
	n := r.Intn(max)
	b := make([]byte, n)
	for i := range b {
		b[i] = pick(r, alpha)
	}
	return string(b)
}

func (document) Generate(r *rand.Rand, size int) reflect.Value {
	var out []byte
	lines := r.Intn(size%12 + 3)
	keys := []string{}
	for i := 0; i < lines; i++ {
		switch r.Intn(10) {
		case 0: // blank line
		case 1: // comment line
			out = append(out, "# "+genRun(r, bare, 10)...)
		default:
			key := genKey(r)
			keys = append(keys, key)
			if r.Intn(3) == 0 {
				out = append(out, "export "...)
			}
			out = append(out, key...)
			// separator variants accepted by the grammar
			switch r.Intn(3) {
			case 0:
				out = append(out, '=')
			case 1:
				out = append(out, " = "...)
			case 2:
				out = append(out, ": "...)
			}
			// value style
			switch r.Intn(6) {
			case 0: // empty
			case 1: // unquoted
				out = append(out, genRun(r, bare, 12)...)
			case 2: // single-quoted (no expansion)
				out = append(out, '\'')
				out = append(out, genRun(r, inner, 12)...)
				out = append(out, '\'')
			case 3: // double-quoted (expansion + escapes)
				out = append(out, '"')
				out = append(out, genRun(r, inner, 8)...)
				out = append(out, `\n`...)
				out = append(out, '"')
			case 4: // reference an earlier key: $KEY
				if len(keys) > 0 {
					out = append(out, '$')
					out = append(out, keys[r.Intn(len(keys))]...)
				} else {
					out = append(out, genRun(r, bare, 6)...)
				}
			case 5: // ${KEY:-default}
				out = append(out, "${"...)
				out = append(out, genKey(r)...)
				out = append(out, ":-"...)
				out = append(out, genRun(r, bare, 6)...)
				out = append(out, '}')
			}
		}
		out = append(out, '\n')
	}
	return reflect.ValueOf(document(out))
}

func TestParseEqualsUpstream(t *testing.T) {
	f := func(d document) bool {
		fenv, ferr := forge.Parse(string(d))
		uenv, uerr := upstream.Parse(string(d))
		if (ferr == nil) != (uerr == nil) {
			return false
		}
		if ferr != nil {
			return true // both rejected the same input
		}
		return reflect.DeepEqual(fenv, uenv)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 5000}); err != nil {
		t.Errorf("Parse differential: %v", err)
	}
}

// Fixed vectors taken verbatim from upstream's own test suite — the canonical
// behaviors a generator might under-sample (YAML-style separator, comment
// override, escapes, defaults).
func TestParseFixedVectors(t *testing.T) {
	vectors := []string{
		"OPTION_A=1\nOPTION_B=2\nOPTION_C= 3\nOPTION_D =4\nOPTION_E = 5\nOPTION_F=\nSMTP_ADDRESS=smtp    # This is a comment\n",
		"OPTION_A='1'\nOPTION_B='2'\nOPTION_C=''\nOPTION_D='\\n'\nOPTION_E=\"1\"\nOPTION_H=\"\\n\"\n#OPTION_I=\"3\"\n",
		"FOO=test\nBAR=$FOO\nBAZ=${FOO}bar\n",
		"FOO=test\nBAR=${NOPE:-default}\n",
		"SOME_VAR=",
		"OPTION_A: 1\nOPTION_B: 2\n",
		"invalid line here that = should = fail",
		"KEY=\"unterminated",
	}
	for _, v := range vectors {
		fenv, ferr := forge.Parse(v)
		uenv, uerr := upstream.Parse(v)
		if (ferr == nil) != (uerr == nil) {
			t.Errorf("error mismatch for %q: forge=%v upstream=%v", v, ferr, uerr)
			continue
		}
		if ferr == nil && !reflect.DeepEqual(fenv, uenv) {
			t.Errorf("map mismatch for %q:\n forge=%v\n upst =%v", v, fenv, uenv)
		}
	}
}
