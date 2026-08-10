/**
 * Hue rotation for the theme palette.
 *
 * The alternative, a CSS `filter: hue-rotate()` on the root element, would also
 * recolour every avatar, game poster and platform logo on the page, and would
 * make the root a containing block for fixed positioning. So the rotation is
 * applied to the theme's own custom properties instead: their colours are read,
 * turned, and re-declared in an overlay stylesheet. Images are left alone.
 */

export const HUE_OVERLAY_STYLE_ID = "theme-hue-overlay";
const HUE_OVERLAY_STYLE_ATTR = "data-theme-hue-overlay";

/** Below this saturation a colour is grey, and turning it only muddies it. */
const NEUTRAL_SATURATION = 0.06;

type Rgba = { r: number; g: number; b: number; a: number };

export function normalizeHueDegrees(value: unknown): number {
  const n = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(n)) {
    return 0;
  }
  return ((Math.round(n) % 360) + 360) % 360;
}

function rgbaToHsl({ r, g, b }: Rgba): { h: number; s: number; l: number } {
  const rn = r / 255;
  const gn = g / 255;
  const bn = b / 255;
  const max = Math.max(rn, gn, bn);
  const min = Math.min(rn, gn, bn);
  const l = (max + min) / 2;
  if (max === min) {
    return { h: 0, s: 0, l };
  }
  const d = max - min;
  const s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
  let h: number;
  switch (max) {
    case rn:
      h = (gn - bn) / d + (gn < bn ? 6 : 0);
      break;
    case gn:
      h = (bn - rn) / d + 2;
      break;
    default:
      h = (rn - gn) / d + 4;
  }
  return { h: h * 60, s, l };
}

function hueToRgbComponent(p: number, q: number, t: number): number {
  let tt = t;
  if (tt < 0) tt += 1;
  if (tt > 1) tt -= 1;
  if (tt < 1 / 6) return p + (q - p) * 6 * tt;
  if (tt < 1 / 2) return q;
  if (tt < 2 / 3) return p + (q - p) * (2 / 3 - tt) * 6;
  return p;
}

function hslToRgb(h: number, s: number, l: number): { r: number; g: number; b: number } {
  if (s === 0) {
    const v = Math.round(l * 255);
    return { r: v, g: v, b: v };
  }
  const q = l < 0.5 ? l * (1 + s) : l + s - l * s;
  const p = 2 * l - q;
  const hn = (((h % 360) + 360) % 360) / 360;
  return {
    r: Math.round(hueToRgbComponent(p, q, hn + 1 / 3) * 255),
    g: Math.round(hueToRgbComponent(p, q, hn) * 255),
    b: Math.round(hueToRgbComponent(p, q, hn - 1 / 3) * 255),
  };
}

function parseHex(token: string): Rgba | null {
  const m = /^#([0-9a-f]{3,8})$/i.exec(token.trim());
  if (!m) return null;
  const hex = m[1];
  const expand = (c: string) => parseInt(c + c, 16);
  if (hex.length === 3 || hex.length === 4) {
    return {
      r: expand(hex[0]),
      g: expand(hex[1]),
      b: expand(hex[2]),
      a: hex.length === 4 ? expand(hex[3]) / 255 : 1,
    };
  }
  if (hex.length === 6 || hex.length === 8) {
    return {
      r: parseInt(hex.slice(0, 2), 16),
      g: parseInt(hex.slice(2, 4), 16),
      b: parseInt(hex.slice(4, 6), 16),
      a: hex.length === 8 ? parseInt(hex.slice(6, 8), 16) / 255 : 1,
    };
  }
  return null;
}

function toHex(rgb: { r: number; g: number; b: number }, a: number): string {
  const pair = (v: number) => Math.max(0, Math.min(255, Math.round(v))).toString(16).padStart(2, "0");
  const base = `#${pair(rgb.r)}${pair(rgb.g)}${pair(rgb.b)}`;
  if (a >= 1) return base;
  return `${base}${pair(a * 255)}`;
}

/** Rotates a single hex colour, leaving greys and transparent values alone. */
function rotateHexToken(token: string, deg: number): string {
  const rgba = parseHex(token);
  if (!rgba) return token;
  const hsl = rgbaToHsl(rgba);
  if (hsl.s < NEUTRAL_SATURATION) return token;
  const rotated = hslToRgb(hsl.h + deg, hsl.s, hsl.l);
  return toHex(rotated, rgba.a);
}

/** Rotates an rgb()/rgba() function, keeping its original notation. */
function rotateRgbFunction(match: string, body: string, deg: number): string {
  const parts = body
    .split(/[,\s/]+/)
    .map((p) => p.trim())
    .filter(Boolean);
  if (parts.length < 3) return match;
  const num = (v: string) => (v.endsWith("%") ? (parseFloat(v) / 100) * 255 : parseFloat(v));
  const rgba: Rgba = { r: num(parts[0]), g: num(parts[1]), b: num(parts[2]), a: 1 };
  if (!Number.isFinite(rgba.r) || !Number.isFinite(rgba.g) || !Number.isFinite(rgba.b)) return match;
  const hsl = rgbaToHsl(rgba);
  if (hsl.s < NEUTRAL_SATURATION) return match;
  const rotated = hslToRgb(hsl.h + deg, hsl.s, hsl.l);
  const alpha = parts.length > 3 ? parts[3] : "";
  return alpha
    ? `rgba(${rotated.r}, ${rotated.g}, ${rotated.b}, ${alpha})`
    : `rgb(${rotated.r}, ${rotated.g}, ${rotated.b})`;
}

