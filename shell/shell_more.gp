// Additional host-shell renderers ported from direnv v2.37.1 internal/cmd
// (shell_zsh.go, shell_fish.go, shell_tcsh.go, shell_elvish.go, shell_murex.go,
// shell_pwsh.go, shell_vim.go, shell_gha.go, shell_systemd.go) — MIT, (c) 2019
// zimbatm and contributors.
//
// Each is a faithful port: identical Export/Dump behavior and byte-for-byte
// identical per-shell escaping. The escapers are differentially tested against
// verbatim upstream copies in internal/upstreamref.
package shell

import (
	crand "crypto/rand"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"goforge.dev/gpdirenv/env"
)

// Extra byte landmarks used by the tcsh and pwsh escapers. The rest live in
// shell.gp alongside BashEscape.
const (
	SPACE             = 32
	STAR              = 42
	COLON             = 58
	EQUALS            = 61
	LOWERCASE_Z       = 122
	OPEN_CURLY_BRACE  = 123
	CLOSE_CURLY_BRACE = 125
)

// --- zsh ---

type zsh struct{}

// Zsh adds support for the venerable Z shell. Escaping is shared with bash.
var Zsh env.Shell = zsh{}

const zshHook = `
_direnv_hook() {
  trap -- '' SIGINT
  eval "$("{{.SelfPath}}" export zsh)"
  trap - SIGINT
}
typeset -ag precmd_functions
if (( ! ${precmd_functions[(I)_direnv_hook]} )); then
  precmd_functions=(_direnv_hook $precmd_functions)
fi
typeset -ag chpwd_functions
if (( ! ${chpwd_functions[(I)_direnv_hook]} )); then
  chpwd_functions=(_direnv_hook $chpwd_functions)
fi
`

func (sh zsh) Hook() (string, error) { return zshHook, nil }

