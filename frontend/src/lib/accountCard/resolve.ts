import { presetLayout, seedCustomLayout } from "./presets";
import type {
  AccountCardConfig,
  CardBlockKind,
  CardLayout,
  CardRow,
} from "./types";

/**
 * Turns a stored config into the layout the renderer draws.
 *
 * Three things are folded in, in this order: the preset supplies the shape,
 * the user's explicit per-block decisions override it, and finally anything
 * the platform cannot draw is dropped. The last step is why a layout built on
 * Steam does not leave holes on a platform with no SteamID to show.
 */
export function resolveLayout(
  config: AccountCardConfig,
  availableKinds: readonly CardBlockKind[],
): CardLayout {
  const base =
    config.preset === "custom"
      ? (config.custom ?? seedCustomLayout("small"))
      : presetLayout(config.preset);

  const available = new Set(availableKinds);
  const enabledOverride = config.blocks ?? {};
  const displayOverride = config.displays ?? {};

  const rows: CardRow[] = [];
  for (const row of base.rows) {
    const blocks = row.blocks
      .filter((b) => available.has(b.kind))
      .map((b) => ({
        kind: b.kind,
        enabled: enabledOverride[b.kind] ?? b.enabled,
        display: displayOverride[b.kind] ?? b.display,
      }))
      .filter((b) => b.enabled);
    if (blocks.length > 0) rows.push({ blocks });
  }

  return {
    ...base,
    rows,
    statusBadgeStyle: config.statusBadgeStyle ?? base.statusBadgeStyle,
  };
}

/**
 * The geometry tokens for a layout, written to the document root so the drag
 * ghost picks them up too: it renders outside the list and would not inherit
 * anything scoped to it.
 */
export function layoutCssVars(layout: CardLayout): Record<string, string> {
  return {
    "--acc-card-min-w": `${layout.minWidth}px`,
    "--acc-card-max-w": `${layout.maxWidth}px`,
    "--acc-card-min-h": `${layout.minHeight}px`,
    "--acc-avatar-size": `${layout.avatarEm}em`,
    "--acc-card-font-scale": String(layout.fontScale),
  };
}

/** Every block the layout actually draws, in order. */
export function layoutBlockKinds(layout: CardLayout): CardBlockKind[] {
  return layout.rows.flatMap((row) => row.blocks.map((b) => b.kind));
}

/**
 * The colour overrides for a config, as CSS variables.
 *
 * Only states the user actually set are emitted, so anything untouched keeps
 * falling through to the theme. The signed-in colour drives the ring rather
 * than a fill, because that is how that state has always been shown.
 */
export function colorCssVars(config: AccountCardConfig): Record<string, string> {
  const colors = config.colors ?? {};
  const vars: Record<string, string> = {};
  if (colors.rest) vars["--acc-card-bg"] = colors.rest;
  if (colors.hover) vars["--acc-card-bg-hover"] = colors.hover;
  if (colors.selected) {
    vars["--acc-card-bg-selected"] = colors.selected;
    vars["--acc-card-selected-edge"] = colors.selected;
  }
  if (colors.current) vars["--acc-ring-color"] = colors.current;
  return vars;
}

/** Colour variables the card sets, so a page can clear the ones it did not set. */
export const CARD_COLOR_VAR_NAMES = [
  "--acc-card-bg",
  "--acc-card-bg-hover",
  "--acc-card-bg-selected",
  "--acc-card-selected-edge",
  "--acc-ring-color",
] as const;
