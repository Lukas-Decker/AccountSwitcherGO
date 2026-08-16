import type { ComponentType } from "svelte";
import type { MenuItemDef } from "../stores/contextMenu";
import type { PlatformSortKind } from "../stores/platformListSort";
import type { SearchResultRow } from "./SearchOverlay.svelte";
import type { TagDefRow } from "../lib/accountTagsContext";
import type { CardBlockKind } from "../lib/accountCard/types";

export type GameStatMetricDTO = { statValue: string; indicatorMarkup: string };

export interface SharedMenuItems {
  swapTo: MenuItemDef;
  changeName: MenuItemDef;
  createShortcut: MenuItemDef;
  changeImage: MenuItemDef;
  forget: MenuItemDef;
  notes: MenuItemDef;
  tags: MenuItemDef;
  gameStats: MenuItemDef | null;
}

export interface AccountRowProjection<TAccount> {
  id(a: TAccount): string;
  name(a: TAccount): string;
  imageUrl(a: TAccount): string | undefined;
  imagePending(a: TAccount): boolean;
  currentSession(a: TAccount): boolean;
  manualProfileImage(a: TAccount): boolean;
  savedDataBroken?(a: TAccount): boolean;
  tags(a: TAccount): TagDefRow[] | undefined;
  note(a: TAccount): string;
  shouldShowNote(a: TAccount): boolean;
  shouldShowLastUsed(a: TAccount): boolean;
  lastUsed(a: TAccount): string;
  accountLogin(a: TAccount): string;
  visualKey(a: TAccount): string;

  /** Whether the login name is meant to be drawn on the card. */
  shouldShowAccountLogin?(a: TAccount): boolean;

  /**
   * The platform's own public identifier for the account (Steam's SteamID64,
   * for example). Anything returned here is treated as identifying and is
   * hidden by streamer mode.
   */
  platformId?(a: TAccount): string;
  shouldShowPlatformId?(a: TAccount): boolean;

  /** A short sync state for the card's footer. */
  statusLine?(a: TAccount): AccountStatusLine | null;

  /**
   * Standing warnings about the account, such as Steam's VAC and limited
   * flags. Where these are drawn is the user's choice, so the adapter reports
   * them rather than deciding how they look.
   */
  badges?(a: TAccount): AccountBadge[];
}

export type AccountBadge = {
  id: string;
  /** i18n key; also the accessible name when the badge draws as a mark only. */
  labelKey: string;
  tone: "danger" | "warning";
};

export type AccountStatusLine = {
  kind: "error" | "pending";
  text: string;
  /** Long errors are truncated on the card and shown in full on hover. */
  title?: string;
};

export interface AccountDataSource<TAccount> {
  loadAccountsList(): Promise<TAccount[]>;
  loadAccountsEnrichment(): Promise<TAccount[]>;
}

export interface AccountCommands {
  swapTo(id: string): Promise<void>;
  saveOrder(ids: string[]): Promise<void>;
  addNew(): Promise<void>;
  forget(id: string): Promise<void>;
  rename(id: string, name: string): Promise<void>;
  changeImage(id: string, path: string): Promise<void>;
  clearManualImage(id: string): Promise<void>;
  getNote(id: string): Promise<string>;
  setNote(id: string, note: string): Promise<void>;
  launch(): Promise<void>;
}

export interface AccountPatchStream<TAccount> {
  updateEventName: string;
  buildPatch(raw: unknown): unknown;
  applyPatch(patch: unknown, account: TAccount): TAccount;
  patchTargetId(patch: unknown): string;
}

export interface AccountSearchProjection<TAccount> {
  searchHay(account: TAccount, query: string): string;
}

export interface AccountLifecycle<TAccount> {
  onAfterLoad?(
    accounts: TAccount[],
    ctx?: { hadCachedAccounts: boolean; enrichChanged: boolean },
  ): void | Promise<void>;
  onCleanup?(): void;
}

export interface AccountPageExtensions {
  saveCurrent?(): Promise<boolean>;
  suggestedSaveName?(): Promise<string>;
  refreshOnWindowFocus?: boolean;
  refreshAccounts?(): Promise<void>;
  refreshAllProfileImages?(): Promise<void>;
  gameSearchRows?(query: string): SearchResultRow[];
  gameSearchHint?: string;
  loginAndLaunchGame?(accountId: string, appId: string): Promise<void>;
  extraSortKinds?: PlatformSortKind[];
}

export interface AccountMenuBuilder<TAccount> {
  buildMenu(account: TAccount, shared: SharedMenuItems): MenuItemDef[];
}

export interface PlatformAccountAdapter<TAccount>
  extends AccountRowProjection<TAccount>,
    AccountDataSource<TAccount>,
    AccountCommands,
    AccountPatchStream<TAccount>,
    AccountSearchProjection<TAccount>,
    AccountLifecycle<TAccount>,
    AccountPageExtensions,
    AccountMenuBuilder<TAccount> {
  /** Stable key identifying the platform (e.g. "Steam", "Battle.net"). */
  platformKey: string;

  /** Default placeholder image when no profile image is available. */
  profileFallback: string;

  /**
   * Card blocks this platform offers on top of the ones every platform has.
   * Declared rather than inferred from the platform's name, so the card never
   * needs to know which platform it is drawing.
   */
  cardBlocks?: CardBlockKind[];

  /**
   * Replaces the default avatar for platforms whose avatars carry more than an
   * image, such as Steam's animated avatars and frames.
   */
  avatarComponent?: ComponentType;
}
