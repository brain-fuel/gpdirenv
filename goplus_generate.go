// `go generate ./...` regenerates every *_gp.go in the module from its .gp
// source; plain `go build`/`go test` consumes the committed generated Go.

//go:generate go tool goplus gen ./...

package gpdirenv
