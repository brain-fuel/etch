// Package scrycli is the scrycli base: the command-line hex viewer. It reads
// a file (or stdin), formats it with the hexdump component, and writes the
// result through rubric's pager component, with color and paging decided by
// rubric's terminal detection — the same behavior rubric/bat have for text.
package scrycli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"goforge.dev/etch/components/hexdump"
	"goforge.dev/rubric/components/config"
	output "goforge.dev/rubric/components/pager"
	terminfo "goforge.dev/rubric/components/termdetect"
)

// Version is the release version, overridable at build time.
var Version = "0.1.2"

// maxStdinBytes caps how much of a non-seekable stdin stream is read when no
// -n limit is given, so `cat /dev/zero | scry` cannot exhaust memory.
const maxStdinBytes = 256 * 1024 * 1024

// Run is the scry entry point. It returns the process exit code.
func Run(args []string) int {
	fs := flag.NewFlagSet("scry", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		length  = fs.String("n", "", "dump at most `N` bytes (0x... accepted)")
		skip    = fs.String("s", "", "skip `OFF` bytes from the start (0x... accepted)")
		width   = fs.Int("w", 16, "bytes per row")
		group   = fs.Int("g", 8, "bytes per group within a row")
		color   = fs.String("color", "auto", "colorize output: auto, always, never")
		paging  = fs.String("paging", "auto", "pipe output through a pager: auto, always, never")
		version = fs.Bool("version", false, "print version and exit")
	)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: scry [options] [file]\n\nA hex viewer: xxd-style dump with hexyl-style colors, paged like rubric.\nReads stdin when no file (or \"-\") is given.\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *version {
		fmt.Println("scry", Version)
		return 0
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(os.Stderr, "scry: at most one file argument")
		return 2
	}
	if err := validateOptions(*width, *group, *color, *paging); err != nil {
		fmt.Fprintln(os.Stderr, "scry:", err)
		return 2
	}

	nBytes, err := parseSize(*length, -1)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scry: bad -n:", err)
		return 2
	}
	offset, err := parseSize(*skip, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scry: bad -s:", err)
		return 2
	}

	data, err := readInput(fs.Arg(0), offset, nBytes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scry:", err)
		return 1
	}

	interactive := terminfo.StdoutIsTerminal()
	useColor := false
	switch *color {
	case "always":
		useColor = true
	case "never":
	default:
		useColor = interactive && terminfo.ColorEnabled()
	}
	mode := config.PagingAuto
	switch *paging {
	case "always":
		mode = config.PagingAlways
	case "never":
		mode = config.PagingNever
	}

	// quit-if-one-screen (less -F) only in auto mode, like rubric/bat:
	// --paging always must keep the pager open for short dumps.
	out := output.FromMode(mode, "", interactive, false, mode == config.PagingAuto)
	defer out.Close()

	cfg := hexdump.Config{BytesPerRow: *width, GroupSize: *group, BaseOffset: offset, Color: useColor}
	for _, row := range hexdump.Dump(data, cfg) {
		if _, err := out.WriteString(row + "\n"); err != nil {
			return 1
		}
	}
	return 0
}

func validateOptions(width, group int, color, paging string) error {
	if width <= 0 {
		return fmt.Errorf("-w must be greater than zero")
	}
	if group <= 0 {
		return fmt.Errorf("-g must be greater than zero")
	}
	if group > width {
		return fmt.Errorf("-g must be less than or equal to -w")
	}
	if !oneOf(color, "auto", "always", "never") {
		return fmt.Errorf("--color must be auto, always, or never")
	}
	if !oneOf(paging, "auto", "always", "never") {
		return fmt.Errorf("--paging must be auto, always, or never")
	}
	return nil
}

func oneOf(got string, allowed ...string) bool {
	for _, v := range allowed {
		if got == v {
			return true
		}
	}
	return false
}

// parseSize parses a decimal or 0x-prefixed size flag; empty means def.
func parseSize(s string, def int64) (int64, error) {
	if s == "" {
		return def, nil
	}
	n, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, fmt.Errorf("negative size %d", n)
	}
	return n, nil
}

// readInput reads the dump window (offset, limit) from path or stdin ("-" or
// empty). limit < 0 means to EOF (capped for stdin).
func readInput(path string, offset, limit int64) ([]byte, error) {
	if path == "" || path == "-" {
		return readStream(os.Stdin, offset, limit)
	}
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	info, err := fh.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return readStream(fh, offset, limit)
	}
	size := info.Size()
	if offset >= size {
		return nil, nil
	}
	n := size - offset
	if limit >= 0 && limit < n {
		n = limit
	}
	buf := make([]byte, n)
	if _, err := fh.ReadAt(buf, offset); err != nil && err != io.EOF {
		return nil, err
	}
	return buf, nil
}

// readStream reads from a non-seekable reader, discarding offset bytes first.
func readStream(r io.Reader, offset, limit int64) ([]byte, error) {
	if offset > 0 {
		if _, err := io.CopyN(io.Discard, r, offset); err != nil {
			if err == io.EOF {
				return nil, nil
			}
			return nil, err
		}
	}
	max := int64(maxStdinBytes)
	if limit >= 0 {
		max = limit
	}
	return io.ReadAll(io.LimitReader(r, max))
}
