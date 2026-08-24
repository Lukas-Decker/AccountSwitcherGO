import { get } from "svelte/store";
import * as PlatformService from "../../../bindings/account-switcher/internal/platform/platformservice.js";
import { offlineMode } from "../../stores/offlineMode";
import { setUserOverride } from "../../stores/backgroundImage";
import { scheduleUpdaterThemeSync } from "../updaterTheme";
import {
  DEFAULT_THEME_ID,
  DEFAULT_THEME_OPTION,
  CUSTOM_THEME_ACCENT_KEY,
  WINDOWS_THEME_ACCENT_KEY,
} from "./types";
import type { ResolvedThemeAccent } from "./types";
import { getThemeOptionById, isKnownThemeId, styleLoaderPathForId, themeStyles } from "./catalog";
import { normalizeHexColor } from "./color";
import {
  syncThemeGoogleFonts,
  removeThemeOverlay,
  removeAccentOverlay,
  removeThemeGoogleFontLinks,
  applyAccentOverlay,
  applyResolvedAccent,
  validateAccentKey,
  clearThemeAccentState,
  ensureWindowsAccentSubscription,
} from "./dom";
import { supportsWindowsThemeAccent } from "./dom";
import { applyHueRotation, normalizeHueDegrees } from "./hue";
import { buildAccentOverlayCss } from "./color";
import { currentThemeId, currentThemeBgUrl, currentThemeAccentKey, currentThemeCustomAccentColor, currentWindowsThemeAccentColor, currentThemeHueRotate } from "./stores";

import {
  THEME_STORAGE_KEY,
  THEME_ACCENT_STORAGE_KEY,
  THEME_ACCENT_CUSTOM_STORAGE_KEY,
  THEME_HUE_STORAGE_KEY,
} from "../storageKeys";

/**
 * Reads the stored rotation. Kept in localStorage as well as in settings so the
 * palette is already turned on the first paint, rather than snapping across
 * once the backend answers.
 */
export function loadCachedHueRotate(): number {
  try {
    return normalizeHueDegrees(localStorage.getItem(THEME_HUE_STORAGE_KEY));
  } catch {
    return 0;
  }
}

async function loadStoredHueRotate(): Promise<number> {
  try {
    return normalizeHueDegrees(await PlatformService.GetThemeHueRotate());
  } catch {
    return loadCachedHueRotate();
  }
}

/** Turns the palette now and remembers the angle. */
export async function setThemeHueRotate(deg: number): Promise<void> {
  const rotation = normalizeHueDegrees(deg);
  currentThemeHueRotate.set(rotation);
  applyHueRotation(rotation, buildAccentOverlayCss);
  try {
    localStorage.setItem(THEME_HUE_STORAGE_KEY, String(rotation));
  } catch {
    /* private mode */
  }
  try {
    await PlatformService.SetThemeHueRotate(rotation);
  } catch {
    /* offline / early boot */
  }
  scheduleUpdaterThemeSync();
}

/** Re-applies the current rotation, for after a theme swap replaces the palette. */
export function reapplyThemeHueRotation(): void {
  applyHueRotation(get(currentThemeHueRotate), buildAccentOverlayCss);
}

let activeThemeRequestId = 0;

export function resolveThemeAccent(
  themeId: string,
  accentKey = get(currentThemeAccentKey),
  customColor = get(currentThemeCustomAccentColor),
): ResolvedThemeAccent {
  const theme = getThemeOptionById(themeId);
  if (accentKey === CUSTOM_THEME_ACCENT_KEY) {
    return {
      id: CUSTOM_THEME_ACCENT_KEY,
      label: "Custom",
      color: normalizeHexColor(customColor) ?? theme.defaultAccentColor,
      isCustom: true,
    };
  }
  if (accentKey === WINDOWS_THEME_ACCENT_KEY && supportsWindowsThemeAccent()) {
    return {
      id: WINDOWS_THEME_ACCENT_KEY,
      label: "Windows Accent",
      color: get(currentWindowsThemeAccentColor) || theme.defaultAccentColor,
      isCustom: false,
    };
  }
  const preset =
    theme.accents.find((option) => option.id === accentKey) ??
    theme.accents.find((option) => option.id === theme.defaultAccentKey) ?? {
      id: theme.defaultAccentKey,
      label: "Accent",
      color: theme.defaultAccentColor,
    };
  return { ...preset, isCustom: false };
}

async function refreshWindowsThemeAccentColor(): Promise<string> {
  if (!supportsWindowsThemeAccent()) {
    currentWindowsThemeAccentColor.set("");
    return "";
  }
  return get(currentWindowsThemeAccentColor);
}

async function loadStoredThemeId(): Promise<string> {
  try {
    const persisted = String((await PlatformService.GetTheme()) ?? "").trim();
    return persisted || DEFAULT_THEME_ID;
  } catch {
    return localStorage.getItem(THEME_STORAGE_KEY)?.trim() || DEFAULT_THEME_ID;
  }
}

async function loadStoredAccentState(): Promise<{ accentKey: string; customColor: string }> {
  try {
    const [accentKey, customColor] = await Promise.all([
      PlatformService.GetThemeAccentPreset(),
      PlatformService.GetThemeAccentCustom(),
    ]);
    return {
      accentKey: String(accentKey ?? "").trim(),
      customColor: String(customColor ?? "").trim(),
    };
  } catch {
    return {
      accentKey: localStorage.getItem(THEME_ACCENT_STORAGE_KEY)?.trim() || "",
      customColor: localStorage.getItem(THEME_ACCENT_CUSTOM_STORAGE_KEY)?.trim() || "",
    };
  }
}

