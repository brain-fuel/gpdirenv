// Ported from direnv v2.37.1 internal/cmd (MIT, (c) 2019 zimbatm and
// contributors). Effectful/runtime-boundary glue kept in plain Go; the pure
// env/shell semantic cores live in the goforge.dev/gpdirenv/{env,shell}
// packages. Faithful vendor + import surgery — behavior preserved verbatim.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"goforge.dev/gpdirenv/gzenv"
)

// CmdShowDump is `direnv show_dump`
var CmdShowDump = &Cmd{
	Name:    "show_dump",
	Desc:    "Show the data inside of a dump for debugging purposes",
	Args:    []string{"DUMP"},
	Private: true,
	Action:  actionSimple(cmdShowDumpAction),
}

func cmdShowDumpAction(_ Env, args []string) (err error) {
	if len(args) < 2 {
		return fmt.Errorf("missing DUMP argument")
	}

	var f interface{}
	err = gzenv.Unmarshal(args[1], &f)
	if err != nil {
		return err
	}

	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "  ")
	return e.Encode(f)
}
