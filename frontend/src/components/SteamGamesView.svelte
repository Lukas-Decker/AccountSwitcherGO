<script lang="ts">
  import { onMount } from "svelte";
  import { get } from "svelte/store";
  import * as SteamService from "../../bindings/account-switcher/internal/steam/steamservice.js";
  import { t } from "../stores/i18n";
  import { pushToast } from "../stores/toast";
  import { formatToastWithError } from "../lib/formatWailsError";
  import { reportLaunchFailure } from "../lib/adminFlow";
  import { fuzzyWordsMatch } from "../lib/searchFuzzy";

  type OwnedGame = {
    appId: string;
    name: string;
    iconUrl: string;
    owners: string[];
    installed: boolean;
  };

  type OwnerInfo = { steamId64: string; name: string };

  let games: OwnedGame[] = [];
  let ownersById: Record<string, OwnerInfo> = {};
  let loading = true;
  let search = "";
  let launchAfterSwitch = true;
  let busyAppId = "";
  /** Which game has its owner picker open. Only games with more than one owner
   * need it; a single-owner game switches straight away. */
  let pickerAppId = "";

  $: filtered = search.trim()
    ? games.filter((g) => fuzzyWordsMatch(search, g.name) || g.appId.includes(search.trim()))
    : games;

  function ownerName(id: string): string {
    return ownersById[id]?.name || id;
  }

  async function load(): Promise<void> {
    loading = true;
    try {
      const [list, accounts] = await Promise.all([
        SteamService.GetOwnedGames(),
        SteamService.GetSteamAccountsList().catch(() => []),
      ]);
      games = (list ?? []) as OwnedGame[];
      const map: Record<string, OwnerInfo> = {};
      for (const a of (accounts ?? []) as Array<{ steamId64: string; displayName?: string; personaName?: string; accountName?: string }>) {
        map[a.steamId64] = {
          steamId64: a.steamId64,
          name: (a.displayName || a.personaName || a.accountName || a.steamId64).trim(),
        };
      }
      ownersById = map;
    } catch (e) {
      pushToast({
        type: "error",
        message: formatToastWithError(get(t)("Toast_GamesLoadFailed"), e),
        duration: 8000,
      });
      games = [];
    } finally {
      loading = false;
    }
  }

  async function switchTo(game: OwnedGame, steamId64: string): Promise<void> {
    if (busyAppId) return;
    busyAppId = game.appId;
    pickerAppId = "";
    try {
      if (launchAfterSwitch) {
        await SteamService.LoginAndLaunchGame(steamId64, -1, game.appId);
        pushToast({
          type: "success",
          message: get(t)("Toast_StartedGame", { program: game.name }),
          duration: 4000,
        });
      } else {
        await SteamService.SwapToSteamAccount(steamId64, -1, []);
      }
    } catch (e) {
      await reportLaunchFailure(e, "Steam");
    } finally {
      busyAppId = "";
    }
  }

  function onGameClick(game: OwnedGame): void {
    if (game.owners.length === 0) {
      pushToast({ type: "info", message: get(t)("Games_NoOwners"), duration: 5000 });
      return;
    }
    if (game.owners.length === 1) {
      void switchTo(game, game.owners[0]);
      return;
    }
    pickerAppId = pickerAppId === game.appId ? "" : game.appId;
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
    <label class="games-launch-toggle">
      <input type="checkbox" bind:checked={launchAfterSwitch} />
      <span>{$t("Games_LaunchAfterSwitch")}</span>
    </label>
  </div>

  {#if loading}
    <p class="games-empty">{$t("Status_Loading")}</p>
  {:else if filtered.length === 0}
    <p class="games-empty">{$t("Games_NoGames")}</p>
  {:else}
    <div class="games-grid">
      {#each filtered as game (game.appId)}
        <div class="game-card" class:busy={busyAppId === game.appId}>
          <button
            type="button"
            class="game-button"
            title={game.name}
            disabled={busyAppId !== ""}
            on:click={() => onGameClick(game)}
          >
            {#if game.iconUrl}
              <img class="game-art" src={game.iconUrl} alt="" draggable="false" loading="lazy" />
            {:else}
              <span class="game-art game-art--placeholder">{game.name}</span>
            {/if}
            <span class="game-name">{game.name}</span>
            <span class="game-owners">
              {#if game.owners.length === 0}
                {$t("Games_NotPlayedHere")}
              {:else if game.owners.length === 1}
                {ownerName(game.owners[0])}
              {:else}
                {$t("Games_OwnerCount", { count: game.owners.length })}
              {/if}
            </span>
          </button>

          {#if pickerAppId === game.appId}
            <div class="game-owner-picker">
              {#each game.owners as id (id)}
                <button type="button" class="game-owner" on:click={() => void switchTo(game, id)}>
                  {ownerName(id)}
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
    gap: 0.75rem;
    padding: 0.5rem 0.25rem 1rem;
    min-height: 0;
  }

  .games-toolbar {
    display: flex;
    align-items: center;
    gap: 1rem;
    flex-wrap: wrap;
  }

  .games-search {
    flex: 1 1 14rem;
    min-width: 0;
    padding: 0.4rem 0.6rem;
    border-radius: 4px;
    border: 1px solid var(--border-bar-bg, #444);
    background: var(--backdrop-dark-25, rgba(0, 0, 0, 0.25));
    color: inherit;
    font: inherit;
  }

  .games-launch-toggle {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    cursor: pointer;
    white-space: nowrap;
  }

  .games-empty {
    opacity: 0.75;
    margin: 1rem 0.25rem;
  }

  .games-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(9rem, 1fr));
    gap: 0.85rem;
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
    gap: 0.35rem;
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
    padding: 0.5rem;
    font-size: 0.8rem;
    text-align: center;
    overflow: hidden;
    word-break: break-word;
  }

  .game-name {
    font-size: 0.85rem;
    line-height: 1.2;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .game-owners {
    font-size: 0.75rem;
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
    margin-top: 0.25rem;
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
    padding: 0.3rem;
    border-radius: 6px;
    border: 1px solid var(--border-bar-bg, #444);
    background: var(--modal-bg, #23252b);
    box-shadow: 0 6px 18px rgba(0, 0, 0, 0.45);
  }

  .game-owner {
    padding: 0.35rem 0.5rem;
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
</style>
