/**
 * localStorage keys, and the one-time rename of the keys used before the app
 * was rebranded.
 *
 * The old keys were prefixed `tcno:`. Renaming them without moving the values
 * would silently reset every user's theme, accent, UI scale, card layout and
 * offline toggle back to defaults on the update that shipped the rename, which
 * reads as the update having lost their settings.
 *
 * The migration runs as a side effect of importing this module. Every consumer
 * takes its key from here, so the copy is guaranteed to have happened before
 * anything reads: a store that reads at module scope still imports its key
 * first.
 */

const LEGACY_PREFIX = "tcno:";
const PREFIX = "accsw:";

/** Every key the app owns, without a prefix. */
const NAMES = [
  "theme",
  "theme-accent",
  "theme-accent-custom",
  "theme-hue",
  "account-card",
  "offlineMode",
  "rounded-corners",
  "ui-scale",
] as const;

type KeyName = (typeof NAMES)[number];

function key(name: KeyName): string {
  return PREFIX + name;
}

/**
 * Copies any value still stored under the old prefix across, then drops it.
 *
 * Never overwrites a value already present under the new key: if the app has
 * run since the rename, the new key is the truth and the leftover is stale.
 */
function migrateLegacyKeys(): void {
  try {
    if (typeof localStorage === "undefined") return;
  } catch {
    // Reading localStorage throws outright when storage is blocked, which is
    // not a reason to fail to boot.
    return;
  }
  for (const name of NAMES) {
    try {
      const legacy = localStorage.getItem(LEGACY_PREFIX + name);
      if (legacy === null) continue;
      if (localStorage.getItem(PREFIX + name) === null) {
        localStorage.setItem(PREFIX + name, legacy);
      }
      localStorage.removeItem(LEGACY_PREFIX + name);
    } catch {
      // A single key failing to move is not worth aborting the rest over.
    }
  }
}

migrateLegacyKeys();

export const THEME_STORAGE_KEY = key("theme");
export const THEME_ACCENT_STORAGE_KEY = key("theme-accent");
export const THEME_ACCENT_CUSTOM_STORAGE_KEY = key("theme-accent-custom");
export const THEME_HUE_STORAGE_KEY = key("theme-hue");
export const ACCOUNT_CARD_STORAGE_KEY = key("account-card");
export const OFFLINE_MODE_STORAGE_KEY = key("offlineMode");
export const ROUNDED_CORNERS_STORAGE_KEY = key("rounded-corners");
export const UI_SCALE_STORAGE_KEY = key("ui-scale");
