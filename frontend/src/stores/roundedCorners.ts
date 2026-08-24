import { writable } from "svelte/store";
import * as PlatformService from "../../bindings/account-switcher/internal/platform/platformservice.js";
import { ROUNDED_CORNERS_STORAGE_KEY } from "../lib/storageKeys";

const STORAGE_KEY = ROUNDED_CORNERS_STORAGE_KEY;
const ROOT_CLASS = "rounded-corners";

export const roundedCorners = writable<boolean>(false);

function applyClass(enabled: boolean): void {
  if (typeof document === "undefined") return;
  document.documentElement.classList.toggle(ROOT_CLASS, enabled);
}

/** Applies the cached value immediately, then confirms it against settings. */
export async function initRoundedCorners(): Promise<void> {
  let enabled = false;
  try {
    enabled = localStorage.getItem(STORAGE_KEY) === "1";
  } catch {
    /* private mode */
  }
  roundedCorners.set(enabled);
  applyClass(enabled);

  try {
    const stored = await PlatformService.GetRoundedCorners();
    roundedCorners.set(stored);
    applyClass(stored);
    try {
      localStorage.setItem(STORAGE_KEY, stored ? "1" : "0");
    } catch {
      /* private mode */
    }
  } catch {
    /* early boot: the cached value stands */
  }
}

export async function setRoundedCorners(enabled: boolean): Promise<void> {
  roundedCorners.set(enabled);
  applyClass(enabled);
  try {
    localStorage.setItem(STORAGE_KEY, enabled ? "1" : "0");
  } catch {
    /* private mode */
  }
  await PlatformService.SetRoundedCorners(enabled);
}
