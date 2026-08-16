import { describe, expect, it } from "vitest";
import { CARD_COLOR_VAR_NAMES, colorCssVars } from "./resolve";
import { presetLayout } from "./presets";
import { validateConfig } from "./types";

describe("card state colours", () => {
  it("emits nothing for a config that sets no colours, so the theme stays in charge", () => {
    expect(colorCssVars({ version: 1, preset: "small" })).toEqual({});
  });

  it("maps each state onto the variable that actually drives it", () => {
    const vars = colorCssVars({
      version: 1,
      preset: "small",
      colors: { rest: "#101010", hover: "#202020", selected: "#303030", current: "#40ff40" },
    });
    expect(vars["--acc-card-bg"]).toBe("#101010");
    expect(vars["--acc-card-bg-hover"]).toBe("#202020");
    expect(vars["--acc-card-bg-selected"]).toBe("#303030");
    // The signed-in state has always been shown as a ring, not a fill.
    expect(vars["--acc-ring-color"]).toBe("#40ff40");
  });

  it("carries both of the signed-in ring's colours, because it is a gradient", () => {
    const vars = colorCssVars({
      version: 1,
      preset: "small",
      colors: { current: "#40ff40", currentGlint: "#ffee88" },
    });
    expect(vars["--acc-ring-color"]).toBe("#40ff40");
    expect(vars["--acc-ring-highlight"]).toBe("#ffee88");
  });

  it("only emits the states that were set", () => {
    const vars = colorCssVars({ version: 1, preset: "small", colors: { hover: "#123456" } });
    expect(Object.keys(vars)).toEqual(["--acc-card-bg-hover"]);
  });

  it("lists every variable it can emit, so a page can clear the ones it did not set", () => {
    const emitted = Object.keys(
      colorCssVars({
        version: 1,
        preset: "small",
        colors: { rest: "#111", hover: "#222", selected: "#333", current: "#444" },
      }),
    );
    for (const name of emitted) {
      expect(CARD_COLOR_VAR_NAMES).toContain(name as (typeof CARD_COLOR_VAR_NAMES)[number]);
    }
  });

  it("refuses anything that is not a hex colour", () => {
    // These values are written straight into a stylesheet, so a stored config
    // must not be able to smuggle arbitrary CSS through this field.
    const config = validateConfig(
      {
        version: 1,
        preset: "small",
        colors: {
          rest: "red; background: url(http://example.com)",
          hover: "javascript:alert(1)",
          selected: "var(--something)",
          current: "#ABCDEF",
        },
      },
      presetLayout("small"),
    );
    expect(config.colors).toEqual({ current: "#abcdef" });
  });

  it("accepts the shorthand and alpha hex forms", () => {
    const config = validateConfig(
      { version: 1, preset: "small", colors: { rest: "#FFF", hover: "#11223344" } },
      presetLayout("small"),
    );
    expect(config.colors).toEqual({ rest: "#fff", hover: "#11223344" });
  });
});
