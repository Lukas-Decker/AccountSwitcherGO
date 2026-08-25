/**
 * How the context menu looks, as one shared setting rather than per call site.
 *
 * The menu itself was already a module: openContextMenu takes items from
 * anywhere and one component draws them. What it had no notion of was how the
 * user wants it to look, so every menu in the app was stuck with one density,
 * one weight and no icons. These preferences are published as CSS custom
 * properties on the menu root, which is why nothing here needs to know about
 * the markup and the markup needs to know nothing about settings.
 */

import { get, writable, derived } from "svelte/store";

/** Row height and padding. Slim is the reason this exists. */
export type ContextMenuDensity = "slim" | "normal" | "roomy";

/** Weight applied to a row's label. */
export type ContextMenuWeight = "normal" | "medium" | "bold";

/** Which font a row's label uses. */
export type ContextMenuFont = "app" | "system" | "mono";

/** How a section title is drawn. All three centre and embolden it; they differ in colour. */
export type ContextMenuHeaderStyle = "accent" | "plain" | "band";

export type ContextMenuStyle = {
  density: ContextMenuDensity;
  /** Icons are optional per item; this hides them all when false. */
  showIcons: boolean;
  fontSize: number;
  weight: ContextMenuWeight;
  font: ContextMenuFont;
  /** Italic labels. Off by default; a whole menu in italic is hard to scan. */
  italic: boolean;
  /** Section titles are optional. Off hides every one, and with them the folding they offer. */
  showHeaders: boolean;
  headerStyle: ContextMenuHeaderStyle;
};

export const CONTEXT_MENU_STYLE_DEFAULT: ContextMenuStyle = {
  density: "normal",
  showIcons: true,
  fontSize: 13,
  weight: "normal",
  font: "app",
  italic: false,
  showHeaders: true,
  headerStyle: "accent",
};

const STORAGE_KEY = "accsw:contextMenuStyle";

function readStored(): ContextMenuStyle {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return { ...CONTEXT_MENU_STYLE_DEFAULT };
    const parsed = JSON.parse(raw) as Partial<ContextMenuStyle>;
    return sanitize({ ...CONTEXT_MENU_STYLE_DEFAULT, ...parsed });
  } catch {
    // A hand-edited or half-written value must not stop menus from drawing.
    return { ...CONTEXT_MENU_STYLE_DEFAULT };
  }
}

/** Keeps a stored value inside the range the CSS was written for. */
function sanitize(s: ContextMenuStyle): ContextMenuStyle {
  const densities: ContextMenuDensity[] = ["slim", "normal", "roomy"];
  const weights: ContextMenuWeight[] = ["normal", "medium", "bold"];
  const fonts: ContextMenuFont[] = ["app", "system", "mono"];
  const headerStyles: ContextMenuHeaderStyle[] = ["accent", "plain", "band"];
  return {
    density: densities.includes(s.density) ? s.density : "normal",
    showIcons: Boolean(s.showIcons),
    fontSize: Math.min(20, Math.max(10, Number(s.fontSize) || 13)),
    weight: weights.includes(s.weight) ? s.weight : "normal",
    font: fonts.includes(s.font) ? s.font : "app",
    italic: Boolean(s.italic),
    showHeaders: Boolean(s.showHeaders),
    headerStyle: headerStyles.includes(s.headerStyle) ? s.headerStyle : "accent",
  };
}

export const contextMenuStyle = writable<ContextMenuStyle>(readStored());

contextMenuStyle.subscribe((s) => {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(s));
  } catch {
    // Private mode, or a full quota. The menu still works this session.
  }
});

export function setContextMenuStyle(patch: Partial<ContextMenuStyle>): void {
  contextMenuStyle.set(sanitize({ ...get(contextMenuStyle), ...patch }));
}

export function resetContextMenuStyle(): void {
  contextMenuStyle.set({ ...CONTEXT_MENU_STYLE_DEFAULT });
}

const DENSITY_PADDING: Record<ContextMenuDensity, string> = {
  slim: "0.15rem 0.5rem",
  normal: "0.3rem 0.6375rem",
  roomy: "0.5rem 0.85rem",
};

const WEIGHT_VALUE: Record<ContextMenuWeight, string> = {
  normal: "400",
  medium: "600",
  bold: "700",
};

const FONT_STACK: Record<ContextMenuFont, string> = {
  app: "inherit",
  system:
    'system-ui, -apple-system, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
  mono: 'ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace',
};

/**
 * A section title is centred and bold whichever of these is chosen, since that is what stops it
 * reading as a row the user cannot click. What varies is how loudly it announces itself.
 */
const HEADER_COLOR: Record<ContextMenuHeaderStyle, string> = {
  accent: "var(--accent)",
  plain: "var(--white)",
  band: "var(--white)",
};

const HEADER_BACKGROUND: Record<ContextMenuHeaderStyle, string> = {
  accent: "transparent",
  plain: "transparent",
  band: "var(--surface-context-row)",
};

/**
 * The style as an inline `style` string for the menu root.
 *
 * Custom properties rather than classes so a submenu nested any number of
 * levels deep inherits the same values without every level having to be told.
 */
export const contextMenuCssVars = derived(contextMenuStyle, (s) =>
  [
    `--ctx-row-padding:${DENSITY_PADDING[s.density]}`,
    `--ctx-font-size:${s.fontSize}px`,
    `--ctx-font-weight:${WEIGHT_VALUE[s.weight]}`,
    `--ctx-font-family:${FONT_STACK[s.font]}`,
    `--ctx-font-style:${s.italic ? "italic" : "normal"}`,
    `--ctx-icon-display:${s.showIcons ? "inline-flex" : "none"}`,
    `--ctx-header-color:${HEADER_COLOR[s.headerStyle]}`,
    `--ctx-header-bg:${HEADER_BACKGROUND[s.headerStyle]}`,
  ].join(";"),
);
