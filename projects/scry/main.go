// Command scry is a terminal hex viewer: xxd-style dump with hexyl-style
// colors, paged through rubric's pager.
package main

import (
	"os"

	"goforge.dev/etch/bases/scrycli"
)

func main() {
	os.Exit(scrycli.Run(os.Args[1:]))
}
