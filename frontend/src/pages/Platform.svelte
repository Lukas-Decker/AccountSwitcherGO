<script lang="ts">
  import { get } from "svelte/store";
  import PlatformAccountsBase from "../components/PlatformAccountsBase.svelte";
  import GamesView from "../components/GamesView.svelte";
  import { platformPageTabs, setPlatformPageTab } from "../stores/platformPageTab";
  import type { PlatformAccountAdapter } from "../components/PlatformAccountAdapter";
  import type { TagDefRow } from "../lib/accountTagsContext";
  import type { MenuItemDef } from "../stores/contextMenu";
  import { pushToast } from "../stores/toast";
  import { locale, t } from "../stores/i18n";
  import * as BasicService from "../../bindings/account-switcher/internal/basic/basicservice.js";
  import {
    AccountDTO,
    AccountEnrichmentDTO,
    AccountImagePatch,
    AccountListItemDTO,
  } from "../../bindings/account-switcher/internal/basic/models.js";
  import { LaunchPlatform } from "../../bindings/account-switcher/internal/platform/platformservice.js";
  import { formatToastWithError } from "../lib/formatWailsError";
  import { offerRestartIfNeedsAdmin, isNeedsAdminError } from "../lib/adminFlow";
  import { openPrompt, openRiotLinkModal } from "../stores/modal";
  import { openExternalUrl } from "../lib/openExternalUrl";
  import { buildRiotMenu, RIOT_PLATFORM_KEY } from "../lib/riot/riotMenu";
  import * as RiotService from "../../bindings/account-switcher/internal/riotservice/service.js";
  import "../styles/platformAccountsShared.scss";

  const PROFILE_FALLBACK = "/img/BasicDefault.webp";

  type BasicRow = InstanceType<typeof AccountDTO> & {
    currentSession?: boolean;
    avatarPending?: boolean;
    tags?: TagDefRow[];
    manualProfileImage?: boolean;
    showLastUsed?: boolean;
    savedDataBroken?: boolean;
    note?: string;
    lastUsed?: string;
  };

  export let name: string;

  type RiotCard = Awaited<ReturnType<typeof RiotService.GetCard>>;
  let riotCards: Record<string, RiotCard | undefined> = {};

  /** Riot submenu from the cached card, refreshed in the background.
   *  Never awaits: a menu must not hang on a game client or a spent API quota. */
  function riotMenuFor(uniqueId: string): MenuItemDef {
    return buildRiotMenu(riotCards[uniqueId] ?? null, {
      tr: (k, v) => get(t)(k, v),
      locale: get(locale),
      openUrl: (url) => void openExternalUrl(url, { allowAnyHttps: true }),
      editLink: () => void editRiotLink(uniqueId),
      refresh: () => void refreshRiotCard(uniqueId),
    });
  }

  /**
   * Loads what is already known about every linked account, in one local read.
   *
   * The linked state and the last captured figures live in the same file the
   * account list comes from, so this needs no network and cannot be late in a way
   * that matters: it completes with the list, before any menu can be opened.
   *
   * This exists because a context menu is built from plain values at the instant
   * it opens and never updates afterwards. Anything the menu needs has to be in
   * hand by then, so the menu depends on stored data only; going to the client or
   * the API is what Refresh is for.
   */
  async function loadRiotCards(): Promise<void> {
    try {
      riotCards = await RiotService.AccountCards();
    } catch {
      // The rest of the page is unaffected; the submenu simply has nothing to add.
    }
  }

  /** The deliberate, on-demand live lookup behind the menu's Refresh. */
  async function refreshRiotCard(uniqueId: string): Promise<void> {
    if (!uniqueId) return;
    try {
      riotCards = { ...riotCards, [uniqueId]: await RiotService.RefreshCard(uniqueId) };
    } catch {
      // A card is an extra, never a reason to break the menu it hangs off.
    }
  }

  /** Region list, fetched once: it is a fixed table in the backend. */
  let riotRegions: { platform: string; display: string }[] = [];

  async function loadRiotRegions(): Promise<void> {
    if (riotRegions.length > 0) return;
    try {
      riotRegions = await RiotService.Regions();
    } catch {
      // An empty list still lets the Riot ID be edited; the select is simply
      // empty, which is no worse than the free-text field it replaced.
      riotRegions = [];
    }
  }

  async function editRiotLink(uniqueId: string): Promise<void> {
    const current = riotCards[uniqueId];
    await loadRiotRegions();
    const result = await openRiotLinkModal({
      title: get(t)("Riot_CardTitle"),
      riotId: current?.riotId ?? "",
      region: current?.region ?? "",
      regions: riotRegions,
    });
    if (result === null) return;
    try {
      await RiotService.SetAccountLink(uniqueId, result.riotId, result.region);
      await loadRiotCards();
    } catch (e) {
      pushToast({ type: "error", message: formatToastWithError(get(t)("Riot_SaveFailed"), e), duration: 8000 });
    }
  }

  $: adapter = {
    platformKey: name,
    profileFallback: PROFILE_FALLBACK,

    id: (a: BasicRow) => a.uniqueId,
    name: (a: BasicRow) => a.displayName ?? "",
    imageUrl: (a: BasicRow) => a.imageUrl,
    imagePending: (a: BasicRow) => a.avatarPending ?? false,
    currentSession: (a: BasicRow) => a.currentSession ?? false,
    manualProfileImage: (a: BasicRow) => a.manualProfileImage ?? false,
    savedDataBroken: (a: BasicRow) => a.savedDataBroken ?? false,
    tags: (a: BasicRow) => a.tags,
    note: (a: BasicRow) => a.note ?? "",
    shouldShowNote: (a: BasicRow) => !!(a.note ?? "").trim(),
    shouldShowLastUsed: (a: BasicRow) => !!(a.lastUsed ?? "").trim(),
    lastUsed: (a: BasicRow) => a.lastUsed ?? "",
    accountLogin: () => "",

    visualKey: (a: BasicRow) => [
      a.uniqueId,
      a.displayName ?? "",
      a.imageUrl ?? "",
      a.avatarPending ?? false,
      a.manualProfileImage ?? false,
      a.currentSession ?? false,
      a.savedDataBroken ?? false,
      a.note ?? "",
      a.lastUsed ?? "",
      (a.tags ?? []).map((t) => t.id).join(","),
    ].join("|"),

    loadAccountsList: async () => {
      const rows = await BasicService.GetAccountsList(name);
      const list = rows.map((r: AccountListItemDTO) => ({
        platformKey: r.platformKey,
        uniqueId: r.uniqueId,
        displayName: r.displayName,
        currentSession: r.currentSession ?? false,
        savedDataBroken: r.savedDataBroken ?? false,
      })) as BasicRow[];
      if (name === RIOT_PLATFORM_KEY) {
        await loadRiotCards();
      }
      return list;
    },
    loadAccountsEnrichment: async () => {
      const rows = await BasicService.GetAccountsEnrichment(name);
      return rows.map((r: AccountEnrichmentDTO) => ({
        uniqueId: r.uniqueId,
        imageUrl: r.imageUrl,
        avatarPending: r.avatarPending ?? false,
        manualProfileImage: r.manualProfileImage ?? false,
        note: r.note ?? "",
        lastUsed: r.lastUsed ?? "",
        showLastUsed: r.showLastUsed ?? false,
        savedDataBroken: r.savedDataBroken ?? false,
        tags: r.tags,
      })) as BasicRow[];
    },
    swapTo: (id: string) => BasicService.SwapToAccount(name, id, []),
    saveOrder: (ids: string[]) => BasicService.SaveAccountOrder(name, ids),
    addNew: () => BasicService.AddNew(name),
    forget: (id: string) => BasicService.ForgetAccount(name, id),
    rename: (id: string, newName: string) => BasicService.RenameAccount(name, id, newName),
    changeImage: (id: string, path: string) => BasicService.ChangeAccountImage(name, id, path),
    clearManualImage: (id: string) => BasicService.ClearManualAccountProfileImage(name, id),
    getNote: (id: string) => BasicService.GetAccountNote(name, id),
    setNote: (id: string, note: string) => BasicService.SetAccountNote(name, id, note),
    launch: () => LaunchPlatform(name),

    buildMenu: (acc, shared) => [
      ...(name === RIOT_PLATFORM_KEY ? [riotMenuFor(acc.uniqueId)] : []),
      shared.swapTo,
      shared.changeName,
      shared.createShortcut,
      shared.changeImage,
      shared.forget,
      shared.notes,
      shared.tags,
      shared.gameStats,
    ].filter((x): x is MenuItemDef => x != null),

    updateEventName: "basic-account-image-updated",
    buildPatch: (raw: unknown) =>
      raw instanceof AccountImagePatch
        ? raw
        : AccountImagePatch.createFrom(raw as Record<string, unknown>),
    patchTargetId: (patch: unknown) => {
      const p = patch as { platformKey?: string; uniqueId?: string };
      return (p.uniqueId ?? "").trim();
    },
    applyPatch: (patch: unknown, account: BasicRow) => {
      const p = patch as {
        platformKey?: string; uniqueId?: string;
        imageUrl?: string | null; avatarPending?: boolean;
        manualProfileImage?: boolean;
      };
      const prevUrl = (account.imageUrl ?? "").trim();
      const nextUrl = p.imageUrl != null ? String(p.imageUrl).trim() : prevUrl;
      const nextPending = typeof p.avatarPending === "boolean" ? p.avatarPending : (account.avatarPending ?? false);
      const nextManual = typeof p.manualProfileImage === "boolean" ? p.manualProfileImage : (account.manualProfileImage ?? false);
      return { ...account, imageUrl: nextUrl, avatarPending: nextPending, manualProfileImage: nextManual } as BasicRow;
    },

    searchHay: (a: BasicRow, _q: string) => [a.displayName, a.uniqueId, a.note ?? ""].join("\n"),

    saveCurrent: async () => {
      let suggestedName = "";
      try {
        suggestedName = await BasicService.SuggestedSaveAccountName(name);
      } catch (e) {
        await offerRestartIfNeedsAdmin(e, name);
        if (isNeedsAdminError(e)) return false;
      }
      const displayName = await openPrompt({
        title: get(t)("Modal_SaveCurrent_Title"),
        body: get(t)("Modal_SaveCurrent_Body"),
        positiveLabel: get(t)("Button_SaveCurrent"),
        negativeLabel: get(t)("Button_Cancel"),
        initialValue: suggestedName || "",
      });
      if (displayName === null || !String(displayName).trim()) return false;
      try {
        await BasicService.SaveCurrent(name, String(displayName).trim());
        pushToast({ type: "success", message: get(t)("Toast_AccountSaved"), duration: 4000 });
        return true;
      } catch (e) {
        await offerRestartIfNeedsAdmin(e, name);
        if (isNeedsAdminError(e)) return false;
        pushToast({ type: "error", message: formatToastWithError(get(t)("Toast_SaveFailed"), e), duration: 8000 });
        return false;
      }
    },

    suggestedSaveName: async () => {
      try { return await BasicService.SuggestedSaveAccountName(name); }
      catch { return ""; }
    },
  } satisfies PlatformAccountAdapter<BasicRow>;

  $: activeTab = $platformPageTabs[name] ?? "accounts";

  /**
   * Switches to the account and starts the platform.
   *
   * Unlike Steam, these launchers take no app id on the command line, so the
   * best that can be done is to switch and open the launcher on the game's
   * account. The games view knows this and hides its launch toggle.
   */
  async function switchToGameAccount(_game: unknown, uniqueId: string): Promise<void> {
    await BasicService.SwapToAccount(name, uniqueId, []);
  }
