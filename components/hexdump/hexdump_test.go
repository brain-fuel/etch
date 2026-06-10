package hexdump

import (
	"strings"
	"testing"
)

func TestRowPlain(t *testing.T) {
	chunk := []byte{0x7f, 'E', 'L', 'F', 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8}
	got := Row(chunk, 0, Config{})
	want := "00000000  7f 45 4c 46 00 00 00 00  01 02 03 04 05 06 07 08 |.ELF............|"
	if got != want {
		t.Errorf("Row = %q, want %q", got, want)
	}
}

func TestRowShortChunkPads(t *testing.T) {
	got := Row([]byte{'h', 'i'}, 16, Config{})
	want := "00000010  68 69                                            |hi|"
	if got != want {
		t.Errorf("Row = %q, want %q", got, want)
	}
}

func TestDumpRowCountAndOffsets(t *testing.T) {
	data := make([]byte, 33)
	rows := Dump(data, Config{BaseOffset: 0x100})
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	if !strings.HasPrefix(rows[1], "00000110") {
		t.Errorf("row 1 offset wrong: %q", rows[1])
	}
	if !strings.HasPrefix(rows[2], "00000120") {
		t.Errorf("row 2 offset wrong: %q", rows[2])
	}
}

func TestDumpCustomWidth(t *testing.T) {
	rows := Dump(make([]byte, 8), Config{BytesPerRow: 4, GroupSize: 2})
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	want := "00000000  00 00  00 00 |....|"
	if rows[0] != want {
		t.Errorf("row = %q, want %q", rows[0], want)
	}
}

func TestColorOutputsSGR(t *testing.T) {
	rows := Dump([]byte{0x00, 'A', '\n', 0x01, 0xff}, Config{Color: true})
	if len(rows) != 1 {
		t.Fatalf("got %d rows", len(rows))
	}
	for _, sgr := range []string{"\x1b[90m", "\x1b[36m", "\x1b[32m", "\x1b[35m", "\x1b[33m"} {
		if !strings.Contains(rows[0], sgr) {
			t.Errorf("row missing %q: %q", sgr, rows[0])
		}
	}
}

func TestPrintable(t *testing.T) {
	if Printable('A') != 'A' || Printable(0x00) != '.' || Printable(0x7f) != '.' {
		t.Error("Printable mapping wrong")
	}
}
