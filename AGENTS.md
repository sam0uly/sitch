# AGENTS.md

## Project

Sitch is a fast, colorful system fetch (think `neofetch`) written in Go, rendered with the Charm stack (lipgloss, charmtone, fang, cobra). Module path: `samouly.fun/sitch`.

## Commands

Build:
```
go build -trimpath -ldflags='-s -w' -o sitch ./cmd/sitch
```

Run:
```
go run ./cmd/sitch                     # default terminal output
go run ./cmd/sitch --json              # JSON dump of system.Info
go run ./cmd/sitch -c examples/minimal.toml
```

Test (CI uses `-race`):
```
go test -race ./...
go vet ./...
```

Release is driven by GoReleaser v2 (`.goreleaser.yaml`); CI releases on tags matching `v*` only. Build flags used by GoReleaser: `-trimpath` plus ldflags `-s -w -X main.version={{.Version}} -X main.commit={{.Commit}}`.

No Makefile, no task runner. VHS (`tapes/*.tape`) is used only to regenerate demo GIFs, not part of the normal dev loop.

## Architecture

Three internal packages, single binary entry point:

- `cmd/sitch/main.go` is a cobra command wired through `charm.land/fang/v2` (not raw cobra). Flags: `--no-ascii/-a` (hide distro logo), `--json/-j`, `--config/-c`, `--color` (charmtone/tty; `custom` must be in TOML), `--truncate` (clip the logo vertically so the body never grows taller than the header+grid), `--logo <id>` (force a specific bundled logo id; case-insensitive), `--logo-file <path>` (read custom ASCII art from a file; overrides `--logo` and auto-detection), `--fetch/-f` (default behavior). `--color` is merged into the loaded config after parsing, so it cannot pick `custom`. The `Options.ASCII` field controls logo visibility (defaults to `true`; set to `false` when `--no-ascii` is passed). `Options.LogoPosition` is one of `left`/`right`/`top`/`bottom` (default `left`); `Options.LogoJustify` is `top`/`middle`/`bottom` (default `top`). `Options.LogoSize` is `regular`/`small` (default `regular`); `small` uses the fastfetch `_small` variant if one exists. `Options.Logo` and `Options.LogoFile` are the bundled-id and custom-file overrides; precedence is `LogoFile > Logo > auto-detect`. `Options.Truncate` is a bool (default `false`); when true, caps the logo line count to the grid's line count. `Options.FooterAlign` is `full`/`grid` (default `full`).
- `internal/config` is the TOML loader (BurntSushi/toml). Validates `format in {terminal, json}`, `color_mode in {charmtone, tty, custom}`, `logo_position in {left, right, top, bottom}`, `logo_justify in {top, middle, bottom}`, `logo_size in {regular, small}`, `footer_align in {full, grid}`, row count (1 to 3 columns), and that every spec in `rows` is a known key. For `color_mode = "custom"` it validates the format of any non-empty color: empty values are accepted and rendered transparent (using the terminal default styling). Invalid non-empty values cause a load error that prints the invalid key names plus `CustomColorExample()` so users have a pasteable block. When `logo` is set, the id is checked against the bundled set (`render.LogoNames()`); unknown ids error with the list of available ids. When `logo_file` is set, the path is `os.Stat`-ed at load time so missing files fail fast.
- `internal/system` is the platform-specific collection of `Info`. Two build tags split reality:
  - `platform_linux.go`, `metrics.go`, `gpu_linux.go`, `wm_linux.go` are the real implementations (Linux only).
  - `platform_other.go`, `metrics_other.go`, `gpu_other.go` are no-op/stub fallbacks for `darwin`/`windows`. Most fields return `"unknown"`, and there is no `osRelease`, no `packageCounts`, no `gpuModels`.
  - `system.go` and `packages.go` have no build tags and are shared. Shared helpers: `readFileOrEmpty`, `readKeyValueFile`, `kernelVersion`, `cpuModel`, `cleanCPUModel`, `gpuModel` (unused on darwin, leftover), `baseName`, `firstNonEmpty`, `desktopEnvironment`, `displayEnvironment`.
