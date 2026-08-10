import { describe, expect, it } from "vitest";
import { buildHueOverlayCss, rotateCssValueHue } from "./hue";

describe("hue rotation", () => {
  it("turns colours, leaves greys, and keeps alpha", () => {
    expect(rotateCssValueHue("#80ffea", 60)).not.toBe("#80ffea");
    expect(rotateCssValueHue("#80ffea", 360)).toBe("#80ffea");
    expect(rotateCssValueHue("#dddddd", 120)).toBe("#dddddd");
    expect(rotateCssValueHue("hsla(170, 100%, 75%, 5%)", 90)).toBe("hsla(260, 100%, 75%, 5%)");
  });

  it("skips values built from other variables, which would double-rotate", () => {
    const css = buildHueOverlayCss(
      new Map([
        ["--cyan", "#80ffea"],
        ["--accent", "var(--cyan)"],
      ]),
      120,
    );
    expect(css).toContain("--cyan:");
    expect(css).not.toContain("--accent");
  });
});
