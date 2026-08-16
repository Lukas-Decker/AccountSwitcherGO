import { describe, expect, it } from "vitest";
import { flatToRows, layoutToFlat, moveBlock } from "./flatLayout";
import { presetLayout } from "./presets";
import { ALL_BLOCK_KINDS, type CardBlockKind } from "./types";

const EVERY = ALL_BLOCK_KINDS;
const CORE: CardBlockKind[] = ["avatar", "displayName", "tags", "note", "gameStats", "lastUsed"];

describe("rows and the editor's flat list", () => {
  it("round-trips a layout that has a shared line", () => {
    // Medium puts last-used and sync status together; that has to survive being
    // flattened for the editor and rebuilt from it. Blocks the layout never
    // placed are appended after, so only the placed rows are compared.
    const medium = presetLayout("medium");
    const placed = layoutToFlat(medium, EVERY).filter((b) =>
      medium.rows.some((r) => r.blocks.some((x) => x.kind === b.kind)),
    );
    const rebuilt = flatToRows(placed);
    expect(rebuilt.map((r) => r.blocks.map((b) => b.kind))).toEqual(
      medium.rows.map((r) => r.blocks.map((b) => b.kind)),
    );
  });

  it("lists blocks the platform offers but the layout never placed, so they can be switched on", () => {
    const flat = layoutToFlat(presetLayout("medium"), EVERY);
    const badges = flat.find((b) => b.kind === "badges");
    expect(badges).toBeDefined();
    expect(badges?.enabled).toBe(false);
  });

  it("leaves out blocks the platform cannot draw at all", () => {
    const kinds = layoutToFlat(presetLayout("large"), CORE).map((b) => b.kind);
    expect(kinds).not.toContain("platformId");
    expect(kinds).not.toContain("accountLogin");
  });

  it("treats a join with nothing above it as a new line rather than dropping the block", () => {
    const rows = flatToRows([
      { kind: "avatar", enabled: true, joinPrevious: true },
      { kind: "displayName", enabled: true, joinPrevious: true },
    ]);
    expect(rows).toHaveLength(1);
    expect(rows[0].blocks.map((b) => b.kind)).toEqual(["avatar", "displayName"]);
  });

  it("never leaves the first block joined to something that is not there", () => {
    const flat = layoutToFlat(presetLayout("medium"), EVERY);
    const moved = moveBlock(flat, 1, -1);
    expect(moved[0].joinPrevious).toBe(false);
  });

  it("carries a block's settings with it when it moves", () => {
    const flat = layoutToFlat(presetLayout("medium"), EVERY).map((b) =>
      b.kind === "note" ? { ...b, display: "icon" as const, enabled: false } : b,
    );
    const from = flat.findIndex((b) => b.kind === "note");
    const moved = moveBlock(flat, from, -1);
    const note = moved.find((b) => b.kind === "note");
    expect(note?.display).toBe("icon");
    expect(note?.enabled).toBe(false);
    expect(moved.findIndex((b) => b.kind === "note")).toBe(from - 1);
  });

  it("refuses to move a block off either end", () => {
    const flat = layoutToFlat(presetLayout("small"), EVERY);
    expect(moveBlock(flat, 0, -1).map((b) => b.kind)).toEqual(flat.map((b) => b.kind));
    expect(moveBlock(flat, flat.length - 1, 1).map((b) => b.kind)).toEqual(flat.map((b) => b.kind));
  });
});