- `internal/render` turns `Info` into a Bento-style grid with an optional per-distro ASCII logo block. `facts.go` defines 16 keyed facts in a fixed `factOrder`. `logos.go` is a generated file (rebuilt by `cmd/gen_logos/main.go` from the `logos_fastfetch/*.txt` source files) containing the full fastfetch ASCII art corpus (94 distro logos) with `$1..$9` placeholder colors that get resolved against the active `titleColors` palette at render time. `logo.go` provides `renderLogoBox`, `pickLogo`, `lookupLogo` (case-insensitive id lookup), `LogoNames()` (sorted id list used by config validation), and `expandColorPlaceholders`. `theme.go` owns the `lipgloss.Style` globals (`sitchStyle`, `headerStyle`, `valueStyle`, `borderStyle`, `surfaceStyle`, `titleColors`) and three palettes (`charmtoneTitles`, `ttyTitles`, plus custom from config); `setPalette` is the single mutator and is called from `init` and at the top of `PrintWithOptions`. `render.go` is the layout engine. `composeWithLogo` routes to `composeSide` (logo on the left or right of the grid) or `composeStacked` (logo on top or bottom of the grid) based on `Options.LogoPosition`. In `composeSide` the header+grid+footer block stays contiguous as one unit; whichever column is shorter per `logo_justify` gets blank filler above/below so the footer is never cut off and `logo_justify="middle"` vertically centers the logo when it is taller than the block. `renderGridConfigured` is the configured-rows path actually used; `renderGrid` and `terminalWidth` are dead code (gopls `unusedfunc`).

Default config location: `$XDG_CONFIG_HOME/sitch/config.toml` (Go `os.UserConfigDir()`, which gives `~/Library/Application Support/sitch/config.toml` on macOS, `%AppData%\sitch\config.toml` on Windows). On first launch with no `--config` flag, Sitch writes `DefaultTOML()` to that path so the user can find and edit it; subsequent runs leave the file alone. Explicit `-c` paths that don't exist do not trigger a write.

## Output flow

`main.go` -> `config.Load` -> `system.Collect` -> either `json.Encode` or `render.PrintWithOptions` (which calls `setPalette`, lays out the grid, renders the footer, paints the background surface via lipgloss, and prints through `lipgloss.Println`).

## Spec keys (16)

`os, host, disk, memory, kernel, uptime, gpu, igpu, cpu, packages, shell, display, desktop, terminal, os-age, locale`. Lowercased on load. These are the same keys used as TOML `colors.titles` entries in custom mode.

## Color modes

- `charmtone` is the default. Uses `charmtone` palette for chrome and 17 hand-picked `charmtoneTitles` accents.
- `tty` uses the ANSI 16-color palette. The `Sitch` wordmark gets a blue (ANSI 4) background.
- `custom` is the user hex palette. Missing custom entries render transparent without styling; invalid non-empty entries produce one error that names the offending keys.

## Nix specifics (packages.go)

- Profiles scanned: `/run/current-system`, `$HOME/.nix-profile`, `$XDG_STATE_HOME/nix/profile` (falls back to `$HOME/.local/state/nix/profile`), and `/etc/profiles/per-user/<user>`.
- Total Nix counts are split: system = profile 0, user = sum of profiles 1+2+3. `total` is intentionally `0` so the normal `info.Packages = strconv.Itoa(total)` branch is skipped.
- `info.Packages` for Nix becomes `"<n> system / <n> user"`. If both are zero, packages read `"unknown"`.
- Cached at `$XDG_CACHE_HOME/sitch/nix-packages.json`. Cache key is `profile1=<resolved1>|profile2=<resolved2>|...` (resolved via `filepath.EvalSymlinks`; missing profiles encode as `missing`). Cache is rewritten atomically via `path + ".tmp"` -> rename, mode `0o600`, dir `0o700`. Empty cache directory is created on demand.
- A cold count runs `nix-store --query --requisites <profile>` for each profile concurrently (4 goroutines) with a 5s context timeout from the caller. The caller in `platform_linux.go` wraps hardware probing in a separate 2s timeout.
- Path filter (`isNixPackagePath`): looks like `/nix/store/<32-hex>-<name>-<ver>`, excludes suffixes `-doc -man -info -dev -bin` and `nixos-system-nixos-*`, requires a `digit.digit` substring in the name.

## Wayland compositor detection (`wm_linux.go`)

