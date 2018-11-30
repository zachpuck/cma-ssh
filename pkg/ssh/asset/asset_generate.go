// +build ignore

package main

import (
	"log"

	"github.com/samsung-cnct/cma-ssh/pkg/ssh/asset"
	"github.com/shurcooL/vfsgen"
)

func main() {
	if err := vfsgen.Generate(asset.Assets, vfsgen.Options{
		PackageName:  "asset",
		BuildTags:    "!dev",
		VariableName: "Assets",
	}); err != nil {
		log.Fatalln(err)
	}
}
