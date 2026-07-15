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
