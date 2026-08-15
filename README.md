# Account Switcher

**A super-fast, open-source account switcher for Steam, Battle.net, Epic Games, Origin, Riot Games, Ubisoft, and more.**

**Saves NO passwords** or any user information (besides what you choose to enter). Most switchers, including Steam, work purely off changing a file and a few registry keys.
*Wastes no time closing, switching and restarting Steam and other platforms.*

**NOTE:** Not created for cheating purposes. All it does is change accounts. Use it as you see fit, accepting responsibility.

## How does it work?

It swaps out files and registry values that point to your last logged-in account while the program is closed. Think of it as freezing a platform like Steam in time, and replacing the "account block" with a previously frozen "account block", then unfreezing it. Swapping the account block lets the program avoid interacting with passwords or 2-factor, so you can "skip" both of those in the login process.

You can see (and edit) how account switching works by checking `Platforms.json`.

## Platforms

Albion Online, Battle.net, Discord (+ PTB & Canary), Epic Games, EA Desktop, Escape from Tarkov, GeForce Now, GOG Galaxy, Genshin Impact, Honkai StarRail, Magic Arena, Origin, OBS Studio, Oculus, PS Remote Play, Riot Games (Valorant, League...), Rockstar, Steam, Ubisoft Connect, and more.

This list can be extended in a simple text file, `Platforms.json`.

## Features

- Fully user/community-customisable **theme system** with several themes built in.
- **Streamer mode** to hide SteamIDs and more while streaming software is running (e.g. OBS, XSplit).
- **Automatic updates** using a patch system, so only a few KB/MB are downloaded at a time.
- **Steam:** log in as Invisible, Offline and more. Copy profile links, SteamID and VAC info, and create quick-switch desktop shortcuts.
- **Easily add & customise new platforms.**
- **Control via tray** for quick switching without keeping everything open.
- **Protocol support** to build access via other tools (`accswitcher:\<command>`).

## Installation

1. Download and run the installer, or
2. Download and run `Account-Switcher.exe` directly for a portable install.

## Building from source

The app is Go on the back end and Svelte on the front end, tied together by [Wails 3](https://v3.wails.io/). Builds are driven by [Task](https://taskfile.dev/), which ships inside the Wails CLI as `wails3 task`.

### Prerequisites

- [Go](https://go.dev/dl/) 1.26 or newer
- [Node.js](https://nodejs.org/) with pnpm (`npm install -g pnpm`)
- The Wails CLI: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.6`
- Windows: the WebView2 runtime, which is already present on Windows 10 and 11
- Only for the installer: [NSIS](https://nsis.sourceforge.io/), with `makensis` on your PATH
- Only for the scripts in `tools/`: Python 3

`wails3 doctor` reports what it can find and what is missing.

### Build

```bash
wails3 task build
```

That one task installs the frontend dependencies, generates the Go bindings, builds the frontend and compiles the binary to `bin/Account-Switcher.exe`. Nothing needs to be run in `frontend/` by hand.

### Other tasks

| Command | Result |
| --- | --- |
| `wails3 task dev` | Runs the app with the frontend hot-reloading |
| `wails3 task run` | Runs the binary that was last built |
| `wails3 task test` | Runs the Go and frontend test suites |
| `wails3 task package` | Builds the NSIS installer. `FORMAT=msix` builds an MSIX package instead |
| `wails3 task dist` | Stages a shippable build in `dist/`, with a zip beside it (Windows) |
| `wails3 task build ARCH=arm64` | Cross-compiles for another architecture |
| `wails3 task version` | Prints the version from `build/config.yml` |
| `wails3 task version:bump -- minor` | Bumps the version there and in the Windows resources. Also takes `patch`, `major` or an explicit `x.y.z` |

### Versioning

`build/config.yml` holds the version the app reports. It is embedded at compile time, so the About row, the crash reports and the update check all read from it. `build/windows/info.json` carries its own copy for the exe's file properties, which is why the bump goes through `wails3 task version:bump` rather than an editor. `wails3 task version:check` fails if the two have drifted apart.

## Disclaimer

```
All trademarks and materials are the property of their respective owners and their licensors. This project is not affiliated
with any companies referenced. This is not "official" software or related to any companies mentioned. All it does is let you
move your files around on your computer the same way you can. The use of names, icons and trademarks does not indicate
endorsement of the trademark holder by this project or its creators, nor vice versa. They are only used to visually indicate
which programs this project interacts with easily to the end-user.

By enabling optional features that scrape the web for publicly available information (such as limited game/profile statistics
and other data), you understand and accept full responsibility for doing so on your own volition. If you appreciate accurate
information, support the services providing it directly. The information collected is incredibly limited and is no replacement
or competitor for sites scraped.

The authors are not responsible for the contents of external links.
For the rest of the disclaimer, refer to the License (GNU General Public License v3.0) file (LICENSE) - see sections 15, 16 and 17.
```

## License

Licensed under the GNU General Public License v3.0. See [LICENSE](LICENSE).
