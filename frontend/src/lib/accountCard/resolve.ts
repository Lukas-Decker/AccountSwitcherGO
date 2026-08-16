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
