// Ported from direnv v2.37.1 internal/cmd (MIT, (c) 2019 zimbatm and
// contributors). Effectful/runtime-boundary glue kept in plain Go; the pure
// env/shell semantic cores live in the goforge.dev/gpdirenv/{env,shell}
// packages. Faithful vendor + import surgery — behavior preserved verbatim.
package cmd

import "strings"

// getStdlib returns the stdlib.sh, with references to direnv replaced.
func getStdlib(config *Config) string {
	return strings.Replace(stdlib, "$(command -v direnv)", config.SelfPath, 1)
}
