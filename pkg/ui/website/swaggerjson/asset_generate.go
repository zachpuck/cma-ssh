// +build ignore

package main

import (
	"github.com/samsung-cnct/cma-ssh/pkg/ui/website/swaggerjson"
	"github.com/shurcooL/vfsgen"
	"log"
)

func main() {
	if err := vfsgen.Generate(swaggerjson.Swagger, vfsgen.Options{
		PackageName:  "swaggerjson",
		BuildTags:    "!dev",
		VariableName: "Swagger",
	}); err != nil {
		log.Fatalln(err)
	}

}