If `WAYLAND_DISPLAY` is set, walk `/proc/*/fd` looking for the socket whose inode matches the socket listed in `/proc/net/unix` for that display path (resolved against `$XDG_RUNTIME_DIR` or `/run/user/<uid>`). The owning process's `cmdline[0]` (or `/proc/<pid>/exe` basename) is reported. Only consulted when `desktopEnvironment()` finds no `XDG_SESSION_DESKTOP`/`XDG_CURRENT_DESKTOP`.

## GPU detection (Linux)

`gpuModels` tries `sysfsGPUModels` first (matches `/sys/class/drm/card*/device/uevent` PCI IDs against `/usr/share/{hwdata,misc,pci}.ids` or `pci.ids` next to a resolved `lspci` path). Falls back to `lspciGPUModels` which greps `lspci -nn` for `vga compatible controller` / `3d controller`. `discreteGPUModels` strips the model identified as integrated; integration is detected by substring: `integrated`, `uhd`, `iris`, `vega`, `apu`. `info.GPU` is the comma-joined discrete list, or `"unknown"`.

## Tests

Three test files: `internal/config/config_test.go`, `internal/system/system_test.go`, `internal/render/render_test.go`, `internal/render/theme_test.go`. Pattern is plain stdlib `testing` with `t.TempDir()` for filesystem cases.

- `render/render_test.go` includes `TestConfiguredRowsKeepDynamicBordersAligned` which enumerates all 3^6 = 729 row shape patterns and asserts equal line widths and bottom corners on each grid. This is the layout invariant test.
- `render/logo_test.go` includes `TestPickLogoKnownDistro`, `TestPickLogoFallback`, `TestPickLogoSmall`, `TestPickLogoNameOverride`, `TestPickLogoCustomText`, `TestLookupLogo`, `TestLogoNames`, `TestRenderLogoBoxWidth`, `TestRenderLogoBoxHonorsTruncateHeight`, `TestComposeWithLogoFooterAlignment`, `TestComposeWithLogoAllPositions`, `TestComposeWithLogoShortGrid`, and `TestExpandColorPlaceholders`.
- `system/system_test.go` includes `TestFormatUptime` (minutes/hours/days/invalid) and `TestKernelVersion`. One subtest (`osrelease fallback`) `t.Skip`s when `/proc/sys/kernel/osrelease` is readable on the host.

## Conventions

- Go 1.26 module; `gopls` reports several `unusedfunc` warnings on darwin builds because the linux files aren't compiled there. These are intentional and not actionable.
- All user-facing strings (icons, labels) live in `internal/render/facts.go`. Spec keys are lowercase; `render` lowercases anything the user puts in TOML.
- Use `slices`, `strings.SplitSeq`, `os.Setenv`/`os.Getenv` directly; standard library Go.
- JSON tags on `system.Info` are stable; `system` package is the public surface that `render` consumes.
- `setPalette` is the only place lipgloss styles are assigned. Don't read or set the package globals elsewhere. Render layout assumes they are populated.

## Gotchas

- `--color` only accepts `charmtone` or `tty`. To use `custom`, declare `color_mode = "custom"` in the TOML. The CLI flag is intentionally non-override for custom because the palette lives in config.
- `CustomColorExample()` returns a hardcoded string used in both the example file `examples/custom.toml` and the error message. Keep them in sync if you add a spec.
- `format = "json"` ignores the row layout but still parses `rows` (validation still runs); the rows are unused at runtime.
- The non-Linux platform files (`platform_other.go`, `metrics_other.go`, `gpu_other.go`) intentionally provide no-op signatures so the package compiles cross-platform. Don't delete them.
- `gpuModel` in `system.go` is dead code (replaced by `gpuModels` in `gpu_linux.go`); gopls flags it on non-Linux builds. Safe to delete if you also drop the linux-only caller. Currently none.
- `renderGrid` and `terminalWidth` in `render/render.go` are dead. Layout went all-configurable. Don't write new callers to them.
- Nix package cache key includes every profile's symlink-resolved path. Moving a profile or recreating a generation invalidates the cache by design.
- `/etc/os-release` is required on Linux. `collectPlatform` returns an error if missing. Don't swallow it.
- The binary in repo (`sitch`, `sitch.exe`) and the GIFs are gitignored build artifacts.