func (sh zsh) Export(e env.ShellExport) (string, error) {
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

func (sh zsh) Dump(e env.Env) (string, error) {
	var out string
	for key, value := range e {
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

// --- fish ---

type fish struct{}

// Fish adds support for the fish shell as a host.
var Fish env.Shell = fish{}

const fishHook = `
    function __direnv_export_eval --on-event fish_prompt;
        "{{.SelfPath}}" export fish | source;

        if test "$direnv_fish_mode" != "disable_arrow";
            function __direnv_cd_hook --on-variable PWD;
                if test "$direnv_fish_mode" = "eval_after_arrow";
                    set -g __direnv_export_again 0;
                else;
                    "{{.SelfPath}}" export fish | source;
                end;
            end;
        end;
    end;

    function __direnv_export_eval_2 --on-event fish_preexec;
        if set -q __direnv_export_again;
            set -e __direnv_export_again;
            "{{.SelfPath}}" export fish | source;
            echo;
        end;

        functions --erase __direnv_cd_hook;
    end;
`

func (sh fish) Hook() (string, error) { return fishHook, nil }

func (sh fish) Export(e env.ShellExport) (string, error) {
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

func (sh fish) Dump(e env.Env) (string, error) {
	var out string
	for key, value := range e {
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

func (sh fish) escape(str string) string {
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
	escaped := func(s string) {
		out += "'" + s + "'"
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

// --- tcsh ---

type tcsh struct{}

// Tcsh adds support for the tickle shell.
var Tcsh env.Shell = tcsh{}

func (sh tcsh) Hook() (string, error) {
	return "alias precmd 'eval `{{.SelfPath}} export tcsh`'", nil
}

func (sh tcsh) Export(e env.ShellExport) (string, error) {
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

func (sh tcsh) Dump(e env.Env) (string, error) {
	var out string
	for key, value := range e {
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

func (sh tcsh) escape(str string) string {
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
	escaped := func(s string) {
		out += s
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

// --- elvish ---

type elvish struct{}

// Elvish adds support for the elvish shell.
var Elvish env.Shell = elvish{}

func (sh elvish) Hook() (string, error) {
	return `## hook for direnv
set @edit:before-readline = $@edit:before-readline {
	try {
		var m = [("{{.SelfPath}}" export elvish | from-json)]
		if (> (count $m) 0) {
			set m = (all $m)
			keys $m | each { |k|
				if $m[$k] {
					set-env $k $m[$k]
				} else {
					unset-env $k
				}
			}
		}
	} catch e {
		echo $e
	}
}
`, nil
}

func (sh elvish) Export(e env.ShellExport) (string, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(e)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (sh elvish) Dump(e env.Env) (string, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(e)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// --- murex ---

type murex struct{}

// Murex is the shell implementation for the Murex shell.
var Murex env.Shell = murex{}

const murexHook = `event: onPrompt direnv_hook=before {
	"{{.SelfPath}}" export murex -> set exports
	if { $exports != "" } {
		$exports -> :json: formap key value {
			if { is-null value } then {
				!export "$key"
			} else {
				$value -> export "$key"
			}
		}
	}
}`

func (sh murex) Hook() (string, error) { return murexHook, nil }

func (sh murex) Dump(e env.Env) (string, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(e)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (sh murex) Export(e env.ShellExport) (string, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(e)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// --- pwsh ---

type pwsh struct{}

// Pwsh renders into PowerShell (7.2+) commands.
var Pwsh env.Shell = pwsh{}

func (sh pwsh) Hook() (string, error) {
	const hook = `using namespace System;
using namespace System.Management.Automation;

if ($PSVersionTable.PSVersion.Major -lt 7 -or ($PSVersionTable.PSVersion.Major -eq 7 -and $PSVersionTable.PSVersion.Minor -lt 2)) {
    throw "direnv: PowerShell version $($PSVersionTable.PSVersion) does not meet the minimum required version 7.2!"
}

$hook = [EventHandler[LocationChangedEventArgs]] {
  param([object] $source, [LocationChangedEventArgs] $eventArgs)
  end {
    $export = ({{.SelfPath}} export pwsh) -join [Environment]::NewLine;
    if ($export) {
      Invoke-Expression -Command $export;
    }
  }
};
$currentAction = $ExecutionContext.SessionState.InvokeCommand.LocationChangedAction;
if ($currentAction) {
  $ExecutionContext.SessionState.InvokeCommand.LocationChangedAction = [Delegate]::Combine($currentAction, $hook);
}
else {
  $ExecutionContext.SessionState.InvokeCommand.LocationChangedAction = $hook;
};

`
	return hook, nil
}

func (sh pwsh) Export(e env.ShellExport) (string, error) {
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

func (sh pwsh) Dump(e env.Env) (string, error) {
	var out string
	for key, value := range e {
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

func (sh pwsh) escapeEnvKey(str string) string { return PowerShellEscapeEnvKey(str) }

// PowerShellEscapeEnvKey escapes environment variable keys for PowerShell.
func PowerShellEscapeEnvKey(str string) string {
	if str == "" {
		return "__DiReNv_UnReAcHaBlE__"
	}
	in := []byte(str)
	out := ""
	i := 0
	l := len(in)

	escaped := func(s string) {
		out += s
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

func (sh pwsh) escapeVerbatimEnvKey(str string) string { return PowerShellEscapeVerbatimEnvKey(str) }

// PowerShellEscapeVerbatimEnvKey escapes environment variable keys using
// verbatim strings for PowerShell.
func PowerShellEscapeVerbatimEnvKey(str string) string {
	if str == "" {
		return "__DiReNv_UnReAcHaBlE__"
	}
	in := []byte(str)
	out := ""
	i := 0
	l := len(in)

	escaped := func(s string) {
		out += s
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

func (sh pwsh) escapeVerbatimString(str string) string { return PowerShellEscapeVerbatimString(str) }

// PowerShellEscapeVerbatimString escapes strings using verbatim string literals
// for PowerShell.
func PowerShellEscapeVerbatimString(str string) string {
	if str == "" {
		return ""
	}
	in := []byte(str)
	out := ""
	i := 0
	l := len(in)

	escaped := func(s string) {
		out += s
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

// --- vim ---

type vim struct{}

// Vim adds support for vim. Not really a shell but it's handy.
var Vim env.Shell = vim{}

func (sh vim) Hook() (string, error) {
	return "", errors.New("this feature is not supported. Install the direnv.vim plugin instead")
}

func (sh vim) Export(e env.ShellExport) (string, error) {
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

func (sh vim) Dump(e env.Env) (string, error) {
	var out string
	for key, value := range e {
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

// --- gha (GitHub Actions) ---

type gha struct{}

// GitHubActions renders into the $GITHUB_ENV heredoc format.
var GitHubActions env.Shell = gha{}

var validKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func (sh gha) Hook() (string, error) {
	return "", fmt.Errorf("Hook not implemented for GitHub Actions shell")
}

func (sh gha) Export(e env.ShellExport) (string, error) {
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

func (sh gha) Dump(e env.Env) (string, error) {
	var b strings.Builder
	for key, value := range e {
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

// --- systemd ---

type systemdShell struct{}

// Systemd is not really a shell but renders into a systemd EnvironmentFile.
var Systemd env.Shell = systemdShell{}

func (sh systemdShell) Hook() (string, error) {
	return "", errors.New("this feature is not supported")
}

func (sh systemdShell) Export(e env.ShellExport) (string, error) {
	var out string
	for key, value := range e {
		if value != nil {
			out += sh.export(key, *value)
		}
	}
	return out, nil
}

func (sh systemdShell) Dump(e env.Env) (string, error) {
	var out string
	for key, value := range e {
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
		// The value contains special characters so it needs to be quoted.
		valueWithoutEncapsulation, encapsulatedBySingleQuotes := cutEncapsulated(value, `'`)

		if encapsulatedBySingleQuotes {
			sanitizedValue = `'` + strings.ReplaceAll(valueWithoutEncapsulation, `'`, `\'`) + `'`
		} else {
			valueWithoutEncapsulation, _ := cutEncapsulated(value, `"`)
			sanitizedValue = `"` + strings.ReplaceAll(valueWithoutEncapsulation, `"`, `\"`) + `"`
		}
	}
	// If the value doesn't contain special characters we don't touch it.
	return sanitizedValue
}

func (sh systemdShell) export(key, value string) string {
	return key + "=" + sanitizeValue(value) + "\n"
}
