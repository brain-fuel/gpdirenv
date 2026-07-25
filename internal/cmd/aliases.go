// Ported from direnv v2.37.1 internal/cmd (MIT, (c) 2019 zimbatm and
// contributors). Import-surgery bridge: upstream kept the environment model
// (env.go, env_diff.go) and the shell renderers (shell.go, shell_*.go) in the
// same internal/cmd package as the command code. In this port those pure cores
// were extracted into goforge.dev/gpdirenv/{env,shell}. This file re-exports the
// moved symbols under their original unqualified names so the vendored command
// code compiles nearly unchanged.
package cmd

import (
	"goforge.dev/gpdirenv/env"
	"goforge.dev/gpdirenv/shell"
)

// Moved types — originally declared in internal/cmd/{env.go,env_diff.go,shell.go}.
type (
	Env         = env.Env
	EnvDiff     = env.EnvDiff
	Shell       = env.Shell
	ShellExport = env.ShellExport
)

// Moved constructors/loaders and shell entry points.
var (
	NewEnvDiff   = env.NewEnvDiff
	BuildEnvDiff = env.BuildEnvDiff
	LoadEnv      = env.LoadEnv
	LoadEnvJSON  = env.LoadEnvJSON
	LoadEnvDiff  = env.LoadEnvDiff
	GetEnv       = env.GetEnv
	IgnoredEnv   = env.IgnoredEnv
	IgnoredKeys  = env.IgnoredKeys

	DetectShell = shell.DetectShell
	BashEscape  = shell.BashEscape
	Bash        = shell.Bash // the only shell var referenced directly by command code
)

// supportedShellList backs the `direnv export` help text. Upstream kept this
// map in internal/cmd/shell.go; the shell package's copy is unexported, so the
// full set of rendered shells is mirrored here. Keep in sync with the shell
// package's supportedShellList (all 12 upstream shells are now implemented).
var supportedShellList = map[string]Shell{
	"bash":    shell.Bash,
	"elvish":  shell.Elvish,
	"fish":    shell.Fish,
	"gha":     shell.GitHubActions,
	"gzenv":   shell.GzEnv,
	"json":    shell.JSON,
	"murex":   shell.Murex,
	"tcsh":    shell.Tcsh,
	"vim":     shell.Vim,
	"zsh":     shell.Zsh,
	"pwsh":    shell.Pwsh,
	"systemd": shell.Systemd,
}
