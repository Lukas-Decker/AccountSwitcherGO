import { presetLayout } from "./presets";
import { validateConfig, type AccountCardConfig } from "./types";

/** The shape of the two fields the platform settings file carries. */
export interface PlatformCardSettings {
  AccountCardCustomizationEnabled?: boolean;
  AccountCard?: unknown;
}

/**
 * Chooses which configuration a platform's cards are drawn from.
 *
 * A platform only uses its own shape while its customisation toggle is on.
 * Turning the toggle off falls back to the global shape without touching what
 * was stored, so the layout is still there when it is turned back on.
 */
export function resolvePlatformCardConfig(
  global: AccountCardConfig,
  platform: PlatformCardSettings | null | undefined,
): AccountCardConfig {
  if (!platform?.AccountCardCustomizationEnabled) return global;
  if (!platform.AccountCard) return global;
  return validateConfig(platform.AccountCard, presetLayout("small"));
}
