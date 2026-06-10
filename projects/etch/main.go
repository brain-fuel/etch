// Command etch is a full-screen terminal hex editor (overwrite-only).
package main

import (
	"os"

	"goforge.dev/etch/bases/etchtui"
)

func main() {
	os.Exit(etchtui.Run(os.Args[1:]))
}
