// +build ignore

package main

import (
	"github.com/samsung-cnct/cma-ssh/pkg/ui/website/homepage"
	"github.com/shurcooL/vfsgen"
	"log"
)

func main() {
	if err := vfsgen.Generate(homepage.Homepage, vfsgen.Options{
		PackageName:  "homepage",
		BuildTags:    "!dev",
		VariableName: "Homepage",
	}); err != nil {
		log.Fatalln(err)
	}
}
