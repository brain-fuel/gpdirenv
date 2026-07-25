# gpdirenv — parity ledger

Go+ reimplementation of [`direnv/direnv`](https://github.com/direnv/direnv),
pinned to **v2.37.1** (MIT). Module: `goforge.dev/gpdirenv`. Wave 2, candidate 1.

Honest status, not aspirational. A row is "done" only when it is authored in
Go+, generated deterministically, and **differentially tested against the
pinned upstream** (not merely compiling). This mirrors the Wave 1 definition of
done in `goplus/GOFORGE_CANDIDATES_WAVE1.md`.

## Surface inventory (denominators)

| Surface | Count | Kind | Status |
|---|---:|---|---|
| Importable library packages | 3 | Go+ (`dotenv`, `sri`, `gzenv`) | **3/3 done** — byte-exact differential |
| Pure engine cores | 3 | Go+ (`env`, `shell`, `xdg`) | **3/3 done** — incl. **all 12 shells**, `BashEscape`/`FishEscape`/`TcshEscape`/PowerShell escapers byte-exact |
| Effectful engine | 4 | **Go+** (`config`, `rc`, `file_times`, `log`) | **done** — allow/deny hash store, `.envrc` resolution, watches; hermetic unit tests |
| CLI subcommands | 23 | **Go+** (`internal/cmd`) | **23/23 authored in Go+ + building**; core behaviors differential-verified vs the pinned binary |
| stdlib.sh functions | 59 | Bash contract | **done — embedded byte-identical** (`cmd/direnv/stdlib.sh`, artifact parity) |
| `direnv` binary | 1 | `cmd/direnv` | **builds; `version`=2.37.1; differential-parity vs upstream binary** |

**Parity verified (2026-07-25) against a freshly-built upstream `direnv` v2.37.1
binary.** CLI subprocess differential: **19 cases** (`version`, `help`, `status`,
all 12 `hook`s, `stdlib`, `dotenv bash`/`json`, `apply_dump`, `show_dump`,
`watch-print`) — **17 byte-identical (modulo the embedded self-path), 2
set-identical** where output order is Go map-iteration nondeterminism present
identically in upstream (proven: upstream's own `help` shell-list order varies
run-to-run). Plus the full **`.envrc` load cycle** (`allow`→`export bash`,
spawning bash + running the embedded `stdlib.sh` incl. `PATH_add`) produces
**identical exports**, and `status`/`exec` match. Not exhaustively
subprocess-differentialed (ported + building, but effect/network/editor bound):
`fetchurl` (network CAS), `edit` (`$EDITOR`), `prune`, `reload`, `log`,
`watch`/`watch-dir`/`watch-list`, `current`.

## Done (differential-first cores)

Each is byte-for-byte compatible with upstream, proven by property tests over
generated inputs plus upstream's own fixed vectors. `go test -race ./...`,
`go vet ./...`, and deterministic regeneration all clean.

- [x] **`sri`** — SRI content hashes (sha256/384/512). `Algo`, `Hash`
  (`String`/`Hex`), `Parse`, `Writer` (`NewWriter`/`Write`/`Sum`). Differential:
  digest + rendering + parse (valid & malformed) equal upstream. `sri_diff_test.go`.
- [x] **`gzenv`** — json+zlib+base64 environment snapshots. `Marshal`,
  `Unmarshal`. Differential: byte-identical encoding, forge/upstream
  cross-compatibility both directions, malformed-input parity. `gzenv_diff_test.go`.
- [x] **`dotenv`** — the `.env` parser (regex grammar, quoting, escapes,
  `$VAR` / `${VAR:-default}` expansion). `Parse`, `MustParse`. Differential: a
  structured `.env` generator exercises every value branch; map + error
  classification equal upstream; upstream's fixed vectors pinned.
  `parse_diff_test.go`.
- [x] **`env`** — the environment diff engine. `Env`, `EnvDiff`,
  `BuildEnvDiff`, `Patch`, `Reverse`, `Any`, `IgnoredEnv`/`IgnoredKeys`,
  `Serialize`/`LoadEnvDiff`, the `Shell`/`ShellExport` port, `DIRENV_*`
  constants. Differential: diff/patch/reverse/ignore parity over generated
  environments (incl. ignore-rule keys) + reverse-involution law.
  `env_diff_test.go`.
- [x] **`xdg`** — XDG base-dir resolution. `DataDir`/`ConfigDir`/`CacheDir`.
  Differential: **imports upstream `github.com/direnv/direnv/v2/xdg` directly**
  (it's not internal) and matches across every XDG/HOME combination.
  `xdg_diff_test.go`.
- [x] **`shell`** — host-shell rendering (3/12 shells this cut: `bash`, `json`,
  `gzenv`) + `DetectShell`. `BashEscape` is proven **byte-for-byte identical to
  upstream over all 256 bytes + random/invalid-UTF-8 inputs**; json/gzenv shells
  round-trip. `shell_diff_test.go`.

> **Differential oracle note:** `env` and `shell` cannot import their upstream
> counterparts — those live in direnv's `internal/cmd`, which Go forbids
> external modules from importing. So `internal/upstreamref/` holds a **verbatim,
> clearly-labeled, test-only** copy of the pinned upstream pure functions
> (`BuildEnvDiff`/`Patch`/`Reverse`/`IgnoredEnv`/`BashEscape`) as the diff
> oracle. `dotenv`/`sri`/`gzenv`/`xdg` import the real upstream directly.

## Pending (inventoried, explicitly not done)

- [ ] **Exhaustive subprocess differential for effect/network/editor commands** —
  `fetchurl` (network CAS), `edit` (`$EDITOR`), `prune`, `reload`, `log`,
  `watch`/`watch-dir`/`watch-list`, `current`. All are ported and build; their
  core logic shares the differential-verified engine, but each lacks a dedicated
  subprocess parity case.
- [x] **Go+ authorship of the effectful glue** — `internal/cmd` (engine + all 23
  commands) is now authored in **Go+** (33 `.gp` files → committed `*_gp.go`),
  matching the rest of the module. The conversion is behavior-preserving
  (differential parity unchanged). This codebase's established enum idiom is the
  typed-const + `switch` form (`sri.Algo`, `AllowStatus`), used consistently in
  the pure cores too — so no `enum`/`match` reshape was forced onto the imperative
  I/O glue, which would have been an inconsistent one-off and risked parity.
  **Regen caveat:** the larger cross-referential `cmd` package triggers the
  `resolution did not converge` bug in released goplus v0.137.0; it regenerates
  cleanly only with the convergence-fixed toolchain (unreleased WIP — same
  situation as Viper). The committed `*_gp.go` builds/tests without the toolchain.
- [ ] **`std/config` extension** — the reason direnv is Wave 2's first pick:
  capability-scoped source loading + watch/reload as a *second independent
  consumer* of the same immutable-snapshot/provenance abstraction Viper needs.
  Landing this retires Viper's open Wave 1 `std/config` item.
- [ ] **goforge workspace restructure** — currently a flat module (fastest path
  to parity). Restructure into declared bricks (semantic core vs. compat facade)
  per the Wave 2 "workspaces from day one" goal — now unblocked by goforge's new
  `.gp` awareness.
- [ ] **Performance discipline**, **blessed release / goforge.dev page** — per the
  shared definition of done. This push is the working-parity milestone, not a
  blessed `v1.0.0`.

## Development notes

- **No `replace`** — the module pins released `goforge.dev/goplus v0.137.0`
  (verified: released goplus reproduces the committed `*_gp.go` identically and
  `goplus gen -check` is clean). Consumers build the committed generated Go and
  never need the toolchain.
- Upstream `github.com/direnv/direnv/v2` is a test-only dependency (differential
  suites); the non-importable `internal/cmd` functions are pinned in
  `internal/upstreamref/` as a labeled test oracle.
- `go generate ./...` (or `go tool goplus gen ./...`) regenerates every `*_gp.go`;
  the effectful `internal/cmd`/`cmd/direnv` glue is plain Go (vendored from
  upstream v2.37.1 with provenance headers).
