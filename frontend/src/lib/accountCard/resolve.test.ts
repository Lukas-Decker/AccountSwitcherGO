import { describe, expect, it } from "vitest";
import { layoutBlockKinds, resolveLayout } from "./resolve";
import { presetLayout } from "./presets";
import { validateConfig, validateLayout, type AccountCardConfig, type CardBlockKind } from "./types";

const EVERY_KIND: CardBlockKind[] = [
  "avatar", "accountLogin", "displayName", "tags", "note",
  "gameStats", "platformId", "lastUsed", "statusLine", "badges",
];

/** What a platform with no extras of its own can draw. */
const CORE_ONLY: CardBlockKind[] = ["avatar", "displayName", "tags", "note", "gameStats", "lastUsed"];

describe("resolveLayout", () => {
  it("drops blocks the platform cannot draw, so a Steam layout leaves no holes elsewhere", () => {
    const config: AccountCardConfig = { version: 1, preset: "large" };
    const kinds = layoutBlockKinds(resolveLayout(config, CORE_ONLY));
    expect(kinds).not.toContain("platformId");
    expect(kinds).not.toContain("accountLogin");
    expect(kinds).toContain("displayName");
  });

  it("lets an explicit block decision override the preset, in either direction", () => {
    const on = resolveLayout({ version: 1, preset: "small", blocks: { note: false } }, EVERY_KIND);
    expect(layoutBlockKinds(on)).not.toContain("note");

    // Large omits nothing, so turning something off is the only visible direction
    // there; small is where turning a block back on has to work.
    const off = resolveLayout({ version: 1, preset: "small", blocks: { note: true } }, EVERY_KIND);
    expect(layoutBlockKinds(off)).toContain("note");
  });

  it("keeps a row only while it still has a block in it", () => {
    const layout = resolveLayout(
      { version: 1, preset: "medium", blocks: { lastUsed: false, statusLine: false } },
      EVERY_KIND,
    );
    // Medium puts those two together on one meta row; with both off the row goes.
    expect(layout.rows.every((r) => r.blocks.length > 0)).toBe(true);
    expect(layoutBlockKinds(layout)).not.toContain("lastUsed");
  });

  it("carries a display override onto the block the preset placed", () => {
    const layout = resolveLayout(
      { version: 1, preset: "medium", displays: { lastUsed: "icon" } },
      EVERY_KIND,
    );
    const lastUsed = layout.rows.flatMap((r) => r.blocks).find((b) => b.kind === "lastUsed");
    expect(lastUsed?.display).toBe("icon");
  });

  it("falls back to small when a custom preset has no stored layout", () => {
    const layout = resolveLayout({ version: 1, preset: "custom" }, EVERY_KIND);
    expect(layout.minWidth).toBe(presetLayout("small").minWidth);
  });
});

describe("validation", () => {
  it("widens rather than rejects a max width below the min, which would break the grid track", () => {
    const layout = validateLayout(
      { ...presetLayout("small"), minWidth: 200, maxWidth: 120 },
      presetLayout("small"),
    );
    expect(layout.minWidth).toBeLessThanOrEqual(layout.maxWidth);
  });

  it("drops unknown block kinds but keeps the rest of the row", () => {
    const layout = validateLayout(
      {
        ...presetLayout("small"),
        rows: [{ blocks: [{ kind: "not-a-block", enabled: true }, { kind: "displayName", enabled: true }] }],
      },
      presetLayout("small"),
    );
    expect(layoutBlockKinds(layout)).toEqual(["displayName"]);
  });

  it("refuses to place the same block twice", () => {
    const layout = validateLayout(
      {
        ...presetLayout("small"),
        rows: [
          { blocks: [{ kind: "displayName", enabled: true }] },
          { blocks: [{ kind: "displayName", enabled: true }] },
        ],
      },
      presetLayout("small"),
    );
    expect(layoutBlockKinds(layout)).toEqual(["displayName"]);
  });

  it("reads a config written by a newer build without discarding what it understands", () => {
    const config = validateConfig(
      { version: 99, preset: "medium", blocks: { note: false, invented: true }, displays: { lastUsed: "hologram" } },
      presetLayout("small"),
    );
    expect(config.preset).toBe("medium");
    expect(config.blocks).toEqual({ note: false });
    expect(config.displays).toBeUndefined();
  });
});
