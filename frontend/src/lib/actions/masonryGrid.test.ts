import { describe, expect, it } from "vitest";

import { spanForHeight } from "./masonryGrid";

describe("spanForHeight", () => {
  it("covers the card and the gap that follows it", () => {
    expect(spanForHeight(200, 10)).toBe(210);
  });

  it("rounds a fractional height up, so a card is never clipped", () => {
    expect(spanForHeight(200.2, 10)).toBe(211);
  });

  it("works without a gap", () => {
    expect(spanForHeight(64, 0)).toBe(64);
  });

  it("never spans less than one row", () => {
    expect(spanForHeight(0, 0)).toBe(1);
  });
});
