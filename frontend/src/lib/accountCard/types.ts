/**
 * The account card's configuration model.
 *
 * Lives here rather than beside the components so both the renderer and the
 * settings UI can depend on it without either depending on the other.
 */

/** The pieces an account card can be built from. */
export type CardBlockKind =
  | "avatar"
  | "accountLogin"
  | "displayName"
  | "tags"
  | "note"
  | "gameStats"
  | "platformId"
  | "lastUsed"
  | "statusLine"
  | "badges";

/**
 * How a block draws itself. Icon modes exist because the card is narrow: a
 * clock and "2h" survive at the small size where "2 hours ago" is ellipsed
 * into nothing.
 */
export type CardBlockDisplay = "text" | "icon" | "iconText";

/**
 * Where account warnings (VAC, limited) are shown. All three are offered
 * because which one reads best depends on the card size and on whether the
 * user cares about the flags at a glance.
 */
export type StatusBadgeStyle = "border" | "corner" | "block";

export type CardSizePreset = "small" | "medium" | "large" | "custom";

export interface CardBlockConfig {
  kind: CardBlockKind;
  enabled: boolean;
  /** Absent means the block's own default. */
  display?: CardBlockDisplay;
}

/**
 * One line of the card. A row with a single block is an ordinary stacked line;
 * several blocks in a row sit side by side, which is what makes icon mode pay
 * off rather than just making each line shorter.
 */
export interface CardRow {
  blocks: CardBlockConfig[];
}

export interface CardLayout {
  /** Card width is fluid between these, so tracks absorb leftover row space. */
  minWidth: number;
  maxWidth: number;
  minHeight: number;
  /**
   * In em, not px, because that is what the card has always used: the avatar
   * then scales with the app's font size instead of pinning itself to one.
   */
  avatarEm: number;
  rows: CardRow[];
  statusBadgeStyle: StatusBadgeStyle;
}

export interface AccountCardConfig {
  version: 1;
  preset: CardSizePreset;
  /**
   * Explicit user decisions layered over the active preset's defaults. Absent
   * keys keep whatever the preset says, so switching preset keeps deliberate
   * choices and picks up defaults for everything untouched.
   */
  blocks?: Partial<Record<CardBlockKind, boolean>>;
  displays?: Partial<Record<CardBlockKind, CardBlockDisplay>>;
  statusBadgeStyle?: StatusBadgeStyle;
  /** Only meaningful when preset is "custom". */
  custom?: CardLayout;
}

export const CARD_CONFIG_VERSION = 1;

/** Blocks every platform can draw. Anything else must be declared by an adapter. */
export const CORE_BLOCK_KINDS: readonly CardBlockKind[] = [
  "avatar",
  "displayName",
  "tags",
  "note",
  "gameStats",
  "lastUsed",
];

export const ALL_BLOCK_KINDS: readonly CardBlockKind[] = [
  "avatar",
  "accountLogin",
  "displayName",
  "tags",
  "note",
  "gameStats",
  "platformId",
  "lastUsed",
  "statusLine",
  "badges",
];

/** Bounds for custom layouts, so a typo cannot produce an unusable grid. */
export const CARD_BOUNDS = {
  minWidth: { min: 72, max: 320 },
  maxWidth: { min: 72, max: 320 },
  minHeight: { min: 80, max: 400 },
  avatarEm: { min: 1.5, max: 12 },
} as const;

function clamp(value: number, lo: number, hi: number): number {
  if (!Number.isFinite(value)) return lo;
  return Math.min(hi, Math.max(lo, Math.round(value)));
}

/** Same clamp, but keeps the fraction that em sizes need. */
function clampEm(value: number, lo: number, hi: number): number {
  if (!Number.isFinite(value)) return lo;
  return Math.min(hi, Math.max(lo, Math.round(value * 100) / 100));
}

function isBlockKind(value: unknown): value is CardBlockKind {
  return typeof value === "string" && (ALL_BLOCK_KINDS as readonly string[]).includes(value);
}

