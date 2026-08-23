<script lang="ts">
  import { onMount } from "svelte";
  import { get } from "svelte/store";
  import * as GameLibraryService from "../../bindings/account-switcher/internal/gamelib/service.js";
  import { t } from "../stores/i18n";
  import { censoredName } from "../stores/streamerMode";
  import { pushToast } from "../stores/toast";
  import { formatToastWithError } from "../lib/formatWailsError";
  import { reportLaunchFailure } from "../lib/adminFlow";
  import { fuzzyWordsMatch } from "../lib/searchFuzzy";

  type Owner = {
    accountId: string;
    accountName: string;
    source: string;
    confidence: string;
    installedBy: boolean;
    playtimeMinutes: number;
    lastPlayed: string;
  };

  type Game = {
    platformKey: string;
    gameId: string;
    name: string;
    artUrl: string;
    installed: boolean;
    installPath: string;
    sizeOnDisk: number;
    owners: Owner[];
    sources: string[];
  };

  /** Which platform's library to show. */
  export let platformKey: string;
  /**
   * Switches to an account and, when launch is true, starts the game.
   * Supplied by the page because only it knows how its platform launches.
   */
  export let onSwitch: (game: Game, accountId: string, launch: boolean) => Promise<void>;
  /** Whether this platform can start a specific game rather than just the launcher. */
  export let canLaunchGame = true;
  /** Whether online enrichment is offered. Only Steam has a keyless source. */
  export let supportsOnline = false;

  let games: Game[] = [];
  let warnings: string[] = [];
  let unsupported = false;
  let loading = true;
  let search = "";
  let installedOnly = false;
  let includeOnline = false;
  let launchAfterSwitch = true;
  let busyGameId = "";
  /**
   * Which game has its account picker open. A game whose owner is known
   * switches straight away; the picker is for the ambiguous ones.
   */
  let pickerGameId = "";

  /**
   * Prefix for the checkbox ids.
   *
   * The app styles a checkbox through a sibling label, which needs an id to
   * point at; a wrapping label leaves the control invisible. The prefix keeps
   * two mounted views from claiming the same id and toggling each other.
   */
  const uid = `games-${Math.random().toString(36).slice(2, 8)}`;

  $: filtered = games
    .filter((g) => !installedOnly || g.installed)
    .filter((g) => {
      const term = search.trim();
      if (!term) return true;
      return fuzzyWordsMatch(term, g.name) || g.gameId.toLowerCase().includes(term.toLowerCase());
    });

  /**
   * The account a click should act on: the one that installed the game when a
   * launcher named it, otherwise the single owner. Anything else needs a pick,
   * because choosing for the user would log them into the wrong account.
   */
  function decidedOwner(game: Game): Owner | null {
    const installer = game.owners.find((o) => o.installedBy);
    if (installer) return installer;
    if (game.owners.length === 1) return game.owners[0];
    return null;
  }

  function ownerLabel(game: Game): string {
    const owner = decidedOwner(game);
    if (owner) {
      return owner.installedBy && game.owners.length > 1
        ? get(t)("Games_InstalledBy", { name: ownerName(owner) })
        : ownerName(owner);
    }
    if (game.owners.length > 1) return get(t)("Games_OwnerCount", { count: game.owners.length });
    return game.installed ? get(t)("Games_OwnerUnknown") : get(t)("Games_NotPlayedHere");
  }

  function ownerName(owner: Owner): string {
    return get(censoredName)(owner.accountName || owner.accountId);
  }

  function confidenceHint(owner: Owner): string {
    switch (owner.confidence) {
      case "exact":
        return get(t)("Games_ConfidenceExact");
      case "strong":
        return get(t)("Games_ConfidenceStrong");
      case "weak":
        return get(t)("Games_ConfidenceWeak");
      case "inferred":
        return get(t)("Games_ConfidenceInferred");
      default:
        return "";
    }
  }

  function playtimeLabel(owner: Owner): string {
    if (!owner.playtimeMinutes) return "";
    return get(t)("Games_Playtime", { hours: Math.round(owner.playtimeMinutes / 60) });
  }

  /** Resolves this platform's library. GetPlatformGames always runs a fresh
   * pass, so the refresh button and the online toggle both just call it again. */
  async function load(): Promise<void> {
    loading = true;
    try {
      const res = await GameLibraryService.GetPlatformGames(platformKey, includeOnline);
      games = ((res?.games ?? []) as Game[]) ?? [];
      warnings = res?.warnings ?? [];
      unsupported = Boolean(res?.unsupported);
    } catch (e) {
      pushToast({
        type: "error",
        message: formatToastWithError(get(t)("Toast_GamesLoadFailed"), e),
        duration: 8000,
      });
      games = [];
      warnings = [];
    } finally {
      loading = false;
    }
  }

  async function switchTo(game: Game, accountId: string): Promise<void> {
    if (busyGameId) return;
    busyGameId = game.gameId;
    pickerGameId = "";
    try {
      await onSwitch(game, accountId, launchAfterSwitch && canLaunchGame);
    } catch (e) {
      await reportLaunchFailure(e, game.platformKey);
    } finally {
      busyGameId = "";
    }
  }

  function onGameClick(game: Game): void {
    if (game.owners.length === 0) {
      pushToast({ type: "info", message: get(t)("Games_NoOwners"), duration: 5000 });
      return;
    }
    const owner = decidedOwner(game);
    if (owner) {
      void switchTo(game, owner.accountId);
      return;
    }
    pickerGameId = pickerGameId === game.gameId ? "" : game.gameId;
  }

  onMount(() => {
    void load();
  });