async function persistAccentState(accentKey: string, customColor: string): Promise<void> {
  try {
    await Promise.all([
      PlatformService.SetThemeAccentPreset(accentKey),
      PlatformService.SetThemeAccentCustom(customColor),
    ]);
  } catch {
    /* offline / early boot */
  }

  if (accentKey) {
    localStorage.setItem(THEME_ACCENT_STORAGE_KEY, accentKey);
  } else {
    localStorage.removeItem(THEME_ACCENT_STORAGE_KEY);
  }

  if (customColor) {
    localStorage.setItem(THEME_ACCENT_CUSTOM_STORAGE_KEY, customColor);
  } else {
    localStorage.removeItem(THEME_ACCENT_CUSTOM_STORAGE_KEY);
  }
}

async function applyTheme(id: string): Promise<void> {
  const requestId = ++activeThemeRequestId;
  removeThemeOverlay();
  removeAccentOverlay();
  removeThemeGoogleFontLinks();

  if (id === DEFAULT_THEME_ID) {
    currentThemeId.set(DEFAULT_THEME_ID);
    currentThemeBgUrl.set("");
    syncThemeGoogleFonts(DEFAULT_THEME_ID);
    scheduleUpdaterThemeSync();
    return;
  }

  const key = styleLoaderPathForId(id);
  if (!key) {
    console.warn("[themes] Unknown or missing style for theme:", id);
    currentThemeId.set(DEFAULT_THEME_ID);
    currentThemeBgUrl.set("");
    syncThemeGoogleFonts(DEFAULT_THEME_ID);
    scheduleUpdaterThemeSync();
    return;
  }

  const load = themeStyles[key];
  const css = await load();
  if (requestId !== activeThemeRequestId) {
    return;
  }

  removeThemeOverlay();
  const style = document.createElement("style");
  style.id = "accsw-theme-overlay";
  style.setAttribute("data-accsw-theme-overlay", "");
  style.textContent = css;
  document.head.appendChild(style);
  currentThemeId.set(id);
  currentThemeBgUrl.set(getThemeOptionById(id).backgroundUrl ?? "");
  syncThemeGoogleFonts(id);
  // The new theme brought its own palette, so the rotation has to be rebuilt
  // from it rather than left pointing at the old colours.
  reapplyThemeHueRotation();
  scheduleUpdaterThemeSync();
}

export async function initTheme(): Promise<void> {
  // Seed from the cached angle first so the very first paint is already turned.
  currentThemeHueRotate.set(loadCachedHueRotate());

  let [id, storedAccent, hue] = await Promise.all([
    loadStoredThemeId(),
    loadStoredAccentState(),
    loadStoredHueRotate(),
  ]);
  if (!isKnownThemeId(id)) {
    id = DEFAULT_THEME_ID;
  }
  currentThemeHueRotate.set(hue);
  ensureWindowsAccentSubscription();
  await refreshWindowsThemeAccentColor();
  await applyTheme(id);
  applyResolvedAccent(id, storedAccent.accentKey, storedAccent.customColor);
  reapplyThemeHueRotation();
}

export async function setUserTheme(id: string): Promise<void> {
  const next = isKnownThemeId(id) ? id : DEFAULT_THEME_ID;
  const previous = get(currentThemeId);
  const persist = next === DEFAULT_THEME_ID ? "" : next;

  try {
    await PlatformService.SetTheme(persist);
  } catch {
    /* offline / early boot */
  }

  localStorage.setItem(THEME_STORAGE_KEY, next);

  if (next === previous) {
    return;
  }

  await persistAccentState("", "");
  clearThemeAccentState();
  await applyTheme(next);
  await setUserOverride(false);
}

export async function setUserThemeAccentPreset(accentKey: string): Promise<void> {
  const theme = getThemeOptionById(get(currentThemeId));
  const validAccentKey = validateAccentKey(theme, accentKey);
  const customColor = normalizeHexColor(get(currentThemeCustomAccentColor)) ?? "";

  if (validAccentKey === WINDOWS_THEME_ACCENT_KEY) {
    await refreshWindowsThemeAccentColor();
  }

  if (!validAccentKey || validAccentKey === theme.defaultAccentKey) {
    await persistAccentState("", customColor);
    applyResolvedAccent(theme.id, "", customColor);
    return;
  }

  await persistAccentState(validAccentKey, customColor);
  applyResolvedAccent(theme.id, validAccentKey, customColor);
}

export async function setUserThemeAccentCustom(color: string): Promise<void> {
  const theme = getThemeOptionById(get(currentThemeId));
  const normalized =
    normalizeHexColor(color) ?? resolveThemeAccent(theme.id).color ?? theme.defaultAccentColor;
  await persistAccentState(CUSTOM_THEME_ACCENT_KEY, normalized);
  applyResolvedAccent(theme.id, CUSTOM_THEME_ACCENT_KEY, normalized);
}

offlineMode.subscribe(() => {
  syncThemeGoogleFonts(get(currentThemeId));
});
