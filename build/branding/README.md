# Branding masters

The originals every shipped icon is derived from. Nothing in the build reads
these files: the generated assets are committed alongside the code that uses
them (`build/appicon.png`, `build/trayicon.png`, `build/windows/icon.ico`, and
the logos under `frontend/public`).

They are kept because regenerating from the master is not the same as scaling a
derived copy up again, and a master is the one thing that cannot be recovered
from the repository once it is gone.

| File | Purpose |
| --- | --- |
| `AccountSwitcher.png` | Full-resolution original |
| `AccountSwitcherFitted.png` | Same artwork scaled to touch the frame, which is what the icon set was cut from |
| `AccountSwitcher.ico` | Multi-size icon supplied with the artwork |

The alpha channel is real and load-bearing: the corners are fully transparent and
the artwork is mostly transparent by area. Anything that flattens it onto a
background, including a resize that is not alpha-premultiplied, produces a halo
at small sizes.
