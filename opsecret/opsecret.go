// Package opsecret loads secret values from the 1Password CLI.
package opsecret

import (
	"os/exec"
	"strings"
)

// Get returns the secret string associated with the given reference.
func Get(ref string) (string, error) {
	cmd := exec.Command("op", "read", ref)
	b, err := cmd.Output()
	return strings.TrimRight(string(b), "\r\n"), err
}
