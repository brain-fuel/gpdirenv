// Package xdg is a minimal implementation of the XDG Base Directory spec.
//
// https://specifications.freedesktop.org/basedir-spec/latest/
//
// Adapted from direnv v2.37.1 xdg (MIT, (c) 2019 zimbatm and contributors).
// Pure (env-in, path-out); differentially tested against the pinned upstream
// logic.
package xdg

import "path/filepath"

// DataDir returns the data directory for the application.
func DataDir(env map[string]string, programName string) string {
	if env["XDG_DATA_HOME"] != "" {
		return filepath.Join(env["XDG_DATA_HOME"], programName)
	} else if env["HOME"] != "" {
		return filepath.Join(env["HOME"], ".local", "share", programName)
	}
	return ""
}

// ConfigDir returns the config directory for the application.
// The XDG_CONFIG_DIRS case is intentionally not handled (matches upstream).
func ConfigDir(env map[string]string, programName string) string {
	if env["XDG_CONFIG_HOME"] != "" {
		return filepath.Join(env["XDG_CONFIG_HOME"], programName)
	} else if env["HOME"] != "" {
		return filepath.Join(env["HOME"], ".config", programName)
	}
	return ""
}

// CacheDir returns the cache directory for the application.
func CacheDir(env map[string]string, programName string) string {
	if env["XDG_CACHE_HOME"] != "" {
		return filepath.Join(env["XDG_CACHE_HOME"], programName)
	} else if env["HOME"] != "" {
		return filepath.Join(env["HOME"], ".cache", programName)
	}
	return ""
}
