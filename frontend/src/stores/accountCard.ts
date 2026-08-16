import { derived, get, writable, type Readable } from "svelte/store";
import * as PlatformService from "../../bindings/account-switcher/internal/platform/platformservice.js";
import { DEFAULT_CARD_CONFIG, presetLayout } from "../lib/accountCard/presets";
import { layoutCssVars } from "../lib/accountCard/resolve";
import { validateConfig, type AccountCardConfig, type CardLayout } from "../lib/accountCard/types";

const STORAGE_KEY = "tcno:account-card";

/** The global card configuration. Platform overrides are layered on at the page. */
export const accountCardConfig = writable<AccountCardConfig>(DEFAULT_CARD_CONFIG);

function readCached(): AccountCardConfig {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return DEFAULT_CARD_CONFIG;
    return validateConfig(JSON.parse(raw), presetLayout("small"));
  } catch {
    return DEFAULT_CARD_CONFIG;
  }
}

function writeCache(config: AccountCardConfig): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(config));
  } catch {
    /* private mode */
  }
}

/**
 * Applies the cached shape immediately, then confirms it against settings.
 * Waiting for the backend first would draw every card at the default size and
 * then visibly resize it once the real config arrived.
 */
export async function initAccountCard(): Promise<void> {
  accountCardConfig.set(readCached());
  try {
    const stored = await PlatformService.GetAccountCardConfig();
    const config = validateConfig(stored, presetLayout("small"));
    accountCardConfig.set(config);
    writeCache(config);
  } catch {
    /* early boot: the cached value stands */
  }
}

export async function setAccountCardConfig(config: AccountCardConfig): Promise<void> {
  const validated = validateConfig(config, presetLayout("small"));
  accountCardConfig.set(validated);
  writeCache(validated);
  await PlatformService.SetAccountCardConfig(validated as never);
}

/** Reads the current global config without subscribing. */
export function currentAccountCardConfig(): AccountCardConfig {
  return get(accountCardConfig);
}

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
