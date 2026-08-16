/**
 * The pieces an account card can be built from.
 *
 * Kept apart from the registry itself so the adapter interface can name them
 * without pulling Svelte components into every module that imports it.
 */
export type CardBlockKind =
  | "avatar"
  | "accountLogin"
  | "displayName"
  | "tags"
  | "note"
  | "gameStats"
  | "platformId"
  | "lastUsed"
  | "statusLine";

/**
 * Blocks every platform can draw, in the order the card has always drawn them.
 * Anything outside this set has to be declared by the platform's adapter.
 */
export const CORE_BLOCK_KINDS: readonly CardBlockKind[] = [
  "avatar",
  "displayName",
  "tags",
  "note",
  "gameStats",
  "lastUsed",
];

/** The stack order of a card, whichever blocks are turned on. */
export const BLOCK_ORDER: readonly CardBlockKind[] = [
  "avatar",
  "accountLogin",
  "displayName",
  "tags",
  "note",
  "gameStats",
  "platformId",
  "lastUsed",
  "statusLine",
];
