// Package etchtui is the etchtui base: a full-screen terminal hex editor.
// It shows the classic offset / hex / ASCII layout and edits bytes in place
// (overwrite-only: no insert or delete, so file size never changes).
//
// Keys: hjkl/arrows move · tab switch hex/ASCII pane (esc returns to hex) ·
// hex digits or ASCII type to overwrite · u undo · g/G start/end ·
// ctrl-d/u half page · ctrl-f/b page · o go to offset · w/ctrl-s write ·
// q/ctrl-q quit. In the ASCII pane printable keys are data; use ctrl-s and
// ctrl-q there, or esc back to the hex pane.
package etchtui

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gdamore/tcell/v3"

	"goforge.dev/etch/components/hexdump"
)

// Version is stamped by the release; "dev" otherwise.
var Version = "dev"

const (
	bytesPerRow = 16
	// maxFileBytes guards the whole-file load; etch is not (yet) a
	// streaming editor.
	maxFileBytes = 1 << 30
)

var (
	gutterSt   = tcell.StyleDefault.Foreground(tcell.PaletteColor(238))
	asciiSt    = tcell.StyleDefault.Foreground(tcell.PaletteColor(245))
	headerSt   = tcell.StyleDefault.Bold(true)
	msgSt      = tcell.StyleDefault.Foreground(tcell.PaletteColor(244)).Italic(true)
	errSt      = tcell.StyleDefault.Foreground(tcell.PaletteColor(1))
	modifiedSt = tcell.StyleDefault.Foreground(tcell.PaletteColor(3)).Bold(true)
	cursorSt   = tcell.StyleDefault.Reverse(true)
	promptSt   = tcell.StyleDefault.Foreground(tcell.PaletteColor(2)).Bold(true)
)

type change struct {
	idx      int64
	old, new byte
}

type editor struct {
	screen tcell.Screen
	path   string
	mode   os.FileMode
	data   []byte
	orig   []byte
	undo   []change
	top    int64 // offset of first visible row (multiple of bytesPerRow)
	cur    int64 // cursor byte offset
	nibble int   // -1, or high-nibble value waiting for the second hex digit
	ascii  bool  // ASCII pane focused
	prompt bool  // offset prompt active
	input  string
	msg    string
	msgErr bool
	quitOK bool // q pressed once with unsaved changes
	quit   bool
}

// Run is the etch entry point. It returns the process exit code.
func Run(args []string) int {
	if len(args) == 1 && (args[0] == "--version" || args[0] == "-version") {
		fmt.Println("etch", Version)
		return 0
	}
	if len(args) != 1 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprintln(os.Stderr, "usage: etch <file>\n\nA terminal hex editor (overwrite-only). Keys: hjkl/arrows move · tab hex/ASCII\npane (esc returns to hex) · type hex digits or ASCII to overwrite · u undo ·\ng/G start/end · ctrl-d/u half page · ctrl-f/b page · o go to offset ·\nw/ctrl-s write · q/ctrl-q quit.")
		if len(args) == 1 {
			return 0
		}
		return 2
	}
	path := args[0]
	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "etch:", err)
		return 1
	}
	if !info.Mode().IsRegular() {
		fmt.Fprintln(os.Stderr, "etch: not a regular file:", path)
		return 1
	}
	if info.Size() > maxFileBytes {
		fmt.Fprintln(os.Stderr, "etch: file larger than 1 GiB:", path)
		return 1
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "etch:", err)
		return 1
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintln(os.Stderr, "etch:", err)
		return 1
	}
	if err := screen.Init(); err != nil {
		fmt.Fprintln(os.Stderr, "etch:", err)
		return 1
	}
	defer screen.Fini()

	ed := &editor{
		screen: screen,
		path:   path,
		mode:   info.Mode(),
		data:   data,
		orig:   append([]byte(nil), data...),
		nibble: -1,
	}
	ed.loop()
	return 0
}

func (ed *editor) loop() {
	ed.draw()
	for ev := range ed.screen.EventQ() {
		switch ev := ev.(type) {
		case *tcell.EventResize:
			ed.screen.Sync()
		case *tcell.EventKey:
			if !ev.Pressed() {
				continue
			}
			ed.msg, ed.msgErr = "", false
			if ed.prompt {
				ed.handlePromptKey(ev)
			} else {
				ed.handleKey(ev)
			}
		default:
			continue
		}
		if ed.quit {
			return
		}
		ed.draw()
	}
}

func (ed *editor) dirty() bool {
	for _, c := range ed.undo {
		if ed.data[c.idx] != ed.orig[c.idx] {
			return true
		}
	}
	return false
}

