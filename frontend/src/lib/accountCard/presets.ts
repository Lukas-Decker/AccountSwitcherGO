import type {
  AccountCardConfig,
  CardBlockDisplay,
  CardBlockKind,
  CardLayout,
  CardRow,
  CardSizePreset,
} from "./types";

/** One block on its own line, which is how the card has always stacked. */
function line(kind: CardBlockKind, display?: CardBlockDisplay): CardRow {
  return { blocks: [{ kind, enabled: true, display }] };
}

/** Several blocks side by side. Only worth it when they are icon-sized. */
function metaRow(...blocks: { kind: CardBlockKind; display?: CardBlockDisplay }[]): CardRow {
  return { blocks: blocks.map((b) => ({ kind: b.kind, enabled: true, display: b.display })) };
}

/**
 * Today's card, exactly: 100px tracks, a 6em avatar, one block per line and no
 * icons anywhere. Existing installs migrate onto this, so upgrading changes
 * nothing they can see.
 */
const SMALL: CardLayout = {
  minWidth: 100,
  maxWidth: 120,
  minHeight: 135,
  avatarEm: 6,
  fontScale: 1,
  statusBadgeStyle: "border",
  rows: [
    line("avatar"),
    line("accountLogin"),
    line("displayName"),
    line("tags"),
    line("note"),
    line("gameStats"),
    line("platformId"),
    line("lastUsed"),
    line("statusLine"),
  ],
};

/**
 * Room for the things Small has to leave out. The last-used and status blocks
 * move onto one icon row rather than spending two full lines on text.
 */
const MEDIUM: CardLayout = {
  minWidth: 140,
  maxWidth: 170,
  minHeight: 175,
  avatarEm: 7.5,
  fontScale: 1.15,
  statusBadgeStyle: "corner",
  rows: [
    line("avatar"),
    line("displayName"),
    line("accountLogin"),
    line("tags"),
    line("note"),
    line("gameStats"),
    metaRow({ kind: "lastUsed", display: "iconText" }, { kind: "statusLine", display: "icon" }),
  ],
};

/**
 * Everything the platform offers, with the identifiers spelled out rather than
 * abbreviated: at this width there is space for them to be readable.
 */
const LARGE: CardLayout = {
  minWidth: 190,
  maxWidth: 230,
  minHeight: 225,
  avatarEm: 9,
  fontScale: 1.3,
  statusBadgeStyle: "corner",
  rows: [
    line("avatar"),
    line("displayName"),
    line("accountLogin"),
    line("platformId"),
    line("tags"),
    line("note"),
    line("gameStats"),
    metaRow({ kind: "lastUsed", display: "iconText" }, { kind: "statusLine", display: "iconText" }),
    line("badges"),
  ],
};

const PRESETS: Record<Exclude<CardSizePreset, "custom">, CardLayout> = {
  small: SMALL,
  medium: MEDIUM,
  large: LARGE,
};

/**
 * Presets are frozen values in code, never stored. Only the preset's name is
 * persisted, so refinements ship with an update instead of being pinned to
 * whatever a config file was written with.
 */
export function presetLayout(preset: CardSizePreset): CardLayout {
  if (preset === "custom") return PRESETS.small;
  return PRESETS[preset];
}

export const DEFAULT_LAYOUT = SMALL;

export const DEFAULT_CARD_CONFIG: AccountCardConfig = {
  version: 1,
  preset: "small",
};

/** A starting point for Custom: whatever the user was already looking at. */
export function seedCustomLayout(from: CardSizePreset): CardLayout {
  const base = presetLayout(from);
  return {
    ...base,
    rows: base.rows.map((row) => ({ blocks: row.blocks.map((b) => ({ ...b })) })),
  };
}
