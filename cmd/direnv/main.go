// Ported from direnv v2.37.1 main.go (MIT, (c) 2019 zimbatm and contributors).
// Package main implements the direnv command-line tool: the process entry point
// that embeds the bash stdlib contract and the pinned version, then hands off to
// the effectful engine in goforge.dev/gpdirenv/internal/cmd.
package main

import (
	_ "embed"
	"os"
	"strings"

	"goforge.dev/gpdirenv/internal/cmd"
)

var (
	// Configured at compile time (kept for upstream parity; normally empty).
	bashPath string
	//go:embed stdlib.sh
	stdlib string
	//go:embed version.txt
	version string
)

func main() {
	var (
		env  = cmd.GetEnv()
		args = os.Args
	)
	err := cmd.Main(env, args, bashPath, stdlib, strings.TrimSpace(version))
	if err != nil {
		os.Exit(1)
	}
}
