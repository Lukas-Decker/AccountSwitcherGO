import { describe, expect, it } from "vitest";
import {
  buildHueOverlayCss,
  normalizeHueDegrees,
  rotateCssValueHue,
  shouldRotateDeclaration,
} from "./hue";

describe("normalizeHueDegrees", () => {
  it("wraps into 0..359 and tolerates junk", () => {
    expect(normalizeHueDegrees(0)).toBe(0);
    expect(normalizeHueDegrees(360)).toBe(0);
    expect(normalizeHueDegrees(-30)).toBe(330);
    expect(normalizeHueDegrees(400)).toBe(40);
    expect(normalizeHueDegrees("abc")).toBe(0);
    expect(normalizeHueDegrees(undefined)).toBe(0);
  });
});

describe("rotateCssValueHue", () => {
  it("turns a hex colour by the given angle", () => {
    // Dracula cyan #80ffea sits at ~170deg; +60 lands in green.
    const rotated = rotateCssValueHue("#80ffea", 60);
    expect(rotated).not.toBe("#80ffea");
    expect(rotated).toMatch(/^#[0-9a-f]{6}$/i);
  });

  it("is a no-op at zero", () => {
    expect(rotateCssValueHue("#80ffea", 0)).toBe("#80ffea");
  });

  it("comes full circle", () => {
    expect(rotateCssValueHue("#80ffea", 360)).toBe("#80ffea");
  });

  it("leaves greys, black and white alone", () => {
    for (const grey of ["#000000", "#ffffff", "#dddddd", "rgba(0, 0, 0, 0.5)"]) {
      expect(rotateCssValueHue(grey, 120)).toBe(grey);
    }
  });

  it("keeps alpha on rgba and hsla", () => {
    expect(rotateCssValueHue("rgba(128, 255, 234, 0.4)", 90)).toMatch(/rgba\(\d+, \d+, \d+, 0\.4\)/);
    expect(rotateCssValueHue("hsla(170, 100%, 75%, 5%)", 90)).toBe("hsla(260, 100%, 75%, 5%)");
  });

  it("shifts an hsl hue term directly", () => {
    expect(rotateCssValueHue("hsl(208, 25%, 6%)", 40)).toBe("hsl(248, 25%, 6%)");
  });

  it("turns every colour inside a gradient", () => {
    const out = rotateCssValueHue("linear-gradient(90deg, #11181d, #0e1419 100%)", 45);
    expect(out).toContain("linear-gradient(90deg,");
    expect(out).not.toContain("#11181d");
    expect(out).not.toContain("#0e1419");
  });
});

describe("shouldRotateDeclaration", () => {
  it("takes plain colour declarations", () => {
    expect(shouldRotateDeclaration("--cyan", "#80ffea")).toBe(true);
    expect(shouldRotateDeclaration("--x", "rgba(1, 2, 3, 0.5)")).toBe(true);
  });

  it("skips values that point at other variables, which follow on their own", () => {
    expect(shouldRotateDeclaration("--accent", "var(--cyan)")).toBe(false);
    expect(
      shouldRotateDeclaration("--cyan-green", "linear-gradient(135deg, var(--cyan) 0%, var(--green) 100%)"),
    ).toBe(false);
  });

  it("skips non-colours and non-custom properties", () => {
    expect(shouldRotateDeclaration("--border-size", "0.1rem")).toBe(false);
    expect(shouldRotateDeclaration("color", "#80ffea")).toBe(false);
  });
});

describe("buildHueOverlayCss", () => {
  it("emits only the declarations that actually changed", () => {
    const props = new Map<string, string>([
      ["--cyan", "#80ffea"],
      ["--text-muted-gray", "#dddddd"],
      ["--accent", "var(--cyan)"],
      ["--border-size", "0.1rem"],
    ]);

    const css = buildHueOverlayCss(props, 120);

    expect(css).toContain("--cyan:");
    // Grey, variable-backed and non-colour declarations are all left out.
    expect(css).not.toContain("--text-muted-gray");
    expect(css).not.toContain("--accent");
    expect(css).not.toContain("--border-size");
  });

  it("produces nothing at zero degrees", () => {
    expect(buildHueOverlayCss(new Map([["--cyan", "#80ffea"]]), 0)).toBe("");
  });
});