</script>

<div class="main-content platform-accounts-root">
  <div class="platform-tabs" role="tablist">
    <button
      type="button"
      role="tab"
      class="platform-tab"
      class:active={activeTab === "accounts"}
      aria-selected={activeTab === "accounts"}
      on:click={() => setPlatformPageTab(name, "accounts")}
    >
      {$t("Steam_Tab_Accounts")}
    </button>
    <button
      type="button"
      role="tab"
      class="platform-tab"
      class:active={activeTab === "games"}
      aria-selected={activeTab === "games"}
      on:click={() => setPlatformPageTab(name, "games")}
    >
      {$t("Steam_Tab_Games")}
    </button>
  </div>

  <!-- Both stay mounted so switching tabs does not throw away the account
       list's avatars and enrichment, or the games grid's artwork. -->
  <div class="platform-tab-panel" class:hidden={activeTab !== "games"}>
    <GamesView platformKey={name} canLaunchGame={false} onSwitch={switchToGameAccount} />
  </div>

  <div class="platform-tab-panel" class:hidden={activeTab !== "accounts"}>
    <PlatformAccountsBase {name} {adapter} />
  </div>
</div>

<style lang="scss">
  /* The page is its own scroll container, as on the Steam page: the tab strip
     has to stay put while a long games grid scrolls under it, and the gutter is
     reserved on the scrollbar's side only so the grid keeps equal margins. */
  .platform-accounts-root {
    scrollbar-gutter: stable;
  }

  /* Matches the Steam page's tabs: the two pages sit side by side in the nav
     and a different tab strip on each would read as a different app. */
  .platform-tabs {
    display: flex;
    gap: 0.1875rem;
    margin: 0 0 0.1875rem;
    border-bottom: 1px solid var(--border-bar-bg, #444);
  }

  .platform-tab {
    padding: 0.3375rem 0.825rem;
    border: none;
    border-bottom: 2px solid transparent;
    background: none;
    color: inherit;
    font: inherit;
    line-height: 1;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    opacity: 0.65;
    cursor: pointer;

    &:hover,
    &:focus-visible {
      opacity: 0.9;
    }

    &.active {
      opacity: 1;
      border-bottom-color: var(--accent, #f90);
    }
  }

  .platform-tab-panel.hidden {
    display: none;
  }
</style>
