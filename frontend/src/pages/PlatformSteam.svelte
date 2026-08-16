<script lang="ts">
  import { onDestroy, onMount, type ComponentType } from "svelte";
  import { get } from "svelte/store";
  import { Events } from "@wailsio/runtime";
  import SteamAccountAvatar from "../components/SteamAccountAvatar.svelte";
  import PlatformAccountsBase from "../components/PlatformAccountsBase.svelte";
  import SteamGamesView from "../components/SteamGamesView.svelte";
  import { steamPageTab } from "../stores/steamPageTab";
  import type { AccountBadge, PlatformAccountAdapter, SharedMenuItems } from "../components/PlatformAccountAdapter";
  import type { TagDefRow } from "../lib/accountTagsContext";
  import type { MenuItemDef } from "../stores/contextMenu";
  import type { PlatformSortKind } from "../stores/platformListSort";
  import { pushToast } from "../stores/toast";
  import { t } from "../stores/i18n";
  import { openConfirm, openPrompt } from "../stores/modal";
  import * as SteamService from "../../bindings/account-switcher/internal/steam/steamservice.js";
  import {
    AccountDTO,
    AccountPatch,
    SteamAccountEnrichmentDTO,
    SteamAccountListItemDTO,
  } from "../../bindings/account-switcher/internal/steam/models.js";
  import * as Shortcuts from "wails-shortcuts-service";
  import { ListPayload } from "../../bindings/account-switcher/internal/shortcuts/models.js";
  import { offlineMode } from "../stores/offlineMode";
  import { formatToastWithError } from "../lib/formatWailsError";
  import * as BasicService from "../../bindings/account-switcher/internal/basic/basicservice.js";
  import { buildSteamExtraMenu } from "../lib/steam/contextMenuBuilder";
  import type { SteamMenuDeps } from "../lib/steam/menuCommands";
  import type { SteamAccountRow } from "../lib/steam/types";
  import { steamAccountVisualKey } from "../lib/steam/accountVisualKey";
  import { reportLaunchFailure } from "../lib/adminFlow";
  import { fuzzyWordsMatch } from "../lib/searchFuzzy";
  import { formatLastLoginForLocale } from "../lib/formatLastLogin";
  import { shortcutIconIndexes, steamGameIconUrl } from "../lib/shortcutAssets";
  import "../styles/miniprofile.scss";
  import "../styles/platformAccountsShared.scss";

  const PROFILE_PLACEHOLDER = "/img/BasicDefault.webp";
  const SHORTCUT_ICON_FALLBACK = "/img/icons/file.svg";


  const STEAM_CONTEXT_MENU_HIDDEN_APP_IDS = new Set(["228980"]);




  type SteamAccountPatch = AccountPatch & {
    avatarFrameUrl?: string; miniProfileHtml?: string;
    showMiniProfile?: boolean; showAvatarFrame?: boolean;
  };

  export let name: string;

  let installedGames: { appId: string; name: string }[] = [];
  let steamShortcutIconByAppId: Record<string, string> = {};
  let steamShortcutIconByStemKey: Record<string, string> = {};
  let gameDataBySteamId: Record<string, { userdata: Set<string>; backup: Set<string> }> = {};
  let offShortcutsUpdated: (() => void) | undefined;

  function applyShortcutIconsFromShortcutList(list: unknown[]): void {
    const indexes = shortcutIconIndexes(list, name, SHORTCUT_ICON_FALLBACK, get(offlineMode));
    steamShortcutIconByAppId = indexes.byAppId;
    steamShortcutIconByStemKey = indexes.byStemKey;
  }

  function resolveSteamGameSearchIcon(g: { appId: string; name: string }): string {
    return steamGameIconUrl(
      g,
      name,
      { byAppId: steamShortcutIconByAppId, byStemKey: steamShortcutIconByStemKey },
      SHORTCUT_ICON_FALLBACK,
      get(offlineMode),
    );
  }

  async function refreshGameDataAppSets(steamIds: string[]): Promise<void> {
    if (steamIds.length === 0) { gameDataBySteamId = {}; return; }
    try {
      const parts = await Promise.all(steamIds.map(async (id) => {
        try {
          const s = await SteamService.GetSteamGameDataAppIDSets(id);
          return [id, { userdata: new Set(s.userdataAppIds.map((x: string) => String(x).trim())), backup: new Set(s.backupAppIds.map((x: string) => String(x).trim())) }] as const;
        } catch { return [id, { userdata: new Set<string>(), backup: new Set<string>() }] as const; }
      }));
      gameDataBySteamId = Object.fromEntries(parts);
    } catch { gameDataBySteamId = {}; }
  }


  function buildSteamExtraMenuAdapter(acc: SteamAccountRow, shared: SharedMenuItems): MenuItemDef[] {
    return buildSteamExtraMenu(acc, shared, getSteamMenuDeps());
  }

  function getSteamMenuDeps(): SteamMenuDeps {
    return {
      name,
      installedGames,
      gameDataBySteamId,
      steamIds,
      refreshGameDataAppSets,
    };
  }

  let steamIds: string[] = [];

  $: adapter = {
    platformKey: "Steam",
    profileFallback: PROFILE_PLACEHOLDER,

    // What Steam adds to the card beyond the blocks every platform has.
    cardBlocks: ["accountLogin", "platformId", "statusLine", "badges"],
    avatarComponent: SteamAccountAvatar as unknown as ComponentType,

    id: (a: SteamAccountRow) => a.steamId64,
    name: (a: SteamAccountRow) => a.displayName?.trim() || a.personaName?.trim() || a.steamId64,
    imageUrl: (a: SteamAccountRow) => a.imageUrl,
    imagePending: (a: SteamAccountRow) => a.avatarPending ?? false,
    currentSession: (a: SteamAccountRow) => a.currentSession ?? false,
    manualProfileImage: (a: SteamAccountRow) => a.manualProfileImage ?? false,
    tags: (a: SteamAccountRow) => a.tags,
    note: (a: SteamAccountRow) => a.note ?? "",
    shouldShowNote: (a: SteamAccountRow) => !!(a.note ?? "").trim(),
    shouldShowLastUsed: (a: SteamAccountRow) => !!(a.lastLogin ?? "").trim(),
    lastUsed: (a: SteamAccountRow) => a.lastLogin ?? "",
    accountLogin: (a: SteamAccountRow) => (a.accountName ?? "").trim(),
    shouldShowAccountLogin: (a: SteamAccountRow) => !!(a.accountName ?? "").trim(),

    platformId: (a: SteamAccountRow) => a.steamId64 ?? "",
    shouldShowPlatformId: (a: SteamAccountRow) => !!(a.steamId64 ?? "").trim(),

    badges: (a: SteamAccountRow) => {
      const out: AccountBadge[] = [];
      if (a.vac) out.push({ id: "vac", labelKey: "Steam_Badge_Vac", tone: "danger" });
      if (a.ltd) out.push({ id: "limited", labelKey: "Steam_Badge_Limited", tone: "warning" });
      return out;
    },

    statusLine: (a: SteamAccountRow) => {
      if (a.syncError) return { kind: "error" as const, text: a.syncError, title: a.syncError };
      if (a.metaPending || a.avatarPending) {
        return { kind: "pending" as const, text: get(t)("Status_Updating") };
      }
      return null;
    },

    visualKey: steamAccountVisualKey,

    loadAccountsList: async () => {
      const rows = await SteamService.GetSteamAccountsList();
      return rows.map((r: SteamAccountListItemDTO) => ({
        steamId64: r.steamId64,
        personaName: r.personaName,
        displayName: r.displayName,
        accountName: r.accountName,
        currentSession: r.currentSession ?? false,
      })) as SteamAccountRow[];
    },
    loadAccountsEnrichment: async () => {
      const rows = await SteamService.GetSteamAccountsEnrichment();
      return rows.map((r: SteamAccountEnrichmentDTO) => ({
        steamId64: r.steamId64,
        displayName: r.displayName,
        lastLogin: r.lastLogin,
        offline: r.offline ?? false,
        imageUrl: r.imageUrl,
        staticImageUrl: r.staticImageUrl,
        avatarPending: r.avatarPending ?? false,
        metaPending: r.metaPending ?? false,
        vac: r.vac ?? false,
        ltd: r.ltd ?? false,
        showSteamId: r.showSteamId ?? false,
        showVac: r.showVac ?? false,
        showLimited: r.showLimited ?? false,
        showLastLogin: r.showLastLogin ?? false,
        showAccUsername: r.showAccUsername ?? false,
        collectInfo: r.collectInfo ?? false,
        showShortNotes: r.showShortNotes ?? false,
        note: r.note ?? "",
        avatarFrameUrl: r.avatarFrameUrl,
        miniProfileHtml: r.miniProfileHtml,
        showMiniProfile: r.showMiniProfile ?? false,
        showAvatarFrame: r.showAvatarFrame ?? false,
        syncError: r.syncError ?? "",
        tags: r.tags,
        manualProfileImage: r.manualProfileImage ?? false,
      })) as SteamAccountRow[];
    },
    swapTo: (id: string) => SteamService.SwapToSteamAccount(id, -1, []),
    saveOrder: (ids: string[]) => SteamService.SaveSteamAccountOrder(ids),
    addNew: () => SteamService.SteamAddNew(),
    forget: (id: string) => SteamService.ForgetSteamAccount(id),
    rename: async (id: string, newName: string) => {
      await BasicService.RenameAccount("Steam", id, newName);
    },
    changeImage: (id: string, path: string) => SteamService.ChangeAccountImage(id, path),
    clearManualImage: (id: string) => SteamService.ClearManualAccountProfileImage(id),
    getNote: (id: string) => BasicService.GetAccountNote("Steam", id),
    setNote: (id: string, note: string) => BasicService.SetAccountNote("Steam", id, note),
    launch: () => SteamService.LaunchSteam(),
    refreshOnWindowFocus: true,
    refreshAccounts: () => SteamService.RefreshAllSteamImages(),
    refreshAllProfileImages: () => SteamService.RefreshAllSteamImages(),

    buildMenu: (_acc, shared) => buildSteamExtraMenuAdapter(_acc as SteamAccountRow, shared),

    updateEventName: "steam-account-updated",
    buildPatch: (raw: unknown) =>
      raw instanceof AccountPatch ? raw : AccountPatch.createFrom(raw as Record<string, unknown>),
    patchTargetId: (patch: unknown) => {
      const p = patch as { steamId64?: string };
      return (p.steamId64 ?? "").trim();
    },
    applyPatch: (patch: unknown, account: SteamAccountRow) => {
      const p = patch as SteamAccountPatch;
      const nextUrl = p.imageUrl != null ? String(p.imageUrl).trim() : account.imageUrl;
      const errMsg = typeof p.error === "string" ? p.error : account.syncError ?? "";
      const nextManual = typeof p.manualProfileImage === "boolean" ? p.manualProfileImage : (account.manualProfileImage ?? false);
      return {
        ...account,
        imageUrl: nextUrl,
        vac: typeof p.vac === "boolean" ? p.vac : account.vac,
        ltd: typeof p.ltd === "boolean" ? p.ltd : account.ltd,
        avatarPending: typeof p.avatarPending === "boolean" ? p.avatarPending : account.avatarPending,
        metaPending: typeof p.metaPending === "boolean" ? p.metaPending : account.metaPending,
        manualProfileImage: nextManual, syncError: errMsg,
        displayName: typeof p.displayName === "string" && p.displayName.trim() !== "" ? p.displayName.trim() : account.displayName ?? "",
        staticImageUrl: typeof p.staticImageUrl === "string" && p.staticImageUrl.trim() !== "" ? p.staticImageUrl.trim() : account.staticImageUrl ?? "",
        avatarFrameUrl: typeof p.avatarFrameUrl === "string" && p.avatarFrameUrl.trim() !== "" ? p.avatarFrameUrl.trim() : account.avatarFrameUrl ?? "",
        miniProfileHtml: typeof p.miniProfileHtml === "string" && p.miniProfileHtml.trim() !== "" ? p.miniProfileHtml.trim() : account.miniProfileHtml ?? "",
        showMiniProfile: typeof p.showMiniProfile === "boolean" ? p.showMiniProfile : account.showMiniProfile,
        showAvatarFrame: typeof p.showAvatarFrame === "boolean" ? p.showAvatarFrame : account.showAvatarFrame,
      } as SteamAccountRow;
    },

    searchHay: (a: SteamAccountRow, trimmed: string) => {
      const parts = [a.displayName ?? "", a.personaName ?? "", a.note ?? ""];
      if (a.showAccUsername) parts.push(a.accountName ?? "");
      if (trimmed.toLowerCase().split(/\s+/).some((w) => /^\d{5,}$/.test(w))) parts.push(a.steamId64 ?? "");
      return parts.join("\n");
    },

    gameSearchRows: (q: string) => {
      const trimmed = q.trim();
      if (!trimmed) return [];
      return installedGames
        .filter((g) => fuzzyWordsMatch(trimmed, g.name))
        .slice(0, 5)
        .map((g) => ({ key: `g:${String(g.appId).trim()}`, title: g.name, badge: get(t)("Search_Section_Game"), isCategory: true, accountIconUrl: resolveSteamGameSearchIcon(g) }));
    },
    gameSearchHint: get(t)("Search_Hint_Games"),

    loginAndLaunchGame: async (accountId: string, appId: string) => {
      await SteamService.LoginAndLaunchGame(accountId, -1, appId);
    },

    onAfterLoad: async (accounts: SteamAccountRow[], ctx) => {
      steamIds = accounts.map((r) => r.steamId64);
      if (!ctx?.hadCachedAccounts || ctx.enrichChanged) {
        SteamService.StartSteamProfileRefresh();
      }
      try { await refreshGameDataAppSets(steamIds); } catch {}
    },
  } satisfies PlatformAccountAdapter<SteamAccountRow>;

  onMount(() => {
    void (async () => {
      try {
        const rows = await SteamService.GetInstalledGames();
        installedGames = rows
          .filter((r) => !STEAM_CONTEXT_MENU_HIDDEN_APP_IDS.has(String(r.appId).trim()))
          .map((r) => ({ appId: r.appId, name: r.name }));
      } catch { installedGames = []; }
    })();

    void (async () => {
      try {
        const list = await Shortcuts.ListShortcuts("Steam");
        applyShortcutIconsFromShortcutList(list as unknown[]);
      } catch {}
    })();

    offShortcutsUpdated = Events.On("shortcuts-updated", (ev) => {
      try {
        const raw = ev.data;
        const p = raw instanceof ListPayload ? raw : ListPayload.createFrom(raw as Record<string, unknown>);
        if (p.platformKey !== name) return;
        applyShortcutIconsFromShortcutList(p.shortcuts ?? []);
      } catch {}
    });
  });

  onDestroy(() => {
    offShortcutsUpdated?.();
  });
