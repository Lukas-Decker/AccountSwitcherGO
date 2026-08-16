/*
  Icons for card blocks that can draw as a mark instead of a sentence.

  Same convention as the settings grid: 24x24 outline paths, stroked rather
  than filled so they stay legible at the ~11px a card renders them at, and
  drawn in currentColor so every theme gets them right without a variant.
*/

export type CardIcon = readonly string[];

export const cardIcons = {
  /** Clock: when the account was last used. */
  lastUsed: ["M12 3a9 9 0 100 18a9 9 0 100-18", "M12 7v5l3 2"],
  /** Warning triangle: something went wrong syncing the account. */
  statusError: ["M12 4 2.5 20h19z", "M12 10v4", "M12 17h.01"],
  /** Arrows chasing each other: the account is still being refreshed. */
  statusPending: ["M20 12a8 8 0 01-8 8", "M4 12a8 8 0 018-8", "M16 4l4 0 0 4", "M8 20l-4 0 0-4"],
} as const satisfies Record<string, CardIcon>;
