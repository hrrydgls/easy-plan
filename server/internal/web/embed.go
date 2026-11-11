// Package web embeds the built SvelteKit static frontend into the binary.
// Run `make build-web` (or the equivalent pnpm build + copy) before
// `go build` so the dist/ directory below is populated.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// FS returns the embedded frontend build, rooted at dist/.
func FS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err) // programmer error: dist/ missing from the embed
	}
	return sub
}
