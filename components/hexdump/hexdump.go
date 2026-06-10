// Package hexdump is the public interface of the hexdump component: pure
// xxd-style formatting of bytes into offset / hex / ASCII rows, with
// optional hexyl-style ANSI coloring (via rubric's decorations component).
// Other bricks depend on this package only; the implementation lives under
// internal/ and may not be imported across brick boundaries.
package hexdump

import "goforge.dev/etch/components/hexdump/internal"

// Config controls how Dump and Row format bytes.
type Config = internal.Config

// Dump formats data as a sequence of xxd-style rows.
func Dump(data []byte, cfg Config) []string {
	return internal.Dump(data, cfg)
}

// Row formats one dump row for chunk, whose first byte sits at offset off.
func Row(chunk []byte, off int64, cfg Config) string {
	return internal.Row(chunk, off, cfg)
}

// Printable maps a byte to its ASCII-column representation: the byte itself
// if printable ASCII, '.' otherwise.
func Printable(b byte) byte {
	return internal.Printable(b)
}
