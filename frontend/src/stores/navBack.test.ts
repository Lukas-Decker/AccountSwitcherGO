import { describe, expect, it, vi, beforeEach } from "vitest";

vi.mock("../../bindings/account-switcher/internal/platform/platformservice.js", () => ({
  GetStartup: vi.fn(),
}));
vi.mock("./platformAccountsCache", () => ({
  setPlatformAccountCounts: vi.fn(),
  setPlatformTagCounts: vi.fn(),
}));
vi.mock("./homeScreenData", () => ({
  homeScreenData: { set: vi.fn(), subscribe: () => () => {} },
}));

/**
 * These tests run without a DOM, so the two globals nav.ts touches are stood up
 * by hand rather than by adding a jsdom dependency for one file.
 *
 * The state reproduced is the one the back button actually died in, read out of
 * a running window: on a platform page, history.state.idx === 0 and
 * history.length === 2. The length is 2 because the stack counts forward entries
 * too, so it says nothing about whether anything sits behind the current entry.
 */
const back = vi.fn();
const replaceState = vi.fn();

beforeEach(() => {
  back.mockClear();
  replaceState.mockClear();
  (globalThis as Record<string, unknown>).window = {
    location: { hash: "#/platform/Riot%20Games" },
    history: { length: 2, state: { idx: 0 }, back, replaceState, pushState: vi.fn() },
  };
  (globalThis as Record<string, unknown>).history =
    (globalThis as { window: { history: unknown } }).window.history;
  (globalThis as Record<string, unknown>).location =
    (globalThis as { window: { location: unknown } }).window.location;
});

describe("navigateBackLikeButton at the oldest history entry", () => {
  it("moves the route rather than calling a back() that cannot fire", async () => {
    const { route, previousPage, navigateBackLikeButton } = await import("./nav");
    const { get } = await import("svelte/store");

    route.set({ page: "platform", platformName: "Riot Games" });
    previousPage.set({ page: "home" });

    navigateBackLikeButton();

    // Reading history.length as "there is somewhere to go back to" meant calling
    // back() at the oldest entry, where the browser correctly does nothing, and
    // returning before this fallback could run. That left the button dead.
    expect(back).not.toHaveBeenCalled();
    expect(get(route)).toEqual({ page: "home" });
  });
});
