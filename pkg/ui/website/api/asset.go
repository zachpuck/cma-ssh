// +build dev

package api

import (
	"go/build"
	"log"
	"net/http"

	"github.com/samsung-cnct/cma-ssh/pkg/util"
)

func importPathToDir(importPath string) string {
	p, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		log.Fatalln(err)
	}
	return p.Dir
}

var Api = util.ZeroModTimeFileSystem{Source: http.Dir(
	importPathToDir("github.com/samsung-cnct/cma-ssh/api"),
)}
