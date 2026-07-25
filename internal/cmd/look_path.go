// Ported from direnv v2.37.1 internal/cmd (MIT, (c) 2019 zimbatm and
// contributors). Effectful/runtime-boundary glue kept in plain Go; the pure
// env/shell semantic cores live in the goforge.dev/gpdirenv/{env,shell}
// packages. Faithful vendor + import surgery — behavior preserved verbatim.
package cmd

import (
	"errors"
	"os"
	"strings"
)

// Similar to os/exec.LookPath except we pass in the PATH
func lookPath(file string, pathenv string) (string, error) {
	if strings.Contains(file, "/") {
		err := findExecutable(file)
		if err == nil {
			return file, nil
		}
		return "", err
	}
	if pathenv == "" {
		return "", errNotFound
	}
	for _, dir := range strings.Split(pathenv, ":") {
		if dir == "" {
			// Unix shell semantics: path element "" means "."
			dir = "."
		}
		path := dir + "/" + file
		if err := findExecutable(path); err == nil {
			return path, nil
		}
	}
	return "", errNotFound
}

// ErrNotFound is the error resulting if a path search failed to find an executable file.
var errNotFound = errors.New("executable file not found in $PATH")

func findExecutable(file string) error {
	d, err := os.Stat(file)
	if err != nil {
		return err
	}
	if m := d.Mode(); !m.IsDir() && m&0111 != 0 {
		return nil
	}
	return os.ErrPermission
}