func (ed *editor) handleKey(ev *tcell.EventKey) {
	rows := ed.viewRows()
	half := int64(max(1, rows/2)) * bytesPerRow
	page := int64(max(1, rows)) * bytesPerRow

	switch ev.Key() {
	case tcell.KeyEscape:
		// Esc cancels a pending nibble and always returns to the hex
		// pane, where w/q/u are commands rather than data.
		ed.nibble = -1
		ed.ascii = false
		return
	case tcell.KeyCtrlC, tcell.KeyCtrlQ:
		if ed.dirty() && !ed.quitOK {
			ed.quitOK = true
			ed.msg, ed.msgErr = "unsaved changes — repeat to discard, ctrl-s to write", true
			return
		}
		ed.quit = true
		return
	case tcell.KeyCtrlS:
		ed.save()
		return
	case tcell.KeyUp:
		ed.move(-bytesPerRow)
		return
	case tcell.KeyDown:
		ed.move(bytesPerRow)
		return
	case tcell.KeyLeft:
		ed.move(-1)
		return
	case tcell.KeyRight:
		ed.move(1)
		return
	case tcell.KeyTab:
		ed.ascii = !ed.ascii
		ed.nibble = -1
		return
	case tcell.KeyCtrlU:
		ed.move(-half)
		return
	case tcell.KeyCtrlD:
		ed.move(half)
		return
	case tcell.KeyCtrlB:
		ed.move(-page)
		return
	case tcell.KeyCtrlF:
		ed.move(page)
		return
	case tcell.KeyEnter:
		return
	}
	if ev.Key() != tcell.KeyRune {
		return
	}
	r := rune(0)
	for _, rr := range ev.Str() {
		r = rr
		break
	}

	// In the ASCII pane every printable rune is data, so pane-agnostic
	// commands must come first only for the hex pane.
	if ed.ascii {
		switch r {
		case 0:
			return
		default:
			if r >= 0x20 && r < 0x7f {
				ed.setByte(byte(r))
				ed.move(1)
				return
			}
		}
		return
	}

	if v := hexDigit(r); v >= 0 {
		if len(ed.data) == 0 {
			return
		}
		if ed.nibble < 0 {
			ed.setByte(byte(v) << 4)
			ed.nibble = v
		} else {
			ed.setByte(byte(ed.nibble)<<4 | byte(v))
			ed.nibble = -1
			ed.move(1)
		}
		return
	}

	switch r {
	case 'q':
		if ed.dirty() && !ed.quitOK {
			ed.quitOK = true
			ed.msg, ed.msgErr = "unsaved changes — q again to discard, w to write", true
			return
		}
		ed.quit = true
	case 'w':
		ed.save()
	case 'u':
		ed.undoOne()
	case 'h':
		ed.move(-1)
	case 'l':
		ed.move(1)
	case 'j':
		ed.move(bytesPerRow)
	case 'k':
		ed.move(-bytesPerRow)
	case 'g':
		ed.cur = 0
		ed.scrollTo()
	case 'G':
		ed.cur = int64(len(ed.data)) - 1
		if ed.cur < 0 {
			ed.cur = 0
		}
		ed.scrollTo()
	case 'o':
		ed.prompt = true
		ed.input = ""
	}
}

func (ed *editor) handlePromptKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyCtrlC:
		ed.prompt = false
		ed.input = ""
	case tcell.KeyEnter:
		ed.prompt = false
		off, err := strconv.ParseInt(ed.input, 0, 64)
		ed.input = ""
		if err != nil || off < 0 {
			ed.msg, ed.msgErr = "bad offset", true
			return
		}
		if off >= int64(len(ed.data)) {
			off = int64(len(ed.data)) - 1
		}
		if off < 0 {
			off = 0
		}
		ed.cur = off
		ed.scrollTo()
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(ed.input) > 0 {
			ed.input = ed.input[:len(ed.input)-1]
		}
	case tcell.KeyRune:
		ed.input += ev.Str()
	}
}

// setByte records an undo entry and overwrites the byte under the cursor.
func (ed *editor) setByte(b byte) {
	if len(ed.data) == 0 {
		return
	}
	ed.undo = append(ed.undo, change{idx: ed.cur, old: ed.data[ed.cur], new: b})
	ed.data[ed.cur] = b
	ed.quitOK = false
}

func (ed *editor) undoOne() {
	if len(ed.undo) == 0 {
		ed.msg = "nothing to undo"
		return
	}
	c := ed.undo[len(ed.undo)-1]
	ed.undo = ed.undo[:len(ed.undo)-1]
	ed.data[c.idx] = c.old
	ed.cur = c.idx
	ed.nibble = -1
	ed.scrollTo()
}

func (ed *editor) save() {
	if err := os.WriteFile(ed.path, ed.data, ed.mode.Perm()); err != nil {
		ed.msg, ed.msgErr = err.Error(), true
		return
	}
	ed.orig = append(ed.orig[:0], ed.data...)
	ed.undo = ed.undo[:0]
	ed.msg = fmt.Sprintf("wrote %d bytes to %s", len(ed.data), ed.path)
}

