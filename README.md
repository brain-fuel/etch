# etch & scry

Hex tools for the [goforge.dev](https://goforge.dev) suite, in one
[goforge](https://goforge.dev/goforge/) Polylith workspace:

- **etch** — a full-screen terminal hex editor (overwrite-only).
- **scry** — a hex viewer: `xxd`-style dump with
  [hexyl](https://github.com/sharkdp/hexyl)-style colors, paged like
  [rubric](https://goforge.dev/rubric/).

Both reuse [rubric](https://github.com/brain-fuel/rubric)'s components for
pager handling, terminal detection, and ANSI painting.

## Install

```sh
go install goforge.dev/etch/cmd/...@latest   # both etch and scry
```

Or individually:

```sh
go install goforge.dev/etch/cmd/etch@latest
go install goforge.dev/etch/cmd/scry@latest
```

## scry — hex viewer

```sh
scry file.bin            # dump whole file, paged, colored on a tty
scry -s 0x40 -n 64 a.out # 64 bytes starting at offset 0x40
scry -w 8 -g 4 file.bin  # 8 bytes per row in groups of 4
cat data | scry          # reads stdin
```

| Flag | Meaning |
|------|---------|
| `-n N` | dump at most N bytes (`0x...` accepted) |
| `-s OFF` | skip OFF bytes from the start (`0x...` accepted) |
| `-w N` | bytes per row (default 16) |
| `-g N` | bytes per group within a row (default 8) |
| `--color auto\|always\|never` | colorize (default auto: tty + `NO_COLOR`/`COLORTERM` rules) |
| `--paging auto\|always\|never` | pager (default auto, like rubric/bat) |

Byte colors follow hexyl's classes: NUL grey, printable cyan, whitespace
green, other ASCII control magenta, non-ASCII yellow.

## etch — hex editor

```sh
etch file.bin
```

Classic offset / hex / ASCII layout. Overwrite-only: bytes are edited in
place and the file size never changes (no insert/delete).

| Key | Action |
|-----|--------|
| `h`/`j`/`k`/`l`, arrows | move by byte / row |
| `tab` | switch between hex and ASCII pane (`esc` returns to hex) |
| hex digits | overwrite the byte under the cursor, nibble by nibble (hex pane) |
| printable keys | overwrite the byte under the cursor (ASCII pane) |
| `u` | undo last change |
| `g` / `G` | first / last byte |
| `ctrl-d`/`ctrl-u`, `ctrl-f`/`ctrl-b` | half page / page |
| `o` | go to offset (decimal or `0x...`) |
| `w`, `ctrl-s` | write file |
| `q`, `ctrl-q` | quit (asks twice when there are unsaved changes) |

In the ASCII pane printable keys are data, so use `ctrl-s`/`ctrl-q` there —
or `esc` back to the hex pane where `w`/`q`/`u` are commands. Modified bytes
are highlighted until written.

## Workspace layout

A goforge Polylith workspace (module `goforge.dev/etch`):

- `components/hexdump` — pure: bytes → offset/hex/ASCII rows, optional
  hexyl-style coloring (uses rubric's `decorations`).
- `bases/scrycli` — the scry CLI: flags, input windowing, color/paging
  policy via rubric's `termdetect`, output via rubric's `pager`.
- `bases/etchtui` — the etch TUI editor (tcell).
- `projects/scry`, `projects/etch` — the deployable artifacts;
  `cmd/scry`, `cmd/etch` are thin aliases so `go install
  goforge.dev/etch/cmd/...@latest` works.

`goforge check` validates the brick boundaries and worker types.

## Used by

[lsf](https://goforge.dev/lsf/) opens binaries with etch (`e`/`enter`) and
views them with scry (`v`); its `x` key shows an inline hex peek.

## License

MIT
