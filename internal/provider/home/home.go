// Package home checks configured provider state directories without opening credentials.
package home

import (
	"errors"
	"os"
)

type Status int

const (
	OK Status = iota
	Missing
	Invalid
)

func Inspect(path string) Status {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Missing
		}
		return Invalid
	}
	if !info.IsDir() {
		return Invalid
	}
	// Looking up "." checks directory traversal without reading its entries.
	if _, err := os.Stat(path + string(os.PathSeparator) + "."); err != nil {
		return Invalid
	}
	return OK
}