</script>

<div class="main-content platform-accounts-root">
  <div class="steam-tabs" role="tablist">
    <button
      type="button"
      role="tab"
      class="steam-tab"
      class:active={$steamPageTab === "accounts"}
      aria-selected={$steamPageTab === "accounts"}
      on:click={() => steamPageTab.set("accounts")}
    >
      {$t("Steam_Tab_Accounts")}
    </button>
    <button
      type="button"
      role="tab"
      class="steam-tab"
      class:active={$steamPageTab === "games"}
      aria-selected={$steamPageTab === "games"}
      on:click={() => steamPageTab.set("games")}
    >
      {$t("Steam_Tab_Games")}
    </button>
  </div>

  <!-- Both stay mounted: rebuilding the account list on every tab switch throws
       away its avatars and enrichment, and the games grid its artwork. -->
  <div class="steam-tab-panel" class:hidden={$steamPageTab !== "games"}>
    <SteamGamesView />
  </div>

  <div class="steam-tab-panel" class:hidden={$steamPageTab !== "accounts"}>
    <PlatformAccountsBase {name} {adapter} />
  </div>
</div>

<style lang="scss">
  /* This element is the page's scroll container. Reserve the gutter on the
     scrollbar's own side only: `both-edges` would add the same reservation on
     the left as well, which pushes the content right and makes the two margins
     less equal, not more. With plain `stable` the scrollbar lives outside the
     content padding, so the grid keeps the same gap either side. */
  .platform-accounts-root {
    scrollbar-gutter: stable;
  }

  .steam-tabs {
    display: flex;
    gap: 0.25rem;
    margin: 0 0 0.25rem;
    border-bottom: 1px solid var(--border-bar-bg, #444);
  }

  .steam-tab {
    padding: 0.45rem 1.1rem;
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

  .steam-tab-panel.hidden {
    display: none;
  }
</style>
