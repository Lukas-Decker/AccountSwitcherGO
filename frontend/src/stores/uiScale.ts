import { writable } from "svelte/store";
import * as PlatformService from "../../bindings/account-switcher/internal/platform/platformservice.js";
import {
  applyUiScale,
  currentDisplay,
  effectiveUiScale,
  UI_SCALE_AUTO,
} from "../lib/uiScale";
import { UI_SCALE_STORAGE_KEY } from "../lib/storageKeys";

const STORAGE_KEY = UI_SCALE_STORAGE_KEY;

/** The stored preference: 0 means automatic. */
export const uiScaleSetting = writable<number>(UI_SCALE_AUTO);
/** What is actually applied, after resolving automatic against this display. */
export const uiScaleEffective = writable<number>(1);

function apply(stored: number): number {
  const scale = effectiveUiScale(stored, currentDisplay());
  applyUiScale(scale);
  uiScaleEffective.set(scale);
  return scale;
}

/**
 * Applies the cached value immediately, then confirms it against settings.
 * Reading from the backend first would leave the window drawing at the wrong
 * size for as long as that call takes, which is very visible.
 */
export async function initUiScale(): Promise<void> {
  let stored = UI_SCALE_AUTO;
  try {
    const cached = Number(localStorage.getItem(STORAGE_KEY));
    if (Number.isFinite(cached) && cached > 0) stored = cached;
  } catch {
    /* private mode */
  }
  uiScaleSetting.set(stored);
  apply(stored);

  try {
    const fromSettings = await PlatformService.GetUIScale();
    uiScaleSetting.set(fromSettings);
    apply(fromSettings);
    try {
      localStorage.setItem(STORAGE_KEY, String(fromSettings));
    } catch {
      /* private mode */
    }
  } catch {
    /* early boot: the cached value stands */
  }
}

export async function setUiScale(scale: number): Promise<void> {
  uiScaleSetting.set(scale);
  apply(scale);
  try {
    localStorage.setItem(STORAGE_KEY, String(scale));
  } catch {
    /* private mode */
  }
  await PlatformService.SetUIScale(scale);
}
