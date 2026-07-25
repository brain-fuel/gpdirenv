// Ported from direnv v2.37.1 internal/cmd (MIT, (c) 2019 zimbatm and
// contributors). Effectful/runtime-boundary glue kept in plain Go; the pure
// env/shell semantic cores live in the goforge.dev/gpdirenv/{env,shell}
// packages. Faithful vendor + import surgery — behavior preserved verbatim.
package cmd

// nolint
const (
	DIRENV_CONFIG = "DIRENV_CONFIG"
	DIRENV_BASH   = "DIRENV_BASH"
	DIRENV_DEBUG  = "DIRENV_DEBUG"

	DIRENV_DIR     = "DIRENV_DIR"
	DIRENV_FILE    = "DIRENV_FILE"
	DIRENV_WATCHES = "DIRENV_WATCHES"
	DIRENV_DIFF    = "DIRENV_DIFF"

	DIRENV_DUMP_FILE_PATH = "DIRENV_DUMP_FILE_PATH"
)
