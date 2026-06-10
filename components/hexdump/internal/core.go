// Package internal holds the hexdump component implementation. Only the
// hexdump component's interface package may import it (enforced by Go's
// internal/ visibility rules and by `goforge check`).
package internal

import (
	"fmt"
	"strings"

	"goforge.dev/rubric/components/decorations"
)

// Config controls how Dump and Row format bytes.
type Config struct {
	BytesPerRow int   // bytes per output row; 0 means 16
	GroupSize   int   // bytes per space-separated group; 0 means 8
	BaseOffset  int64 // offset printed for the first byte
	Color       bool  // emit ANSI SGR colors (hexyl-style byte classes)
}

// WithDefaults fills zero fields with the xxd-compatible defaults.
func (c Config) WithDefaults() Config {
	if c.BytesPerRow <= 0 {
		c.BytesPerRow = 16
	}
	if c.GroupSize <= 0 {
		c.GroupSize = 8
	}
	return c
}

// SGR classes per byte, following hexyl's scheme: NUL grey, printable cyan,
// whitespace green, other ASCII control magenta, non-ASCII yellow.
const (
	sgrOffset    = "90"
	sgrNull      = "90"
	sgrPrintable = "36"
	sgrSpace     = "32"
	sgrControl   = "35"
	sgrNonASCII  = "33"
)

func classSGR(b byte) string {
	switch {
	case b == 0x00:
		return sgrNull
	case b >= 0x20 && b < 0x7f:
		return sgrPrintable
	case b == '\t' || b == '\n' || b == '\r' || b == '\v' || b == '\f':
		return sgrSpace
	case b < 0x80:
		return sgrControl
	default:
		return sgrNonASCII
	}
}

// Printable maps a byte to its ASCII-column representation.
func Printable(b byte) byte {
	if b >= 0x20 && b < 0x7f {
		return b
	}
	return '.'
}

func paint(color bool, sgr, text string) string {
	if !color || sgr == "" {
		return text
	}
	return decorations.Paint(sgr, text)
}

// Row formats one dump row for chunk, whose first byte sits at offset off.
// len(chunk) may be less than cfg.BytesPerRow; the hex column is padded so
// the ASCII column always aligns.
func Row(chunk []byte, off int64, cfg Config) string {
	cfg = cfg.WithDefaults()
	var sb strings.Builder
	sb.WriteString(paint(cfg.Color, sgrOffset, fmt.Sprintf("%08x", off)))
	sb.WriteString("  ")
	for i := 0; i < cfg.BytesPerRow; i++ {
		if i > 0 && i%cfg.GroupSize == 0 {
			sb.WriteByte(' ')
		}
		if i < len(chunk) {
			sb.WriteString(paint(cfg.Color, classSGR(chunk[i]), fmt.Sprintf("%02x", chunk[i])))
			sb.WriteByte(' ')
		} else {
			sb.WriteString("   ")
		}
	}
	sb.WriteByte('|')
	for _, b := range chunk {
		sb.WriteString(paint(cfg.Color, classSGR(b), string(Printable(b))))
	}
	sb.WriteByte('|')
	return sb.String()
}

// Dump formats data as a sequence of rows.
func Dump(data []byte, cfg Config) []string {
	cfg = cfg.WithDefaults()
	rows := make([]string, 0, (len(data)+cfg.BytesPerRow-1)/cfg.BytesPerRow)
	for i := 0; i < len(data); i += cfg.BytesPerRow {
		end := i + cfg.BytesPerRow
		if end > len(data) {
			end = len(data)
		}
		rows = append(rows, Row(data[i:end], cfg.BaseOffset+int64(i), cfg))
	}
	return rows
}
