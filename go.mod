module goforge.dev/gpdirenv

go 1.26.0

// Go+ toolchain. `go tool goplus gen ./...` regenerates every *_gp.go.
tool goforge.dev/goplus/cmd/goplus

require goforge.dev/goplus v0.138.0 // indirect

require (
	github.com/direnv/direnv/v2 v2.37.1
	goforge.dev/goplus/std v0.207.0
	golang.org/x/mod v0.38.0
)

// Effectful port of the direnv CLI/engine (internal/cmd) — plain-Go runtime glue
// vendored from direnv v2.37.1; versions pinned to upstream go.sum.
require (
	github.com/BurntSushi/toml v1.5.0
	github.com/mattn/go-isatty v0.0.20
)

require (
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
)
