import { presetLayout } from "./presets";
import type {
  CardBlockDisplay,
  CardBlockKind,
  CardLayout,
  CardRow,
} from "./types";
import { ALL_BLOCK_KINDS } from "./types";

/**
 * The editor's view of a layout: one flat ordered list.
 *
 * Rows are expressed as a "same line as the one above" flag rather than as a
 * nested structure. Editing a list is a far simpler interaction than editing
 * a tree, and the two forms carry exactly the same information.
 */
export interface FlatBlock {
  kind: CardBlockKind;
  enabled: boolean;
  display?: CardBlockDisplay;
  /** Draws beside the previous block instead of below it. */
  joinPrevious: boolean;
}

export function layoutToFlat(layout: CardLayout, available: readonly CardBlockKind[]): FlatBlock[] {
  const flat: FlatBlock[] = [];
  const seen = new Set<CardBlockKind>();

  for (const row of layout.rows) {
    row.blocks.forEach((block, indexInRow) => {
      if (!available.includes(block.kind)) return;
      seen.add(block.kind);
      flat.push({
        kind: block.kind,
        enabled: block.enabled,
        display: block.display,
        // Only the blocks after the first in a row are joined to it.
        joinPrevious: indexInRow > 0,
      });
    });
  }

  // Blocks the platform offers that this layout never placed, so the editor can
  // still list them as something to switch on.
  for (const kind of ALL_BLOCK_KINDS) {
    if (!available.includes(kind) || seen.has(kind)) continue;
    flat.push({ kind, enabled: false, joinPrevious: false });
  }

  return flat;
}

export function flatToRows(flat: readonly FlatBlock[]): CardRow[] {
  const rows: CardRow[] = [];
  for (const block of flat) {
    const entry = { kind: block.kind, enabled: block.enabled, display: block.display };
    // A join with nothing above it is just a new row; the flag cannot make the
    // first block of the card sit beside something that is not there.
    if (block.joinPrevious && rows.length > 0) {
      rows[rows.length - 1].blocks.push(entry);
    } else {
      rows.push({ blocks: [entry] });
    }
  }
  return rows;
}

export function flatToLayout(flat: readonly FlatBlock[], base: CardLayout): CardLayout {
  return { ...base, rows: flatToRows(flat) };
}

/** Moves a block one place, carrying its settings with it. */
export function moveBlock(flat: readonly FlatBlock[], index: number, delta: -1 | 1): FlatBlock[] {
  const target = index + delta;
  if (index < 0 || index >= flat.length || target < 0 || target >= flat.length) return [...flat];
  const next = flat.map((b) => ({ ...b }));
  [next[index], next[target]] = [next[target], next[index]];
  // A block that has been moved to the top cannot be joined to the one above.
  if (next[0]) next[0].joinPrevious = false;
  return next;
}

/** A starting point for Custom: whatever the user is currently looking at. */
export function flatFromPreset(preset: Parameters<typeof presetLayout>[0], available: readonly CardBlockKind[]): FlatBlock[] {
  return layoutToFlat(presetLayout(preset), available);
}
