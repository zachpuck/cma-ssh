// Package asset provides the assets to a virtual filesystem.
package swaggerui

import (
	_ "github.com/shurcooL/vfsgen"
)

//go:generate go run -tags=dev asset_generate.go
