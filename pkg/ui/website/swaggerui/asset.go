// +build dev

package swaggerui

import (
	"github.com/samsung-cnct/cma-ssh/pkg/util"
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

var SwaggerUI util.ZeroModTimeFileSystem = util.ZeroModTimeFileSystem{Source: http.Dir(
	importPathToDir("github.com/samsung-cnct/cma-ssh/third_party/swagger-ui"),
)}
