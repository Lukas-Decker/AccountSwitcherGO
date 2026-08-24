# Branding masters

`AccountSwitcher.svg` is the master. Every shipped icon is derived from it, and
it is the one file here that cannot be recovered from anything else in the
repository.

## Regenerating

From the repository root:

```bash
go run ./tools/genicons
```

That rewrites `build/appicon.png`, `build/trayicon.png`,
`build/trayicon-darkmode.png`, `build/branding/AccountSwitcher.ico` and
`frontend/public/img/favicon.png`. To see what has drifted without writing
anything:

```bash
go run ./tools/genicons -check
```

`build/windows/icon.ico` and `build/darwin/icons.icns` are **not** produced by
that tool. The build derives them from `build/appicon.png` through
`wails3 task common:generate:icons`, so they have exactly one source and
`-check` does not fight the build over them.

## Framing

The master's own 120x120 canvas leaves uneven margins around the drawing, which
at 16 pixels reads as an icon sitting slightly off centre. `genicons` shifts the
artwork onto a square canvas so it is centred, which is what the icon set is cut
from. `frontend/public/img/AccountSwitcherLogo.svg` repeats the same framing by
hand so the title bar and the taskbar show the same mark.

## Colour

The rear figure is `#201e1d`, very nearly black. That is correct on the light
surfaces an installer and a file listing use, and close to invisible on a dark
taskbar, where the mark would read as one blue shape with a bite out of it.

Two derived forms exist for that reason, and both keep the brand blue untouched:

- `build/trayicon-darkmode.png` redraws the rear figure in a light neutral. Wails
  selects it over `trayicon.png` when the system is in dark mode.
- `frontend/public/img/AccountSwitcherLogo.svg` gives the rear figure no fill at
  all, so it inherits the host element's colour. Every theme already sets that to
  its own foreground, so the logo follows the theme without a second file.

## Legacy files

| File | Purpose |
| --- | --- |
| `AccountSwitcher.png` | Full-resolution original of the previous artwork |
| `AccountSwitcherFitted.png` | The previous artwork scaled to touch the frame |

These belong to the icon set that shipped before this one. They are kept because
a master is the one thing that cannot be reconstructed once it is gone, and
nothing in the build reads them.
