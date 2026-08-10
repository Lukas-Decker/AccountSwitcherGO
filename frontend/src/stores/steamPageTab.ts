import { writable } from "svelte/store";

export type SteamPageTab = "accounts" | "games";

/** Which tab the Steam page shows. Module-level so tabbing away and back keeps
 * the choice, and so the action bar can read it without prop drilling. */
export const steamPageTab = writable<SteamPageTab>("accounts");

export function setSteamPageTab(tab: SteamPageTab): void {
  steamPageTab.set(tab);
}
