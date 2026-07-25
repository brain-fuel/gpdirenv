// Package upstreamref is a VERBATIM, PINNED copy of the pure functions from
// direnv v2.37.1 internal/cmd (env_diff.go, shell_bash.go, and the per-shell
// renderers shell_zsh.go, shell_fish.go, shell_tcsh.go, shell_elvish.go,
// shell_murex.go, shell_pwsh.go, shell_vim.go, shell_gha.go, shell_systemd.go)
// — MIT, (c) 2019 zimbatm and contributors.
//
// TEST ORACLE ONLY. It is NOT part of gpdirenv's shipped API and is imported
// only by the differential test suites. It exists because these functions live
// in direnv's `internal/cmd` package, which Go will not let an external module
// import — so to differentially test gpdirenv's Go+ reimplementation against
// the *actual upstream behavior*, we pin an exact copy here. Do not edit except
// to re-sync with a newer pinned upstream (and bump the version in this note).
//
// The one deliberate deviation from a byte-verbatim copy: upstream
// shell_systemd.go's sanitizeValue calls the package-private logDebug (from
// log.go, which is not importable and has no effect on the returned string);
// that debug-only call is dropped here. Behavior of the returned value is
// unchanged.
package upstreamref

import (
	crand "crypto/rand"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// ShellExport mirrors cmd.ShellExport: add (non-nil) or remove (nil) a variable.
type ShellExport map[string]*string

// Env mirrors cmd.Env.
type Env map[string]string

// IgnoredKeys is copied verbatim from cmd.IgnoredKeys.
var IgnoredKeys = map[string]bool{
	"DIRENV_CONFIG":   true,
	"DIRENV_BASH":     true,
	"DIRENV_IN_ENVRC": true,
	"COMP_WORDBREAKS": true,
	"PS1":             true,
	"OLDPWD":          true,
	"PWD":             true,
	"SHELL":           true,
	"SHELLOPTS":       true,
	"SHLVL":           true,
	"_":               true,
}

// EnvDiff mirrors cmd.EnvDiff.
type EnvDiff struct {
	Prev map[string]string `json:"p"`
	Next map[string]string `json:"n"`
}

// NewEnvDiff mirrors cmd.NewEnvDiff.
func NewEnvDiff() *EnvDiff {
	return &EnvDiff{make(map[string]string), make(map[string]string)}
}

// BuildEnvDiff is copied verbatim from cmd.BuildEnvDiff.
func BuildEnvDiff(e1, e2 Env) *EnvDiff {
	diff := NewEnvDiff()

	in := func(key string, e Env) bool {
		_, ok := e[key]
		return ok
	}

	for key := range e1 {
		if IgnoredEnv(key) {
			continue
		}
		if e2[key] != e1[key] || !in(key, e2) {
			diff.Prev[key] = e1[key]
		}
	}

	for key := range e2 {
		if IgnoredEnv(key) {
			continue
		}
		if e2[key] != e1[key] || !in(key, e1) {
			diff.Next[key] = e2[key]
		}
	}

	return diff
}

// Patch is copied verbatim from cmd.EnvDiff.Patch.
func (diff *EnvDiff) Patch(env Env) (newEnv Env) {
	newEnv = make(Env)
	for k, v := range env {
		newEnv[k] = v
	}
	for key := range diff.Prev {
		delete(newEnv, key)
	}
	for key, value := range diff.Next {
		newEnv[key] = value
	}
	return newEnv
}

// Reverse is copied verbatim from cmd.EnvDiff.Reverse.
func (diff *EnvDiff) Reverse() *EnvDiff {
	return &EnvDiff{diff.Next, diff.Prev}
}

// IgnoredEnv is copied verbatim from cmd.IgnoredEnv.
func IgnoredEnv(key string) bool {
	if strings.HasPrefix(key, "__fish") {
		return true
	}
	if strings.HasPrefix(key, "BASH_FUNC_") {
		return true
	}
	_, found := IgnoredKeys[key]
	return found
}

// Byte landmarks, copied verbatim from shell_bash.go.
const (
	ACK               = 6
	TAB               = 9
	LF                = 10
	CR                = 13
	US                = 31
	SPACE             = 32
	AMPERSTAND        = 38
	SINGLE_QUOTE      = 39
	STAR              = 42
	PLUS              = 43
	NINE              = 57
	COLON             = 58
	EQUALS            = 61
	QUESTION          = 63
	UPPERCASE_Z       = 90
	OPEN_BRACKET      = 91
	BACKSLASH         = 92
	CLOSE_BRACKET     = 93
	UNDERSCORE        = 95
	BACKTICK          = 96
	LOWERCASE_Z       = 122
	OPEN_CURLY_BRACE  = 123
	CLOSE_CURLY_BRACE = 125
	TILDE             = 126
	DEL               = 127
)

// BashEscape is copied verbatim from cmd.BashEscape.
func BashEscape(str string) string {
	if str == "" {
		return "''"
	}
	in := []byte(str)
	out := ""
	i := 0
	l := len(in)
	escape := false

	hex := func(char byte) {
		escape = true
		out += fmt.Sprintf("\\x%02x", char)
	}
	backslash := func(char byte) {
		escape = true
		out += string([]byte{BACKSLASH, char})
	}
	escaped := func(str string) {
		escape = true
		out += str
	}
	quoted := func(char byte) {
		escape = true
		out += string([]byte{char})
	}
	literal := func(char byte) {
		out += string([]byte{char})
	}

	for i < l {
		char := in[i]
		switch {
		case char == ACK:
			hex(char)
		case char == TAB:
			escaped(`\t`)
		case char == LF:
			escaped(`\n`)
		case char == CR:
			escaped(`\r`)
		case char <= US:
			hex(char)
		case char <= AMPERSTAND:
			quoted(char)
		case char == SINGLE_QUOTE:
			backslash(char)
		case char <= PLUS:
			quoted(char)
		case char <= NINE:
			literal(char)
		case char <= QUESTION:
			quoted(char)
		case char <= UPPERCASE_Z:
			literal(char)
		case char == OPEN_BRACKET:
			quoted(char)
		case char == BACKSLASH:
			backslash(char)
		case char == UNDERSCORE:
			literal(char)
		case char <= CLOSE_BRACKET:
			quoted(char)
		case char <= BACKTICK:
			quoted(char)
		case char <= TILDE:
			quoted(char)
		case char == DEL:
			hex(char)
		default:
			hex(char)
		}
		i++
	}

	if escape {
		out = "$'" + out + "'"
	}
	return out
}

// ============================================================================
// Per-shell renderers, copied verbatim from direnv v2.37.1 internal/cmd.
// Each shell's Export/Dump/escape logic is reproduced exactly so the Go+ port
// in the shell package can be differentially tested against it.
// ============================================================================

// --- zsh (shell_zsh.go) ---

type zsh struct{}

// Zsh is the oracle instance.
var Zsh = zsh{}

func (sh zsh) Export(e ShellExport) (string, error) {
	var out string
	for key, value := range e {
		if value == nil {
			out += sh.unset(key)
		} else {
			out += sh.export(key, *value)
		}
	}
	return out, nil
}

func (sh zsh) Dump(env Env) (string, error) {
	var out string
	for key, value := range env {
		out += sh.export(key, value)
	}
	return out, nil
}

func (sh zsh) export(key, value string) string {
	return "export " + sh.escape(key) + "=" + sh.escape(value) + ";"
}

func (sh zsh) unset(key string) string {
	return "unset " + sh.escape(key) + ";"
}

func (sh zsh) escape(str string) string { return BashEscape(str) }

// --- fish (shell_fish.go) ---

type fish struct{}

// Fish is the oracle instance.
var Fish = fish{}

func (sh fish) Export(e ShellExport) (string, error) {
	var out string
	for key, value := range e {
		if value == nil {
			out += sh.unset(key)
		} else {
			out += sh.export(key, *value)
		}
	}
	return out, nil
}

func (sh fish) Dump(env Env) (string, error) {
	var out string
	for key, value := range env {
		out += sh.export(key, value)
	}
	return out, nil
}

func (sh fish) export(key, value string) string {
	if key == "PATH" {
		command := "set -x -g PATH"
		for _, path := range strings.Split(value, ":") {
			command += " " + sh.escape(path)
		}
		return command + ";"
	}
	return "set -x -g " + sh.escape(key) + " " + sh.escape(value) + ";"
}

func (sh fish) unset(key string) string {
	return "set -e -g " + sh.escape(key) + ";"
}

// FishEscape is the oracle for fish.escape (unexported upstream).
func (sh fish) escape(str string) string { return FishEscape(str) }

// FishEscape reproduces fish.escape verbatim for byte-exact testing.
func FishEscape(str string) string {
	in := []byte(str)
	out := "'"
	i := 0
	l := len(in)

	hex := func(char byte) {
		out += fmt.Sprintf("'\\X%02x'", char)
	}
	backslash := func(char byte) {
		out += string([]byte{BACKSLASH, char})
	}
	escaped := func(str string) {
		out += "'" + str + "'"
	}
	literal := func(char byte) {
		out += string([]byte{char})
	}

	for i < l {
		char := in[i]
		switch {
		case char == TAB:
			escaped(`\t`)
		case char == LF:
			escaped(`\n`)
		case char == CR:
			escaped(`\r`)
		case char <= US:
			hex(char)
		case char == SINGLE_QUOTE:
			backslash(char)
		case char == BACKSLASH:
			backslash(char)
		case char <= TILDE:
			literal(char)
		case char == DEL:
			hex(char)
		default:
			hex(char)
		}
		i++
	}

	out += "'"

	return out
}

// --- tcsh (shell_tcsh.go) ---

type tcsh struct{}

// Tcsh is the oracle instance.
var Tcsh = tcsh{}

func (sh tcsh) Export(e ShellExport) (string, error) {
	var out string
	for key, value := range e {
		if value == nil {
			out += sh.unset(key)
		} else {
			out += sh.export(key, *value)
		}
	}
	return out, nil
}

func (sh tcsh) Dump(env Env) (string, error) {
	var out string
	for key, value := range env {
		out += sh.export(key, value)
	}
	return out, nil
}

func (sh tcsh) export(key, value string) string {
	if key == "PATH" {
		command := "set path = ("
		for _, path := range strings.Split(value, ":") {
			command += " " + sh.escape(path)
		}
		return command + " );"
	}
	return "setenv " + sh.escape(key) + " " + sh.escape(value) + " ;"
}

func (sh tcsh) unset(key string) string {
	return "unsetenv " + sh.escape(key) + " ;"
}

func (sh tcsh) escape(str string) string { return TcshEscape(str) }

// TcshEscape reproduces tcsh.escape verbatim for byte-exact testing.
func TcshEscape(str string) string {
	if str == "" {
		return "''"
	}
	in := []byte(str)
	out := ""
	i := 0
	l := len(in)

	hex := func(char byte) {
		out += fmt.Sprintf("\\x%02x", char)
	}
	backslash := func(char byte) {
		out += string([]byte{BACKSLASH, char})
	}
	escaped := func(str string) {
		out += str
	}
	quoted := func(char byte) {
		out += `"` + string([]byte{char}) + `"`
	}
	literal := func(char byte) {
		out += string([]byte{char})
	}

	for i < l {
		char := in[i]
		switch {
		case char == ACK:
			hex(char)
		case char == TAB:
			escaped(`\t`)
		case char == LF:
			escaped(`\n`)
		case char == CR:
			escaped(`\r`)
		case char == SPACE:
			backslash(char)
		case char <= US:
			hex(char)
		case char <= AMPERSTAND:
			quoted(char)
		case char == SINGLE_QUOTE:
			backslash(char)
		case char <= PLUS:
			quoted(char)
		case char <= NINE:
			literal(char)
		case char <= QUESTION:
			quoted(char)
		case char <= UPPERCASE_Z:
			literal(char)
		case char == OPEN_BRACKET:
			quoted(char)
		case char == BACKSLASH:
			backslash(char)
		case char == UNDERSCORE:
			literal(char)
		case char <= LOWERCASE_Z:
			literal(char)
		case char <= CLOSE_BRACKET:
			quoted(char)
		case char <= BACKTICK:
			quoted(char)
		case char <= TILDE:
			quoted(char)
		case char == DEL:
			hex(char)
		default:
			hex(char)
		}
		i++
	}

	return out
}

// --- elvish (shell_elvish.go) ---

type elvish struct{}

// Elvish is the oracle instance.
var Elvish = elvish{}

func (sh elvish) Export(e ShellExport) (string, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(e)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (sh elvish) Dump(env Env) (string, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(env)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// --- murex (shell_murex.go) ---

type murex struct{}

// Murex is the oracle instance.
var Murex = murex{}

func (sh murex) Dump(env Env) (string, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(env)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (sh murex) Export(e ShellExport) (string, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(e)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// --- pwsh (shell_pwsh.go) ---

type pwsh struct{}

// Pwsh is the oracle instance.
var Pwsh = pwsh{}

func (sh pwsh) Export(e ShellExport) (string, error) {
	var out string
	for key, value := range e {
		if key != "" {
			if value == nil {
				out += sh.unset(key)
			} else {
				out += sh.export(key, *value)
			}
		}
	}
	return out, nil
}

func (sh pwsh) Dump(env Env) (string, error) {
	var out string
	for key, value := range env {
		out += sh.export(key, value)
	}
	return out, nil
}

func (sh pwsh) export(key, value string) string {
	return fmt.Sprintf("${env:%s}='%s';", sh.escapeEnvKey(key), sh.escapeVerbatimString(value))
}

func (sh pwsh) unset(key string) string {
	return fmt.Sprintf("Remove-Item -LiteralPath 'env:/%s';", sh.escapeVerbatimEnvKey(key))
}

func (pwsh) escapeEnvKey(str string) string { return PowerShellEscapeEnvKey(str) }

// PowerShellEscapeEnvKey escapes environment variable keys for PowerShell.
func PowerShellEscapeEnvKey(str string) string {
	if str == "" {
		return "__DiReNv_UnReAcHaBlE__"
	}
	in := []byte(str)
	out := ""
	i := 0
	l := len(in)

	escaped := func(str string) {
		out += str
	}

	hex := func(char byte) {
		out += fmt.Sprintf("\\x%02x", char)
	}

	literal := func(char byte) {
		out += string([]byte{char})
	}

	for i < l {
		char := in[i]
		switch char {
		case STAR:
			hex(char)
		case COLON:
			hex(char)
		case EQUALS:
			hex(char)
		case QUESTION:
			hex(char)
		case OPEN_BRACKET:
			hex(char)
		case CLOSE_BRACKET:
			hex(char)
		case OPEN_CURLY_BRACE:
			escaped("`{")
		case CLOSE_CURLY_BRACE:
			escaped("`}")
		default:
			literal(char)
		}
		i++
	}

	return out
}

func (pwsh) escapeVerbatimEnvKey(str string) string { return PowerShellEscapeVerbatimEnvKey(str) }

// PowerShellEscapeVerbatimEnvKey escapes environment variable keys using verbatim strings for PowerShell.
func PowerShellEscapeVerbatimEnvKey(str string) string {
	if str == "" {
		return "__DiReNv_UnReAcHaBlE__"
	}
	in := []byte(str)
	out := ""
	i := 0
	l := len(in)

	escaped := func(str string) {
		out += str
	}

	literal := func(char byte) {
		out += string([]byte{char})
	}

	for i < l {
		char := in[i]
		switch char {
		case SINGLE_QUOTE:
			escaped("''")
		default:
			literal(char)
		}
		i++
	}

	return out
}

func (pwsh) escapeVerbatimString(str string) string { return PowerShellEscapeVerbatimString(str) }

// PowerShellEscapeVerbatimString escapes strings using verbatim string literals for PowerShell.
func PowerShellEscapeVerbatimString(str string) string {
	if str == "" {
		return ""
	}
	in := []byte(str)
	out := ""
	i := 0
	l := len(in)

	escaped := func(str string) {
		out += str
	}

	literal := func(char byte) {
		out += string([]byte{char})
	}

	for i < l {
		char := in[i]
		switch char {
		case SINGLE_QUOTE:
			escaped("''")
		default:
			literal(char)
		}
		i++
	}

	return out
}

// --- vim (shell_vim.go) ---

type vim struct{}

// Vim is the oracle instance.
var Vim = vim{}

func (sh vim) Export(e ShellExport) (string, error) {
	var out string
	for key, value := range e {
		if value == nil {
			out += sh.unset(key)
		} else {
			out += sh.export(key, *value)
		}
	}
	return out, nil
}

func (sh vim) Dump(env Env) (string, error) {
	var out string
	for key, value := range env {
		out += sh.export(key, value)
	}
	return out, nil
}

func (sh vim) export(key, value string) string {
	return "call setenv(" + sh.escapeKey(key) + "," + sh.escapeValue(value) + ")\n"
}

func (sh vim) unset(key string) string {
	return "call setenv(" + sh.escapeKey(key) + ",v:null)\n"
}

func (sh vim) escapeKey(str string) string { return sh.escapeValue(str) }

func (sh vim) escapeValue(str string) string {
	replacer := strings.NewReplacer(
		"\n", "\\n",
		"'", "''",
	)
	return "'" + replacer.Replace(str) + "'"
}

// --- gha (shell_gha.go) ---

type gha struct{}

// GitHubActions is the oracle instance.
var GitHubActions = gha{}

var validKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func (sh gha) Export(e ShellExport) (string, error) {
	var b strings.Builder
	for key, value := range e {
		if !validKeyPattern.MatchString(key) {
			fmt.Fprintf(os.Stderr, "direnv: Skipping invalid environment variable key: %s\n", key)
			continue
		}
		if value == nil {
			sh.unset(&b, key)
		} else {
			if err := sh.export(&b, key, *value); err != nil {
				return "", err
			}
		}
	}
	return b.String(), nil
}

func (sh gha) Dump(env Env) (string, error) {
	var b strings.Builder
	for key, value := range env {
		if !validKeyPattern.MatchString(key) {
			fmt.Fprintf(os.Stderr, "direnv: Skipping invalid environment variable key: %s\n", key)
			continue
		}
		if err := sh.export(&b, key, value); err != nil {
			return "", err
		}
	}
	return b.String(), nil
}

func (sh gha) export(b *strings.Builder, key, value string) error {
	delimiter := sh.generateDelimiter()

	if strings.Contains(key, delimiter) || strings.Contains(value, delimiter) {
		fmt.Fprintf(os.Stderr, "direnv: Delimiter collision detected for key %s, regenerating delimiter\n", key)
		delimiter = sh.generateDelimiter()

		if strings.Contains(key, delimiter) || strings.Contains(value, delimiter) {
			return fmt.Errorf("delimiter collision after regeneration for key %s", key)
		}
	}

	b.WriteString(key)
	b.WriteString("<<")
	b.WriteString(delimiter)
	b.WriteByte('\n')
	b.WriteString(value)
	b.WriteByte('\n')
	b.WriteString(delimiter)
	b.WriteByte('\n')
	return nil
}

func (sh gha) unset(_ *strings.Builder, _ string) {
	// Don't do anything. > $GITHUB_ENV will overwrite the existing env.
}

func (sh gha) generateDelimiter() string {
	randomBytes := make([]byte, 16)
	_, err := crand.Read(randomBytes)
	if err != nil {
		return fmt.Sprintf("ghadelimiter_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("ghadelimiter_%x", randomBytes)
}

// --- systemd (shell_systemd.go) ---

type systemdShell struct{}

// Systemd is the oracle instance.
var Systemd = systemdShell{}

func (sh systemdShell) Export(e ShellExport) (string, error) {
	var out string
	for key, value := range e {
		if value != nil {
			out += sh.export(key, *value)
		}
	}
	return out, nil
}

func (sh systemdShell) Dump(env Env) (string, error) {
	var out string
	for key, value := range env {
		out += sh.export(key, value)
	}
	return out, nil
}

func cutEncapsulated(valueToTest, encapsulatingValue string) (cutValue string, wasEncapsulated bool) {
	withoutPrefix, startsWithEncapsulatingValue := strings.CutPrefix(valueToTest, encapsulatingValue)
	if startsWithEncapsulatingValue {
		withoutPrefixAndSuffix, endsWithEncapsulatingValue := strings.CutSuffix(withoutPrefix, encapsulatingValue)
		if endsWithEncapsulatingValue {
			return withoutPrefixAndSuffix, true
		}
	}
	return valueToTest, false
}

func sanitizeValue(value string) string {
	containSpecialChar := false
	specialCharacterList := []string{"\n", "\\", `"`, `'`}
	for _, specialChar := range specialCharacterList {
		if strings.ContainsAny(value, specialChar) {
			containSpecialChar = true
		}
	}

	sanitizedValue := value

	if containSpecialChar {
		// Since the value contains special characters it needs to be quoted
		valueWithoutEncapsulation, encapsulatedBySingleQuotes := cutEncapsulated(value, `'`)

		if encapsulatedBySingleQuotes {
			sanitizedValue = `'` + strings.ReplaceAll(valueWithoutEncapsulation, `'`, `\'`) + `'`
		} else {
			valueWithoutEncapsulation, _ := cutEncapsulated(value, `"`)
			// upstream logs encapsulatedByDoubleQuotes via logDebug here (no effect on result)
			sanitizedValue = `"` + strings.ReplaceAll(valueWithoutEncapsulation, `"`, `\"`) + `"`
		}
	}
	// if the value doesn't contains special characters then we don't touch it
	return sanitizedValue
}

func (sh systemdShell) export(key, value string) string {
	return key + "=" + sanitizeValue(value) + "\n"
}
