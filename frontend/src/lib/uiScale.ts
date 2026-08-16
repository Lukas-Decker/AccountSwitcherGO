/**
 * Interface scale.
 *
 * Applied as `zoom` on the document root, which is the only lever that scales
 * every length in the app at once. Scaling the root font size instead would
 * move the text and leave the app's many hardcoded pixel values behind, which
 * reads worse than not scaling at all.
 */

export const UI_SCALE_AUTO = 0;
export const UI_SCALE_MIN = 0.75;
export const UI_SCALE_MAX = 2;

/** The resolution the app's pixel values were chosen against. */
const BASELINE_HEIGHT = 1080;

export interface DisplayInfo {
  /** Logical pixels, as the browser reports them. */
  screenHeight: number;
  devicePixelRatio: number;
}

/**
 * The scale to use when the user has not chosen one.
 *
 * Windows already scales the webview when the display is set above 100%, and
 * that shows up as a device pixel ratio above 1. Scaling again on top of it
 * would double-count, so anything already being scaled is left alone. What is
 * left is the case the OS does not handle: a tall display sitting at 100%.
 */
export function autoUiScale(display: DisplayInfo): number {
  if (display.devicePixelRatio > 1.05) return 1;
  if (!Number.isFinite(display.screenHeight) || display.screenHeight <= 0) return 1;

  const ratio = display.screenHeight / BASELINE_HEIGHT;
  if (ratio <= 1.1) return 1;

  // Quarter steps: enough to matter, few enough that the app does not render
  // at a slightly different size on every machine.
  const stepped = Math.round(ratio * 4) / 4;
  return clampUiScale(stepped);
}

export function clampUiScale(scale: number): number {
  if (!Number.isFinite(scale)) return 1;
  return Math.min(UI_SCALE_MAX, Math.max(UI_SCALE_MIN, Math.round(scale * 100) / 100));
}

/** Resolves a stored setting, where zero or absent means automatic. */
export function effectiveUiScale(stored: number, display: DisplayInfo): number {
  if (!stored || stored <= 0) return autoUiScale(display);
  return clampUiScale(stored);
}

export function currentDisplay(): DisplayInfo {
  if (typeof window === "undefined") {
    return { screenHeight: BASELINE_HEIGHT, devicePixelRatio: 1 };
  }
  return {
    screenHeight: window.screen?.height ?? BASELINE_HEIGHT,
    devicePixelRatio: window.devicePixelRatio || 1,
  };
}

/**
 * The factor between visual coordinates and this document's CSS pixels.
 *
 * `getBoundingClientRect` and pointer events report visual coordinates, which
 * already include the root zoom. Anything that feeds those numbers back into
 * CSS lengths has to divide by this first, or the zoom gets applied twice.
 */
/** The element the zoom is applied to: everything the app draws lives in it. */
function scaleHost(): HTMLElement | null {
  if (typeof document === "undefined") return null;
  return document.getElementById("app") ?? document.documentElement;
}

export function appliedZoom(): number {
  const host = scaleHost();
  if (!host) return 1;
  const parsed = Number(host.style.zoom);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 1;
}

export function applyUiScale(scale: number): void {
  const host = scaleHost();
  if (!host || typeof document === "undefined") return;

  // A scale of exactly 1 is left unset: `zoom` establishes a containing block
  // for fixed-position descendants, and there is no reason to take that on
  // when nothing is being scaled.
  if (scale === 1) {
    host.style.removeProperty("zoom");
    host.style.removeProperty("width");
    host.style.removeProperty("height");
  } else {
    host.style.zoom = String(scale);
    // Sizes resolve before the zoom multiplies the rendered result, so a plain
    // 100% overflows the window by exactly the scale factor. Viewport units
    // rather than percentages, because a percentage height needs a definite
    // parent and would otherwise resolve against content that is itself
    // growing from the overflow this is meant to prevent.
    host.style.width = `calc(100vw / ${scale})`;
    host.style.height = `calc(100vh / ${scale})`;
  }

  document.documentElement.style.setProperty("--ui-scale", String(scale));
}