/** Rotates an hsl()/hsla() function by shifting its hue term directly. */
function rotateHslFunction(match: string, body: string, deg: number): string {
  const parts = body
    .split(/[,\s/]+/)
    .map((p) => p.trim())
    .filter(Boolean);
  if (parts.length < 3) return match;
  const h = parseFloat(parts[0]);
  const s = parseFloat(parts[1]);
  if (!Number.isFinite(h) || !Number.isFinite(s)) return match;
  if (s / 100 < NEUTRAL_SATURATION) return match;
  const rotated = (((h + deg) % 360) + 360) % 360;
  const rest = parts.slice(1).join(", ");
  const fn = parts.length > 3 ? "hsla" : "hsl";
  return `${fn}(${Math.round(rotated)}, ${rest})`;
}

/**
 * Rotates every colour literal inside a CSS value, so gradients and shadows
 * carry their colours over intact along with plain values.
 */
export function rotateCssValueHue(value: string, deg: number): string {
  const rotation = normalizeHueDegrees(deg);
  if (!value || rotation === 0) return value;

  let out = value.replace(/hsla?\(([^()]*)\)/gi, (m, body: string) => rotateHslFunction(m, body, rotation));
  out = out.replace(/rgba?\(([^()]*)\)/gi, (m, body: string) => rotateRgbFunction(m, body, rotation));
  out = out.replace(/#[0-9a-f]{3,8}\b/gi, (m) => rotateHexToken(m, rotation));
  return out;
}

/** A declaration worth rotating: it names a colour of its own. */
export function shouldRotateDeclaration(name: string, value: string): boolean {
  if (!name.startsWith("--")) return false;
  const v = value.trim();
  if (!v) return false;
  // Values built from other variables inherit the rotation once those are
  // turned. Re-declaring them here would freeze them at a doubly-rotated value.
  if (v.includes("var(")) return false;
  return /#[0-9a-f]{3,8}\b|rgba?\(|hsla?\(/i.test(v);
}

/**
 * Reads the custom properties the loaded stylesheets declare on the root.
 * Later declarations win, which mirrors how the cascade resolved them.
 */
export function collectRootCustomProperties(sheets: Iterable<CSSStyleSheet>): Map<string, string> {
  const out = new Map<string, string>();
  for (const sheet of sheets) {
    let rules: CSSRuleList;
    try {
      rules = sheet.cssRules;
    } catch {
      // A stylesheet from another origin cannot be inspected; nothing of ours.
      continue;
    }
    for (const rule of Array.from(rules)) {
      const styleRule = rule as CSSStyleRule;
      if (!styleRule.selectorText || !styleRule.style) continue;
      if (!/(^|,)\s*(:root|html)\s*(,|$)/i.test(styleRule.selectorText)) continue;
      for (const name of Array.from(styleRule.style)) {
        if (!name.startsWith("--")) continue;
        out.set(name, styleRule.style.getPropertyValue(name).trim());
      }
    }
  }
  return out;
}

export function buildHueOverlayCss(props: Map<string, string>, deg: number): string {
  const rotation = normalizeHueDegrees(deg);
  if (rotation === 0) return "";
  const lines: string[] = [];
  for (const [name, value] of props) {
    if (!shouldRotateDeclaration(name, value)) continue;
    const rotated = rotateCssValueHue(value, rotation);
    if (rotated !== value) {
      lines.push(`  ${name}: ${rotated};`);
    }
  }
  if (lines.length === 0) return "";
  return `:root {\n${lines.join("\n")}\n}\n`;
}

export function removeHueOverlay(): void {
  if (typeof document === "undefined") return;
  document.querySelectorAll(`style[${HUE_OVERLAY_STYLE_ATTR}]`).forEach((node) => node.remove());
}

/**
 * Rebuilds the accent family around a turned accent colour.
 *
 * The accent is not kept as a colour but as pieces of one - "190, 100%" for hue
 * and saturation, "128, 254, 234" for the channels - and most of the interface
 * is painted from those pieces rather than from --accent itself. They carry no
 * colour syntax, so rotating declarations alone leaves every heading, border and
 * highlight on the original hue while the rest of the theme turns.
 */
function buildRotatedAccentCss(deg: number, buildAccentCss: (color: string) => string): string {
  if (typeof document === "undefined") return "";
  const current = getComputedStyle(document.documentElement).getPropertyValue("--accent").trim();
  if (!current) return "";
  const rotated = rotateCssValueHue(current, deg);
  if (!rotated || rotated === current) return "";
  return buildAccentCss(rotated);
}

/**
 * Applies the rotation to the page. Called again whenever the theme changes, so
 * the overlay is rebuilt from whatever palette is now in force.
 */
export function applyHueRotation(deg: number, buildAccentCss?: (color: string) => string): void {
  if (typeof document === "undefined") return;
  removeHueOverlay();
  const rotation = normalizeHueDegrees(deg);
  if (rotation === 0) return;

  const props = collectRootCustomProperties(Array.from(document.styleSheets) as CSSStyleSheet[]);
  let css = buildHueOverlayCss(props, rotation);
  if (buildAccentCss) {
    css += buildRotatedAccentCss(rotation, buildAccentCss);
  }
  if (!css.trim()) return;

  const style = document.createElement("style");
  style.id = HUE_OVERLAY_STYLE_ID;
  style.setAttribute(HUE_OVERLAY_STYLE_ATTR, "");
  style.textContent = css;
  // Last in the head, so it also turns the accent overlay's own declarations.
  // Leaving the accent untouched left every heading, border and highlight
  // painted from it on the theme's original hue.
  document.head.appendChild(style);
}
