// +build ignore

package main

import (
	"github.com/samsung-cnct/cma-ssh/pkg/ui/website/swaggerui"
	"github.com/shurcooL/vfsgen"
	"log"
)

func main() {
	if err := vfsgen.Generate(swaggerui.SwaggerUI, vfsgen.Options{
		PackageName:  "swaggerui",
		BuildTags:    "!dev",
		VariableName: "SwaggerUI",
	}); err != nil {
		log.Fatalln(err)
	}
}
