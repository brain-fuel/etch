// Command scry is a terminal hex viewer: xxd-style dump with hexyl-style
// colors, paged through rubric's pager.
//
// This entry point exists so `go install goforge.dev/etch/cmd/scry@latest`
// works. The canonical project lives under projects/scry (goforge Polylith
// layout); both share the same bases/scrycli entry point.
package main

import (
	"os"

	"goforge.dev/etch/bases/scrycli"
)

func main() {
	os.Exit(scrycli.Run(os.Args[1:]))
}
