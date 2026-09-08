# Sitch

A fast, colorful system fetch in Go.

## Install

```sh
go install samouly.fun/sitch@latest
```

## Run

```sh
sitch                       # print the fetch with per-distro ASCII logo
sitch --no-ascii           # print without the logo column
sitch --json                # dump a JSON snapshot of the system
sitch -c examples/minimal.toml
```

Build from source:

```sh
go build -trimpath -ldflags='-s -w' -o sitch ./cmd/sitch
```

## Config

Sitch reads a TOML file. Pass it with `--config` / `-c`, or just run `sitch` once and it writes a default file to your config directory (`$XDG_CONFIG_HOME/sitch/config.toml`, `~/Library/Application Support/sitch/config.toml`, or `%AppData%\sitch\config.toml` depending on OS). Edit it and run `sitch` again.

Top-level keys (each has a matching flag where one exists; see the [Flag reference](#flag-reference) below):

- `format` (`"terminal"` or `"json"`). Default: `terminal`. CLI: `--json` / `-j`.
- `color_mode` (`"charmtone"`, `"tty"`, or `"custom"`). Default: `charmtone`. CLI: `--color`.
- `ascii` (`true` or `false`). Default: `true`. Set to `false` to disable the per-distro ASCII logo. CLI: `--no-ascii` / `-a`.
- `logo_position` (`"left"`, `"right"`, `"top"`, `"bottom"`). Default: `"left"`. Where the ASCII logo sits relative to the Bento grid. `top` and `bottom` stack the logo above/below the grid; `left` and `right` place it in a side column. No CLI flag; set in TOML.
- `logo_justify` (`"top"`, `"middle"`, `"bottom"`). Default: `"top"`. Vertical alignment of the logo against the grid when the logo is in a side column. Only meaningful when the logo is taller than the grid; if the logo is shorter, the column simply pads with blank lines per the chosen justify. No CLI flag; set in TOML.
- `logo_size` (`"regular"` or `"small"`). Default: `"regular"`. `"small"` uses the fastfetch small variant if one exists for the distro (note: small variants use newer Unicode block characters that may not render in all terminals). No CLI flag; set in TOML.
- `truncate` (`true` or `false`). Default: `false`. When `true`, clips the logo vertically so the body never grows taller than the (header + grid) — useful when the logo is much taller than the data block. CLI: `--truncate`.
- `logo` (`"<id>"`). Default: empty (auto-detect from the host distro). Force a specific bundled logo by id. `logo = "arch"` always shows the Arch art; `logo = "nixos_small"` shows the small NixOS variant. Case-insensitive. Unknown ids cause a load error that lists the available ids. CLI: `--logo <id>`.
- `logo_file` (`"<path>"`). Default: empty. Use a custom ASCII art file instead of any bundled logo. Takes precedence over `logo` and over auto-detection. Errors out at load if the path is missing or unreadable; errors at render if the file is empty. CLI: `--logo-file <path>`.
- `footer_align` (`"full"` or `"grid"`). Default: `"full"`. Controls how the bottom user/confetti footer aligns. The footer's right edge always lines up with the grid's right edge regardless of logo position; `full` lets the footer span the full composed width when the logo is on top or bottom, while `grid` keeps it grid-width in all cases. No CLI flag; set in TOML.
- `rows`: list of rows. Each row is a list of 1 to 3 spec keys (see below). Default: the layout from `config.Default()`. No CLI flag.

## Flag reference

Spec keys: `os`, `host`, `disk`, `memory`, `kernel`, `uptime`, `gpu`, `igpu`, `cpu`, `packages`, `shell`, `display`, `desktop`, `terminal`, `os-age`, `locale`. Anything else fails to load.

Color modes:

- `charmtone`: the default palette. Use this unless you need ANSI 16 or your own colors.
- `tty`: ANSI 16 colors. Picked for terminals without truecolor. The `Sitch` wordmark gets a blue background.
- `custom`: you supply only the colors you want to style. With `color_mode = "custom"` you may define `colors.border`, `colors.background`, `colors.header`, `colors.value`, `colors.sitch`, and one `colors.titles.<spec>` per spec. Empty or missing values stay transparent and use your terminal's default styling. Non-empty invalid colors are rejected.

Flags override TOML where it makes sense. Every flag below is also accepted in the config file under the same name (without the leading `--`).

| Flag | Short | TOML key | Type | Default | Description |
| --- | --- | --- | --- | --- | --- |
| `--fetch` | `-f` | — | bool | `true` | Print the fetch (default behavior; the flag exists for explicitness). |
| `--json` | `-j` | `format = "json"` | bool | `false` | Force JSON output, ignoring the TOML `format`. |
| `--config <path>` | `-c` | — | string | user config dir | Read a specific TOML file instead of the default. |
| `--color <mode>` | — | `color_mode` | `charmtone` \| `tty` | unset | Switch the palette for one run. Cannot pick `custom` because the palette lives in the file. |
| `--no-ascii` | `-a` | `ascii = false` | bool | `false` | Hide the per-distro ASCII logo column; show only the Bento grid. |
| `--truncate` | — | `truncate = true` | bool | `false` | Clip the logo vertically so the body never grows taller than the (header + grid). |
| `--logo <id>` | — | `logo = "<id>"` | string | empty | Show the bundled logo for `<id>` instead of the detected distro. Case-insensitive. |
| `--logo-file <path>` | — | `logo_file = "<path>"` | string | empty | Read ASCII art from a file. Overrides `--logo` and auto-detection. |

## Examples

`examples/` has ready-to-use configs. `minimal.toml` is the one charmtone example and `tty.toml` is the one tty example. Everything else is `custom` with a different well-known palette:

- `minimal.toml`: one column, charmtone.
- `no-ascii.toml`: two-column layout, logo hidden (ascii = false).
- `ascii-minimal.toml`: one-column, logo on the left, charmtone.
- `logo-right.toml`: logo on the right of the grid.
- `logo-top.toml`: logo above the grid.
- `logo-bottom.toml`: logo below the grid.
- `logo-small.toml`: small logo variant with truncate enabled.
- `logo-override.toml`: forces the Arch bundled logo via `logo = "arch"`.
- `footer-grid.toml`: footer aligned to the grid box instead of full width.
- `tty.toml`: ANSI 16-color palette.
- `1-2-3-1.toml`: Catppuccin Mocha.
- `1-3-2-1.toml`: Gruvbox Dark.
- `2-3-1-2.toml`: Nord.
- `3-1-2-3.toml`: Tokyo Night.
- `3-2-1-3.toml`: Dracula.
- `all-rows.toml`: Kanagawa.
- `one-per-row.toml`: Solarized Dark.
- `three-columns.toml`: Atom One Dark.
- `custom.toml`: Rose Pine.
- `json.toml`: Monokai, JSON output.

Try one:

```sh
sitch -c examples/3-1-2-3.toml
```

## Nix

NixOS package totals are exact, not estimated. Sitch scans `/run/current-system`, `~/.nix-profile`, the XDG state profile, and `/etc/profiles/per-user/<user>`. The first run shells out to `nix-store --query --requisites` for each profile in parallel, which is slow. The result is cached at `$XDG_CACHE_HOME/sitch/nix-packages.json`, keyed by the resolved paths of every profile, so later runs are instant until you change a generation or move a profile.

## Demo

The demo GIF is built with [VHS](https://github.com/charmbracelet/vhs):

```sh
go install github.com/charmbracelet/vhs@latest
vhs tapes/preview.tape
```

Licensed under [MIT](./LICENSE)