func (ed *editor) move(d int64) {
	ed.nibble = -1
	ed.cur += d
	if ed.cur < 0 {
		ed.cur = 0
	}
	if maxIdx := int64(len(ed.data)) - 1; ed.cur > maxIdx {
		ed.cur = maxIdx
		if ed.cur < 0 {
			ed.cur = 0
		}
	}
	ed.scrollTo()
}

// viewRows returns how many dump rows fit between header and status line.
func (ed *editor) viewRows() int {
	_, h := ed.screen.Size()
	return max(1, h-2)
}

// scrollTo keeps the cursor's row visible.
func (ed *editor) scrollTo() {
	rows := int64(ed.viewRows())
	curRow := ed.cur / bytesPerRow
	topRow := ed.top / bytesPerRow
	if curRow < topRow {
		topRow = curRow
	}
	if curRow >= topRow+rows {
		topRow = curRow - rows + 1
	}
	ed.top = topRow * bytesPerRow
}

func (ed *editor) draw() {
	ed.screen.Clear()
	w, h := ed.screen.Size()

	// Header: path, size, dirty marker.
	dirty := ""
	if ed.dirty() {
		dirty = " [modified]"
	}
	printLine(ed.screen, 0, 0, w, fmt.Sprintf(" %s — %d bytes%s", ed.path, len(ed.data), dirty), headerSt)

	rows := ed.viewRows()
	for row := 0; row < rows; row++ {
		off := ed.top + int64(row)*bytesPerRow
		if off >= int64(len(ed.data)) {
			break
		}
		ed.drawRow(1+row, off, w)
	}

	// Status line: prompt, message, or cursor info.
	y := h - 1
	switch {
	case ed.prompt:
		printLine(ed.screen, 0, y, w, " offset: "+ed.input, promptSt)
	case ed.msg != "":
		st := msgSt
		if ed.msgErr {
			st = errSt
		}
		printLine(ed.screen, 0, y, w, " "+ed.msg, st)
	default:
		pane := "hex"
		if ed.ascii {
			pane = "ascii"
		}
		val := ""
		if len(ed.data) > 0 {
			val = fmt.Sprintf("0x%02x %3d", ed.data[ed.cur], ed.data[ed.cur])
		}
		printLine(ed.screen, 0, y, w,
			fmt.Sprintf(" 0x%08x (%d/%d)  %s  [%s]", ed.cur, ed.cur, len(ed.data), val, pane), msgSt)
	}
	ed.screen.Show()
}

// drawRow renders one offset/hex/ascii row at screen row y.
func (ed *editor) drawRow(y int, off int64, w int) {
	end := off + bytesPerRow
	if end > int64(len(ed.data)) {
		end = int64(len(ed.data))
	}
	chunk := ed.data[off:end]

	x := printLine(ed.screen, 0, y, w, fmt.Sprintf("%08x  ", off), gutterSt)
	for i := 0; i < bytesPerRow; i++ {
		if i > 0 && i%8 == 0 {
			x += printLine(ed.screen, x, y, w-x, " ", tcell.StyleDefault)
		}
		if i >= len(chunk) {
			x += printLine(ed.screen, x, y, w-x, "   ", tcell.StyleDefault)
			continue
		}
		idx := off + int64(i)
		st := tcell.StyleDefault
		if ed.data[idx] != ed.orig[idx] {
			st = modifiedSt
		}
		if idx == ed.cur && !ed.ascii {
			st = cursorSt
		}
		x += printLine(ed.screen, x, y, w-x, fmt.Sprintf("%02x", ed.data[idx]), st)
		x += printLine(ed.screen, x, y, w-x, " ", tcell.StyleDefault)
	}
	x += printLine(ed.screen, x, y, w-x, "|", asciiSt)
	for i := range chunk {
		idx := off + int64(i)
		st := asciiSt
		if ed.data[idx] != ed.orig[idx] {
			st = modifiedSt
		}
		if idx == ed.cur && ed.ascii {
			st = cursorSt
		}
		x += printLine(ed.screen, x, y, w-x, string(rune(hexdump.Printable(ed.data[idx]))), st)
	}
	printLine(ed.screen, x, y, w-x, "|", asciiSt)
}

// printLine draws str at (x, y) clipped to maxw columns; returns columns used.
func printLine(s tcell.Screen, x, y, maxw int, str string, st tcell.Style) int {
	col := 0
	for _, r := range str {
		if col >= maxw {
			break
		}
		s.PutStrStyled(x+col, y, string(r), st)
		col++
	}
	return col
}

// hexDigit returns the value of a hex digit rune, or -1.
func hexDigit(r rune) int {
	switch {
	case r >= '0' && r <= '9':
		return int(r - '0')
	case r >= 'a' && r <= 'f':
		return int(r-'a') + 10
	case r >= 'A' && r <= 'F':
		return int(r-'A') + 10
	}
	return -1
}
