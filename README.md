<div align="center">

<img src="build/appicon.png" alt="Ports icon" width="96"/>

# Ports

**See every app holding a port on Windows and WSL — kill it in one click.**

A lightweight desktop app for developers who juggle services on both sides of the Windows / WSL boundary.

[Download](#install) · [Build from source](#build) · [How it works](#how-it-works)

</div>

---

## Why

Finding which process is squatting on port 3000 means running `netstat -ano`, digging through `tasklist`, and repeating the whole dance inside WSL — where the PID namespaces don't even match. Ports does it in one window:

- **Windows and WSL side by side** — every running distro is scanned automatically
- **Grouped by program** — one card per app with all its ports; expand to see individual PIDs
- **Kill with one click** — no more `taskkill /F /PID 12345` archaeology
- **Jump to the binary** — "open containing folder" works for Windows paths *and* WSL paths (via `\\wsl.localhost`)

## Features

| | |
|---|---|
| Unified scan | Listening TCP + bound UDP sockets on Windows and every running WSL distro |
| Live refresh | Auto-refresh with a configurable interval (2 s and up) |
| Grouped view | Processes grouped by program name; expand for per-PID detail |
| Kill | `taskkill /F /T` on Windows, `kill -9` (root) inside the distro |
| Open folder | Reveal the executable in Explorer, including `\\wsl.localhost` paths |
| System tray | Close to tray, left-click to restore, optional start-on-boot |
| Filters | Search by name, port, PID or path; per-source chips (All / Windows / WSL) |
| Clean UI | Frameless dark UI, no terminal windows ever flash |

WSL infrastructure processes (`wslhost`, `wslrelay`) are hidden automatically.

## Install

Grab the latest release from [Releases](../../releases):

- **`Ports-vX.Y.Z-windows-amd64-installer.exe`** — standard installer with Start Menu / Desktop shortcuts and an uninstaller
- **`Ports-vX.Y.Z-windows-amd64-portable.zip`** — portable ZIP, just extract and run `Ports.exe`

Requirements: Windows 10/11 (WebView2 is auto-installed by the installer if missing) and WSL2 for WSL process visibility.

## Build

Requires [Go 1.23+](https://go.dev), [Node 18+](https://nodejs.org), and the [Wails CLI](https://wails.io/docs/gettingstarted/installation):

```sh
wails build
```

Output lands in `build/bin/ports.exe`. To also produce the NSIS installer:

```sh
wails build -nsis
```

For live development:

```sh
wails dev
```

## How it works

| Source | Enumerate | Kill | Path lookup |
|---|---|---|---|
| Windows | `netstat -ano` + `tasklist` + `QueryFullProcessImageName` | `taskkill /F /T /PID` | Win32 API |
| WSL | `wsl -u root ss -tulpn` (netstat fallback) | `wsl -u root kill -9` | `readlink /proc/<pid>/exe` |

All child processes are spawned with `HideWindow`, so the app never flashes a console. WSL paths are resolved in batches of 10 PIDs per round-trip to stay fast.

## Settings

Stored in `%APPDATA%\ports\settings.json`:

| Key | Default | Description |
|---|---|---|
| `autoRefresh` | `true` | Periodic rescans |
| `intervalSecs` | `5` | Seconds between scans (min 2) |
| `showPaths` | `true` | Show executable paths |
| `showSystem` | `true` | Include system processes |
| `closeAction` | `ask` | What the X button does: `ask` / `hide` / `quit` |
| `startOnBoot` | `false` | Start hidden in the tray at login |

---

<div align="center">

Made by **[Ainz](https://ainz.uk)**

</div>
