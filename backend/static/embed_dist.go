//go:build dist

package static

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

func init() {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	Dist = sub
}