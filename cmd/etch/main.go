// Command etch is a full-screen terminal hex editor (overwrite-only).
//
// This entry point exists so `go install goforge.dev/etch/cmd/etch@latest`
// works. The canonical project lives under projects/etch (goforge Polylith
// layout); both share the same bases/etchtui entry point.
package main

import (
	"os"

	"goforge.dev/etch/bases/etchtui"
)

func main() {
	os.Exit(etchtui.Run(os.Args[1:]))
}
