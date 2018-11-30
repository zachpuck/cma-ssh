// +build dev

package asset

import (
	"go/build"
	"log"
	"net/http"
)

func importPathToDir(importPath string) string {
	p, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		log.Fatalln(err)
	}
	return p.Dir
}

var Assets http.FileSystem = http.Dir(
	importPathToDir("github.com/samsung-cnct/cma-ssh/pkg/ssh/asset/data"),
)