</script>

<div class="games-view">
  <div class="games-toolbar">
    <input
      class="games-search"
      type="search"
      bind:value={search}
      placeholder={$t("Games_Search")}
      aria-label={$t("Games_Search")}
    />
    <span class="games-toggle">
      <input type="checkbox" id="{uid}-installed" bind:checked={installedOnly} />
      <label for="{uid}-installed" class="games-toggle-box" aria-hidden="true"></label>
      <label for="{uid}-installed" class="games-toggle-text">{$t("Games_FilterInstalled")}</label>
    </span>
    {#if canLaunchGame}
      <span class="games-toggle">
        <input type="checkbox" id="{uid}-launch" bind:checked={launchAfterSwitch} />
        <label for="{uid}-launch" class="games-toggle-box" aria-hidden="true"></label>
        <label for="{uid}-launch" class="games-toggle-text">{$t("Games_LaunchAfterSwitch")}</label>
      </span>
    {/if}
    {#if supportsOnline}
      <span class="games-toggle" title={$t("Games_OnlineHint")}>
        <input
          type="checkbox"
          id="{uid}-online"
          bind:checked={includeOnline}
          on:change={() => void load()}
        />
        <label for="{uid}-online" class="games-toggle-box" aria-hidden="true"></label>
        <label for="{uid}-online" class="games-toggle-text">{$t("Games_IncludeOnline")}</label>
      </span>
    {/if}
    <button type="button" class="games-refresh" on:click={() => void load()} disabled={loading}>
      {$t("Games_Refresh")}
    </button>
  </div>

  {#each warnings as warning (warning)}
    <p class="games-warning">{warning}</p>
  {/each}

  {#if loading}
    <p class="games-empty">{$t("Status_Loading")}</p>
  {:else if unsupported}
    <p class="games-empty">{$t("Games_Unsupported")}</p>
  {:else if filtered.length === 0}
    <p class="games-empty">{$t("Games_NoGamesPlatform")}</p>
  {:else}
    <div class="games-grid">
      {#each filtered as game (game.gameId)}
        <div class="game-card" class:busy={busyGameId === game.gameId}>
          <!-- title carries the install path because it is genuinely useful on
               hover, but it would otherwise become the accessible name and a
               screen reader would read out a path instead of the game. -->
          <button
            type="button"
            class="game-button"
            title={game.installPath || game.name}
            aria-label={`${game.name}. ${ownerLabel(game)}`}
            disabled={busyGameId !== ""}
            on:click={() => onGameClick(game)}
          >
            <span class="game-art-wrap">
              {#if game.artUrl}
                <img class="game-art" src={game.artUrl} alt="" draggable="false" loading="lazy" />
              {:else}
                <span class="game-art game-art--placeholder">{game.name}</span>
              {/if}
              {#if game.installed}
                <span class="game-badge">{$t("Games_Installed")}</span>
              {/if}
            </span>
            <span class="game-name">{game.name}</span>
            <span class="game-owners">{ownerLabel(game)}</span>
          </button>

          {#if pickerGameId === game.gameId}
            <div class="game-owner-picker">
              <span class="game-owner-picker-title">{$t("Games_PickAccount")}</span>
              {#each game.owners as owner (owner.accountId)}
                <button
                  type="button"
                  class="game-owner"
                  title={confidenceHint(owner)}
                  aria-label={`${ownerName(owner)}. ${confidenceHint(owner)}`}
                  on:click={() => void switchTo(game, owner.accountId)}
                >
                  <span class="game-owner-name">{ownerName(owner)}</span>
                  {#if playtimeLabel(owner)}
                    <span class="game-owner-meta">{playtimeLabel(owner)}</span>
                  {/if}
                </button>
              {/each}
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style lang="scss">
  .games-view {
    display: flex;
    flex-direction: column;
    gap: 0.5625rem;
    padding: 0.375rem 0.5625rem 1.125rem;
    min-height: 0;
  }

  .games-toolbar {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: wrap;
  }

  .games-search {
    flex: 1 1 10.5rem;
    min-width: 0;
    padding: 0.3rem 0.45rem;
    border-radius: 4px;
    border: 1px solid var(--border-bar-bg, #444);
    background: var(--backdrop-dark-25, rgba(0, 0, 0, 0.25));
    color: inherit;
    font: inherit;
  }

  .games-toggle {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    white-space: nowrap;
  }

  /* The box itself comes from the app-wide input[type="checkbox"] + label rule,
     which sizes it for the settings screens. Shrink it to sit with toolbar
     text without overpowering the row. */
  .games-toggle-box {
    width: 0.9rem;
    height: 0.9rem;
    flex: none;
  }

  .games-toggle-text {
    cursor: pointer;
  }

  .games-refresh {
    padding: 0.3rem 0.6rem;
    border-radius: 4px;
    border: 1px solid var(--border-bar-bg, #444);
    background: var(--backdrop-dark-25, rgba(0, 0, 0, 0.25));
    color: inherit;
    font: inherit;
    cursor: pointer;

    &:disabled {
      opacity: 0.6;
      cursor: default;
    }
  }

  .games-warning {
    margin: 0 0.1875rem;
    font-size: 0.6375rem;
    opacity: 0.8;
  }

  .games-empty {
    opacity: 0.75;
    margin: 0.75rem 0.1875rem;
  }

  .games-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(6.75rem, 1fr));
    gap: 0.6375rem;
  }

  .game-card {
    position: relative;
    min-width: 0;

    &.busy {
      opacity: 0.6;
    }
  }

  .game-button {
    display: flex;
    flex-direction: column;
    align-items: stretch;
    gap: 0.2625rem;
    width: 100%;
    padding: 0;
    background: none;
    border: none;
    color: inherit;
    font: inherit;
    text-align: left;
    cursor: pointer;

    &:hover .game-art,
    &:focus-visible .game-art {
      outline: 2px solid var(--accent, #f90);
      transform: translateY(-2px);
    }

    &:disabled {
      cursor: default;
    }
  }

  .game-art-wrap {
    position: relative;
    display: block;
  }

  .game-art {
    display: block;
    width: 100%;
    aspect-ratio: 2 / 3;
    object-fit: cover;
    border-radius: 6px;
    background: var(--backdrop-dark-25, rgba(0, 0, 0, 0.3));
    transition:
      transform 0.12s ease,
      outline-color 0.12s ease;
  }

  .game-art--placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0.375rem;
    font-size: 0.6rem;
    text-align: center;
    overflow: hidden;
    word-break: break-word;
  }

  .game-badge {
    position: absolute;
    top: 0.225rem;
    left: 0.225rem;
    padding: 0.075rem 0.225rem;
    border-radius: 3px;
    background: var(--accent, #f90);
    color: #000;
    font-size: 0.5rem;
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .game-name {
    font-size: 0.6375rem;
    line-height: 1.2;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .game-owners {
    font-size: 0.5625rem;
    opacity: 0.7;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .game-owner-picker {
    position: absolute;
    z-index: 5;
    left: 0;
    right: 0;
    top: 100%;
    margin-top: 0.1875rem;
    display: flex;
    flex-direction: column;
    gap: 0.1125rem;
    padding: 0.225rem;
    border-radius: 6px;
    border: 1px solid var(--border-bar-bg, #444);
    background: var(--modal-bg, #23252b);
    box-shadow: 0 6px 18px rgba(0, 0, 0, 0.45);
  }

  .game-owner-picker-title {
    padding: 0.075rem 0.375rem 0.15rem;
    font-size: 0.5625rem;
    opacity: 0.7;
  }

  .game-owner {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 0.375rem;
    padding: 0.2625rem 0.375rem;
    border: none;
    border-radius: 4px;
    background: none;
    color: inherit;
    font: inherit;
    text-align: left;
    cursor: pointer;

    &:hover,
    &:focus-visible {
      background: var(--backdrop-dark-25, rgba(255, 255, 255, 0.08));
    }
  }

  .game-owner-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .game-owner-meta {
    flex: none;
    font-size: 0.5625rem;
    opacity: 0.65;
  }
</style>
