import { describe, expect, it } from "vitest";
import { autoUiScale, effectiveUiScale, UI_SCALE_MAX, UI_SCALE_MIN } from "./uiScale";

describe("autoUiScale", () => {
  it("leaves a display the OS is already scaling alone, rather than scaling it twice", () => {
    // Windows at 150% reports a device pixel ratio of 1.5 and has already made
    // everything physically bigger; scaling again would double-count it.
    expect(autoUiScale({ screenHeight: 2160, devicePixelRatio: 1.5 })).toBe(1);
    expect(autoUiScale({ screenHeight: 1440, devicePixelRatio: 2 })).toBe(1);
  });

  it("changes nothing at the resolution the app was built against", () => {
    expect(autoUiScale({ screenHeight: 1080, devicePixelRatio: 1 })).toBe(1);
  });

  it("scales up a tall display left at 100%, which is the case the OS does not cover", () => {
    expect(autoUiScale({ screenHeight: 1440, devicePixelRatio: 1 })).toBe(1.25);
    expect(autoUiScale({ screenHeight: 2160, devicePixelRatio: 1 })).toBe(UI_SCALE_MAX);
  });

  it("does not scale for a display only slightly taller than the baseline", () => {
    expect(autoUiScale({ screenHeight: 1152, devicePixelRatio: 1 })).toBe(1);
  });

  it("survives a display it cannot read", () => {
    expect(autoUiScale({ screenHeight: 0, devicePixelRatio: 1 })).toBe(1);
    expect(autoUiScale({ screenHeight: Number.NaN, devicePixelRatio: 1 })).toBe(1);
  });
});

describe("effectiveUiScale", () => {
  const display = { screenHeight: 1440, devicePixelRatio: 1 };

  it("treats zero as automatic, so the setting can mean 'decide for me'", () => {
    expect(effectiveUiScale(0, display)).toBe(autoUiScale(display));
  });

  it("lets an explicit choice overrule what the display suggests", () => {
    expect(effectiveUiScale(1, display)).toBe(1);
    expect(effectiveUiScale(1.5, display)).toBe(1.5);
  });

  it("clamps a stored value that is out of range instead of applying it", () => {
    expect(effectiveUiScale(9, display)).toBe(UI_SCALE_MAX);
    expect(effectiveUiScale(0.1, display)).toBe(UI_SCALE_MIN);
  });
});