function isDisplay(value: unknown): value is CardBlockDisplay {
  return value === "text" || value === "icon" || value === "iconText";
}

function isBadgeStyle(value: unknown): value is StatusBadgeStyle {
  return value === "border" || value === "corner" || value === "block";
}

function isPreset(value: unknown): value is CardSizePreset {
  return value === "small" || value === "medium" || value === "large" || value === "custom";
}

/**
 * Brings a stored layout into range and drops anything unrecognised, keeping
 * the rest. A layout written by a newer build should degrade to something
 * usable rather than be thrown away.
 */
export function validateLayout(raw: unknown, fallback: CardLayout): CardLayout {
  if (!raw || typeof raw !== "object") return fallback;
  const r = raw as Partial<CardLayout>;

  const minWidth = clamp(Number(r.minWidth ?? fallback.minWidth), CARD_BOUNDS.minWidth.min, CARD_BOUNDS.minWidth.max);
  const maxWidth = clamp(Number(r.maxWidth ?? fallback.maxWidth), CARD_BOUNDS.maxWidth.min, CARD_BOUNDS.maxWidth.max);

  const rows: CardRow[] = [];
  const seen = new Set<CardBlockKind>();
  for (const row of Array.isArray(r.rows) ? r.rows : []) {
    const blocks: CardBlockConfig[] = [];
    for (const b of Array.isArray(row?.blocks) ? row.blocks : []) {
      if (!isBlockKind(b?.kind) || seen.has(b.kind)) continue;
      seen.add(b.kind);
      blocks.push({
        kind: b.kind,
        enabled: b.enabled !== false,
        display: isDisplay(b.display) ? b.display : undefined,
      });
    }
    if (blocks.length > 0) rows.push({ blocks });
  }

  return {
    // A max below the min would make the grid track invalid, so widen rather
    // than reject: the user's intent is still readable from the two numbers.
    minWidth: Math.min(minWidth, maxWidth),
    maxWidth: Math.max(minWidth, maxWidth),
    minHeight: clamp(Number(r.minHeight ?? fallback.minHeight), CARD_BOUNDS.minHeight.min, CARD_BOUNDS.minHeight.max),
    avatarEm: clampEm(Number(r.avatarEm ?? fallback.avatarEm), CARD_BOUNDS.avatarEm.min, CARD_BOUNDS.avatarEm.max),
    rows: rows.length > 0 ? rows : fallback.rows,
    statusBadgeStyle: isBadgeStyle(r.statusBadgeStyle) ? r.statusBadgeStyle : fallback.statusBadgeStyle,
  };
}

/** Reads a stored config, falling back to the default rather than failing. */
export function validateConfig(raw: unknown, fallbackLayout: CardLayout): AccountCardConfig {
  const base: AccountCardConfig = { version: CARD_CONFIG_VERSION, preset: "small" };
  if (!raw || typeof raw !== "object") return base;
  const r = raw as Partial<AccountCardConfig>;

  const blocks: Partial<Record<CardBlockKind, boolean>> = {};
  for (const [k, v] of Object.entries(r.blocks ?? {})) {
    if (isBlockKind(k) && typeof v === "boolean") blocks[k] = v;
  }

  const displays: Partial<Record<CardBlockKind, CardBlockDisplay>> = {};
  for (const [k, v] of Object.entries(r.displays ?? {})) {
    if (isBlockKind(k) && isDisplay(v)) displays[k] = v;
  }

  return {
    version: CARD_CONFIG_VERSION,
    preset: isPreset(r.preset) ? r.preset : "small",
    blocks: Object.keys(blocks).length > 0 ? blocks : undefined,
    displays: Object.keys(displays).length > 0 ? displays : undefined,
    statusBadgeStyle: isBadgeStyle(r.statusBadgeStyle) ? r.statusBadgeStyle : undefined,
    custom: r.custom ? validateLayout(r.custom, fallbackLayout) : undefined,
  };
}
