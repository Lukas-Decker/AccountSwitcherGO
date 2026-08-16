import { derived, writable, type Readable } from "svelte/store";
import { DEFAULT_CARD_CONFIG, presetLayout } from "../lib/accountCard/presets";
import { layoutCssVars } from "../lib/accountCard/resolve";
import type { AccountCardConfig, CardLayout } from "../lib/accountCard/types";

/** The global card configuration. Platform overrides are layered on at the page. */
export const accountCardConfig = writable<AccountCardConfig>(DEFAULT_CARD_CONFIG);

/**
 * Geometry is written to the document root rather than to the list, because
 * the drag ghost is rendered outside the list and would not inherit anything
 * scoped to it.
 */
export function applyCardGeometry(layout: CardLayout): void {
  if (typeof document === "undefined") return;
  const root = document.documentElement;
  for (const [name, value] of Object.entries(layoutCssVars(layout))) {
    root.style.setProperty(name, value);
  }
}

/** Restores the stylesheet's own values, so no page leaks its geometry to the next. */
export function clearCardGeometry(): void {
  if (typeof document === "undefined") return;
  const root = document.documentElement;
  for (const name of Object.keys(layoutCssVars(presetLayout("small")))) {
    root.style.removeProperty(name);
  }
}

/** The layout implied by the global config alone, before any platform override. */
export const globalCardLayout: Readable<CardLayout> = derived(
  accountCardConfig,
  ($config) => ($config.preset === "custom" && $config.custom ? $config.custom : presetLayout($config.preset)),
);
