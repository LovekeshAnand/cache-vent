// Package definitions embeds the definition files into the binary.
package definitions

import "embed"

//go:embed *.yaml
var FS embed.FS
