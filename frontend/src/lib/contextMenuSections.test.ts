import { describe, expect, it } from "vitest";
import { menuSectionOwners, sectionHasRows } from "./contextMenuSections";

describe("context menu sections", () => {
  it("gives the rows after a header to that header", () => {
    expect(
      menuSectionOwners([
        { type: "header" },
        {},
        {},
      ]),
    ).toEqual([[], [0], [0]]);
  });

  it("ends a section at the separator, so a title at the top cannot fold the whole menu", () => {
    const owners = menuSectionOwners([
      { type: "header" },
      {},
      {},
      { type: "separator" },
      {},
      {},
    ]);
    expect(owners).toEqual([[], [0], [0], [], [], []]);
  });

  it("folds a subheader and its rows away with the header above it", () => {
    expect(
      menuSectionOwners([
        { type: "header" },
        {},
        { type: "subheader" },
        {},
      ]),
    ).toEqual([[], [0], [0], [0, 2]]);
  });

  it("starts a fresh subsection at the next subheader", () => {
    expect(
      menuSectionOwners([
        { type: "header" },
        { type: "subheader" },
        {},
        { type: "subheader" },
        {},
      ]),
    ).toEqual([[], [0], [0, 1], [0], [0, 3]]);
  });

  it("leaves a subheader with no header above it standing on its own", () => {
    expect(
      menuSectionOwners([{ type: "subheader" }, {}]),
    ).toEqual([[], [0]]);
  });

  it("leaves rows before any title unowned", () => {
    expect(
      menuSectionOwners([{}, { type: "header" }, {}]),
    ).toEqual([[], [], [1]]);
  });

  it("knows when a title has nothing under it to fold", () => {
    const owners = menuSectionOwners([
      { type: "header" },
      { type: "separator" },
      { type: "header" },
      {},
    ]);
    expect(sectionHasRows(owners, 0)).toBe(false);
    expect(sectionHasRows(owners, 2)).toBe(true);
  });
});
