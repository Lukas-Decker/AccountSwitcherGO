import { describe, expect, it } from "vitest";
import { resolvePlatformCardConfig } from "./platformOverride";
import type { AccountCardConfig } from "./types";

const GLOBAL: AccountCardConfig = { version: 1, preset: "medium" };
const PLATFORM_CARD = { version: 1, preset: "large" };

describe("resolvePlatformCardConfig", () => {
  it("follows the global shape when a platform has no opinion", () => {
    expect(resolvePlatformCardConfig(GLOBAL, null)).toBe(GLOBAL);
    expect(resolvePlatformCardConfig(GLOBAL, {})).toBe(GLOBAL);
  });

  it("uses the platform's own shape once customisation is on", () => {
    const resolved = resolvePlatformCardConfig(GLOBAL, {
      AccountCardCustomizationEnabled: true,
      AccountCard: PLATFORM_CARD,
    });
    expect(resolved.preset).toBe("large");
  });

  it("ignores a stored layout while the toggle is off, without consuming it", () => {
    // The point of the toggle: a platform layout is kept on disk and simply not
    // read, so turning customisation back on returns the user's work intact.
    const platform = { AccountCardCustomizationEnabled: false, AccountCard: PLATFORM_CARD };
    expect(resolvePlatformCardConfig(GLOBAL, platform).preset).toBe("medium");
    expect(platform.AccountCard).toEqual(PLATFORM_CARD);

    const reEnabled = { ...platform, AccountCardCustomizationEnabled: true };
    expect(resolvePlatformCardConfig(GLOBAL, reEnabled).preset).toBe("large");
  });

  it("falls back to global when customisation is on but nothing was ever built", () => {
    const resolved = resolvePlatformCardConfig(GLOBAL, { AccountCardCustomizationEnabled: true });
    expect(resolved.preset).toBe("medium");
  });

  it("validates a platform layout rather than trusting the file", () => {
    const resolved = resolvePlatformCardConfig(GLOBAL, {
      AccountCardCustomizationEnabled: true,
      AccountCard: { version: 1, preset: "gigantic", blocks: { note: false, bogus: true } },
    });
    expect(resolved.preset).toBe("small");
    expect(resolved.blocks).toEqual({ note: false });
  });
});
