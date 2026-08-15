/*
  Masonry packing for a grid, done by hand.

  What the layout wants is `display: grid-lanes`, where a short card lets the
  card below it move up instead of waiting for the tallest card in its row. No
  engine ships that yet, and the app runs on the Chromium webview, so this
  recreates it: the grid gets 1px rows and each item is given a row span equal
  to its own height, which lets items sit at any offset rather than snapping to
  a shared row height. Auto-placement then fills the holes.

  The columns still come from `repeat(auto-fit, minmax(...))` in the
  stylesheet. This only decides how far down each item reaches, so the number
  of columns, the breakpoint-free reflow and the gap all stay in CSS.

  It bows out entirely once the browser can do this itself.
*/

import type { Action } from "svelte/action";

/**
 * Height of one implicit row. Spans are measured in these, so 1px means a card
 * can end anywhere rather than being rounded up to a shared row height.
 */
const ROW_UNIT_PX = 1;

/** True when the engine has native masonry, in which case CSS handles it. */
export function hasNativeMasonry(): boolean {
  if (typeof CSS === "undefined" || typeof CSS.supports !== "function") {
    return false;
  }
  return CSS.supports("display", "grid-lanes") || CSS.supports("grid-template-rows", "masonry");
}

/** Rows a card must span to fit its own height plus the gap under it. */
export function spanForHeight(height: number, gap: number): number {
  return Math.max(1, Math.ceil((height + gap) / ROW_UNIT_PX));
}

function isHidden(el: HTMLElement): boolean {
  // Search hides cards with display:none, and those take no part in the grid.
  return el.getClientRects().length === 0;
}

export const masonryGrid: Action<HTMLElement> = (node) => {
  if (typeof window === "undefined" || typeof ResizeObserver === "undefined" || hasNativeMasonry()) {
    return;
  }

  let frame = 0;
  const measured = new Set<HTMLElement>();

  const itemResize = new ResizeObserver(() => schedule());
  const containerResize = new ResizeObserver(() => schedule());
  const childList = new MutationObserver(() => {
    syncItems();
    schedule();
  });

  function items(): HTMLElement[] {
    return Array.from(node.children).filter(
      (child): child is HTMLElement => child instanceof HTMLElement,
    );
  }

  function syncItems(): void {
    const current = new Set(items());
    for (const el of measured) {
      if (!current.has(el)) {
        itemResize.unobserve(el);
        measured.delete(el);
      }
    }
    for (const el of current) {
      if (!measured.has(el)) {
        itemResize.observe(el);
        measured.add(el);
      }
    }
  }

  function rowGapPx(): number {
    const gap = parseFloat(getComputedStyle(node).columnGap);
    return Number.isFinite(gap) ? gap : 0;
  }

  function measure(): void {
    frame = 0;
    const gap = rowGapPx();
    for (const el of items()) {
      if (isHidden(el)) {
        el.style.removeProperty("grid-row-end");
        continue;
      }
      // The card never stretches, so this is its content height and reading it
      // back after setting the span cannot feed into itself.
      el.style.gridRowEnd = `span ${spanForHeight(el.getBoundingClientRect().height, gap)}`;
    }
  }

  function schedule(): void {
    if (frame) {
      return;
    }
    frame = requestAnimationFrame(measure);
  }

  node.classList.add("settings-grid--masonry");
  syncItems();
  containerResize.observe(node);
  childList.observe(node, { childList: true });
  schedule();

  // Text reflows when the real font arrives, which changes every card's height.
  void document.fonts?.ready.then(schedule).catch(() => {});

  return {
    destroy(): void {
      if (frame) {
        cancelAnimationFrame(frame);
      }
      itemResize.disconnect();
      containerResize.disconnect();
      childList.disconnect();
      node.classList.remove("settings-grid--masonry");
      for (const el of items()) {
        el.style.removeProperty("grid-row-end");
      }
    },
  };
};
