// direnv as a std/config consumer.
//
// direnv's per-directory `.envrc` is a configuration source, and its two
// defining mechanisms map exactly onto the std/config source-loading laws: the
// allow/deny policy IS the effect Capability that gates a Load, and a watched
// file's mtime IS the Fingerprint that drives reload. This makes direnv a
// second, independent consumer of the same primitive viper uses for its
// configuration file — the promotion bar for `std/config`.
package cmd

import (
	"fmt"
	"os"

	stdconfig "goforge.dev/goplus/std/config"
)

// rcConfigSchema is direnv's schema id for a resolved .envrc snapshot.
const rcConfigSchema = 1

// rcSource adapts an allowed-or-denied `.envrc` (plus the environment it would
// contribute) to the std/config Loader contract.
type rcSource struct {
	rc  *RC
	env Env
}

// Provenance: an `.envrc` is a file-backed configuration source.
func (s rcSource) Provenance() stdconfig.Source { return stdconfig.FileSource{} }

// Probe fingerprints the `.envrc` by its mtime — the same signal FileTimes
// records for the watch list.
func (s rcSource) Probe() (stdconfig.Fingerprint, error) {
	stat, err := getLatestStat(s.rc.path)
	if os.IsNotExist(err) {
		return stdconfig.Fingerprint{Exists: false}, nil
	}
	if err != nil {
		return stdconfig.Fingerprint{}, err
	}
	return stdconfig.Fingerprint{Token: fmt.Sprint(stat.ModTime().Unix()), Exists: true}, nil
}

// Load contributes the `.envrc` environment as a Layer only when the capability
// is granted; a denied or blocked `.envrc` is Skipped and contributes nothing —
// exactly direnv's allow/deny semantics.
func (s rcSource) Load(capability stdconfig.Capability) (stdconfig.Loaded, error) {
	fp, err := s.Probe()
	if err != nil {
		return stdconfig.Skipped{Reason: err.Error(), Fingerprint: fp}, err
	}
	if !stdconfig.IsGranted(capability) {
		return stdconfig.Skipped{Reason: "not allowed", Fingerprint: fp}, nil
	}
	values := make(map[string]any, len(s.env))
	for key, value := range s.env {
		values[key] = value
	}
	return stdconfig.Applied{
		Layer:       stdconfig.Layer{Source: stdconfig.FileSource{}, Values: values},
		Fingerprint: fp,
	}, nil
}

// rcCapability projects direnv's AllowStatus onto a std/config Capability: an
// allowed `.envrc` grants the load; not-allowed or blocked denies it.
func rcCapability(rc *RC) stdconfig.Capability {
	match rc.Allowed() {
	case Allowed:
		return stdconfig.Granted{}
	case NotAllowed:
		return stdconfig.Denied{Reason: "not allowed"}
	case Denied:
		return stdconfig.Denied{Reason: "blocked"}
	}
}

// ResolveRCConfig loads the `.envrc` under its allow-capability and resolves the
// std/config snapshot for it. A denied `.envrc` yields an empty snapshot rather
// than an error, matching direnv's silent-skip behavior.
func ResolveRCConfig(rc *RC, env Env) stdconfig.Snapshot[rcConfigSchema] {
	src := rcSource{rc: rc, env: env}
	grant := func(stdconfig.Source) stdconfig.Capability { return rcCapability(rc) }
	layers, _, _ := stdconfig.LoadAll(grant, src)
	return stdconfig.Resolve(rcConfigSchema, layers...)
}
