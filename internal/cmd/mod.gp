// Ported from direnv v2.37.1 internal/cmd (MIT, (c) 2019 zimbatm and
// contributors). Effectful/runtime-boundary glue kept in plain Go; the pure
// env/shell semantic cores live in the goforge.dev/gpdirenv/{env,shell}
// packages. Faithful vendor + import surgery — behavior preserved verbatim.
package cmd

import (
	"fmt"
	"os"
)

var (
	bashPath string
	stdlib   string
	version  string
)

// Main is the main entrypoint to direnv
func Main(env Env, args []string, modBashPath string, modStdlib string, modVersion string) error {
	// We drop $PWD from caller since it can include symlinks, which will
	// break relative path access when finding .envrc or .env in a parent.
	_ = os.Unsetenv("PWD")

	setupLogging(env)
	bashPath = modBashPath
	stdlib = modStdlib
	version = modVersion

	err := CommandsDispatch(env, args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sdirenv: error %v%s\n", errorColor, err, clearColor)
	}
	return err
}
