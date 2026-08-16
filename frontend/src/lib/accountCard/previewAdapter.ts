import type { PlatformAccountAdapter } from "../../components/PlatformAccountAdapter";
import type { CardBlockKind } from "./types";

/**
 * A stand-in account and adapter, so the card editor and the theme preview can
 * draw a real card instead of a hand-made imitation of one.
 *
 * The card preview drifting from the card is exactly how the theme preview page
 * ended up missing tags, game stats and the live-session marker.
 */
export interface PreviewAccount {
  id: string;
  name: string;
  login: string;
  platformId: string;
  note: string;
  lastUsed: string;
  current: boolean;
  broken: boolean;
  vac: boolean;
  limited: boolean;
  syncError: string;
  tags: { id: string; name: string; colour?: string }[];
}

export const PREVIEW_ACCOUNT: PreviewAccount = {
  id: "preview",
  name: "kestrel_9",
  login: "kestrel_login",
  platformId: "76561198000000001",
  note: "main / ranked",
  // Fixed rather than relative to now, so a preview never renders as "just now"
  // and leaves the reader unsure whether it is live data.
  lastUsed: "2026-01-14T18:20:00Z",
  current: true,
  broken: false,
  vac: false,
  limited: false,
  syncError: "",
  tags: [{ id: "t1", name: "main" }],
};

const NOOP = async () => {};

/**
 * Builds an adapter over the sample account. Only the projection is real: the
 * commands are inert, because nothing in a preview should be able to act on an
 * account.
 */
export function previewAdapter(
  blocks: readonly CardBlockKind[],
  account: PreviewAccount = PREVIEW_ACCOUNT,
): PlatformAccountAdapter<PreviewAccount> {
  const extras = blocks.filter((k) => k === "accountLogin" || k === "platformId" || k === "statusLine" || k === "badges");

  return {
    platformKey: "Preview",
    profileFallback: "/img/BasicDefault.webp",
    cardBlocks: extras as CardBlockKind[],

    id: (a) => a.id,
    name: (a) => a.name,
    imageUrl: () => undefined,
    imagePending: () => false,
    currentSession: (a) => a.current,
    manualProfileImage: () => false,
    savedDataBroken: (a) => a.broken,
    tags: (a) => a.tags as never,
    note: (a) => a.note,
    shouldShowNote: (a) => !!a.note,
    shouldShowLastUsed: (a) => !!a.lastUsed,
    lastUsed: (a) => a.lastUsed,
    accountLogin: (a) => a.login,
    visualKey: (a) => a.id,

    shouldShowAccountLogin: () => true,
    platformId: (a) => a.platformId,
    shouldShowPlatformId: () => true,
    statusLine: (a) =>
      a.syncError ? { kind: "error" as const, text: a.syncError, title: a.syncError } : null,
    badges: (a) => [
      ...(a.vac ? [{ id: "vac", labelKey: "Steam_Badge_Vac", tone: "danger" as const }] : []),
      ...(a.limited ? [{ id: "limited", labelKey: "Steam_Badge_Limited", tone: "warning" as const }] : []),
    ],

    loadAccountsList: async () => [account],
    loadAccountsEnrichment: async () => [account],
    swapTo: NOOP,
    saveOrder: NOOP,
    addNew: NOOP,
    forget: NOOP,
    rename: NOOP,
    changeImage: NOOP,
    clearManualImage: NOOP,
    getNote: async () => account.note,
    setNote: NOOP,
    launch: NOOP,

    updateEventName: "preview-never-fires",
    buildPatch: (raw) => raw,
    applyPatch: (_patch, a) => a,
    patchTargetId: () => "",

    searchHay: (a) => a.name,
    buildMenu: () => [],
  };
}
