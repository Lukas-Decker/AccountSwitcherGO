import { writable } from "svelte/store";

export type PlatformPageTab = "accounts" | "games";

/**
 * Which tab each non-Steam platform page shows, keyed by platform.
 *
 * Kept per platform rather than as one shared value: someone who browses games
 * on Epic and accounts on Ubisoft should find each page as they left it, and a
 * single store would make every page follow the last one visited.
 */
const tabs = writable<Record<string, PlatformPageTab>>({});

export const platformPageTabs = tabs;

export function setPlatformPageTab(platformKey: string, tab: PlatformPageTab): void {
  tabs.update((current) => ({ ...current, [platformKey]: tab }));
}
