// Ported from direnv v2.37.1 internal/cmd (MIT, (c) 2019 zimbatm and
// contributors). Effectful/runtime-boundary glue kept in plain Go; the pure
// env/shell semantic cores live in the goforge.dev/gpdirenv/{env,shell}
// packages. Faithful vendor + import surgery — behavior preserved verbatim.
package cmd

import (
	"errors"
	"fmt"
)

// CmdLog is `direnv log [--status | --error] <message>`
var CmdLog = &Cmd{
	Name:   "log",
	Desc:   "Logs a given message",
	Args:   []string{"[--status | --error]", "<message>"},
	Action: actionWithConfig(cmdLog),
}

func cmdLog(_ Env, args []string, c *Config) (err error) {
	if len(args) != 3 {
		return errors.New("invalid arguments")
	}
	logType := args[1]
	message := args[2]
	switch logType {
	case "--status", "-status":
		logStatus(c, message)
	case "--error", "-error":
		logError(c, message)
	default:
		return fmt.Errorf("invalid log-type '%s'", logType)
	}
	return nil
}
