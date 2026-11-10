// Package web holds the built frontend, embedded into the server binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed dist
var dist embed.FS

// Dist returns the built frontend, with index.html at its root. `make web` builds it.
func Dist() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err) // dist is embedded at build time; fs.Sub fails only on an invalid name
	}
	return sub
}
