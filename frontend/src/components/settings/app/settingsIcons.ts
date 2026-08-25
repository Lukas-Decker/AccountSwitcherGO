/*
  Card icons for the settings grid.

  Each entry is the path data for one 24x24 outline icon, drawn with a stroke
  rather than a fill so the shapes stay legible at the ~11px they render at.
  Circles are written as two arcs because the card renders paths only.
*/

export type SettingsIcon = readonly string[];

/** Circle as path data, so an icon needs no extra element types. */
function circle(cx: number, cy: number, r: number): string {
  return `M${cx - r} ${cy}a${r} ${r} 0 10${r * 2} 0a${r} ${r} 0 10${-r * 2} 0`;
}

export const settingsIcons = {
  /** Half-filled circle: the usual light/dark theme mark. */
  appearance: [circle(12, 12, 9), "M12 3v18", "M12 7h5", "M12 12h8", "M12 17h5"],
  /** Globe. */
  language: [circle(12, 12, 9), "M3 12h18", "M12 3a14 14 0 010 18a14 14 0 010-18"],
  /** A pointer beside a stack of menu rows. */
  contextMenu: ["M4 5h10", "M4 9h10", "M4 13h6", "M14 12l6 6-2.5.5L19 21l-2 1-1.5-2.5L14 21z"],
  /** Gamepad. */
  games: [
    "M7 12h4M9 10v4",
    "M6 8h12a4 4 0 014 4v0a4 4 0 01-4 4h-1l-2-2H9l-2 2H6a4 4 0 01-4-4v0a4 4 0 014-4z",
    "M16 11h.01",
    "M18 13h.01",
  ],
  /** Application window with a title bar. */
  window: ["M3 5h18v14H3z", "M3 9h18", "M6 7h.01", "M9 7h.01"],
  /** Eye: what other people can see of the app. */
  privacy: ["M2 12s3.8-6.5 10-6.5S22 12 22 12s-3.8 6.5-10 6.5S2 12 2 12z", circle(12, 12, 3)],
  /** Padlock. */
  security: ["M5 11h14v10H5z", "M8.5 11V7.5a3.5 3.5 0 017 0V11"],
  /** Keyboard. */
  input: ["M2 6h20v12H2z", "M6 10h.01", "M10 10h.01", "M14 10h.01", "M18 10h.01", "M7.5 14h9"],
  /** Speech bubble. */
  discord: ["M4 5h16v11H9l-5 4V5z", "M8 10h.01", "M12 10h.01", "M16 10h.01"],
  /** Stacked disks: where the user data lives. */
  data: [
    "M4 6c0-1.7 3.6-3 8-3s8 1.3 8 3-3.6 3-8 3-8-1.3-8-3z",
    "M4 6v6c0 1.7 3.6 3 8 3s8-1.3 8-3V6",
    "M4 12v6c0 1.7 3.6 3 8 3s8-1.3 8-3v-6",
  ],
  /** Download arrow onto a line. */
  updates: ["M12 3v11", "M7.5 10L12 14.5 16.5 10", "M4 20h16"],
  /** Sliders, for the settings that are not for everyone. */
  advanced: ["M4 7h5M13 7h7", "M4 12h9M17 12h3", "M4 17h3M11 17h9", "M11 5v4", "M15 10v4", "M9 15v4"],
} as const satisfies Record<string, SettingsIcon>;
