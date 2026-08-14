<script lang="ts">
  import { onDestroy } from "svelte";
  import { t } from "../stores/i18n";
  import { pushToast } from "../stores/toast";
  import { formatToastWithError } from "../lib/formatWailsError";
  import { selectedAccount } from "../stores/platformPage";
  import { streamerMode } from "../stores/streamerMode";
  import * as RiotService from "../../bindings/account-switcher/internal/riotservice/service.js";

  type Card = Awaited<ReturnType<typeof RiotService.GetCard>>;
  type Region = Awaited<ReturnType<typeof RiotService.Regions>>[number];

  let card: Card | null = null;
  let regions: Region[] = [];
  let loading = false;
  let editing = false;
  let saving = false;

  let riotIdInput = "";
  let regionInput = "";

  let hasKey = false;
  let keyStoreAvailable = true;
  let keyInput = "";
  let savingKey = false;

  // The card follows whichever account is selected, so switching rows in the
  // list re-targets it rather than showing a stale profile.
  $: uniqueId = $selectedAccount.uniqueId;
  $: platformKey = $selectedAccount.platformKey;
  $: if (platformKey === "Riot Games") void load(uniqueId);

  let lastLoaded: string | null = null;

  async function load(id: string): Promise<void> {
    if (id === lastLoaded) return;
    lastLoaded = id;
    editing = false;
    if (!id) {
      card = null;
      return;
    }
    loading = true;
    try {
      card = await RiotService.GetCard(id);
      riotIdInput = card?.riotId ?? "";
      regionInput = card?.region ?? "";
      hasKey = card?.hasKey ?? (await RiotService.HasAPIKey());
    } catch (e) {
      card = null;
      pushToast({ type: "error", message: formatToastWithError($t("Riot_LoadFailed"), e), duration: 8000 });
    } finally {
      loading = false;
    }
  }

  async function ensureRegions(): Promise<void> {
    if (regions.length > 0) return;
    try {
      regions = await RiotService.Regions();
    } catch {
      regions = [];
    }
  }

  async function beginEdit(): Promise<void> {
    await ensureRegions();
    keyStoreAvailable = await RiotService.CredentialStoreAvailable();
    hasKey = await RiotService.HasAPIKey();
    riotIdInput = card?.riotId ?? "";
    regionInput = card?.region || regions[0]?.platform || "";
    editing = true;
  }

  async function save(): Promise<void> {
    if (saving) return;
    saving = true;
    try {
      await RiotService.SetAccountLink(uniqueId, riotIdInput, regionInput);
      editing = false;
      lastLoaded = null;
      await load(uniqueId);
    } catch (e) {
      pushToast({ type: "error", message: formatToastWithError($t("Riot_SaveFailed"), e), duration: 8000 });
    } finally {
      saving = false;
    }
  }

  async function saveKey(): Promise<void> {
    if (savingKey) return;
    savingKey = true;
    try {
      await RiotService.SetAPIKey(keyInput);
      keyInput = "";
      hasKey = await RiotService.HasAPIKey();
      lastLoaded = null;
      await load(uniqueId);
    } catch (e) {
      pushToast({ type: "error", message: formatToastWithError($t("Riot_KeySaveFailed"), e), duration: 8000 });
    } finally {
      savingKey = false;
    }
  }

  async function refresh(): Promise<void> {
    lastLoaded = null;
    await load(uniqueId);
  }

  function titleLabel(title: string): string {
    switch (title) {
      case "tft": return "TFT";
      case "valorant": return "VALORANT";
      default: return "League";
    }
  }

  onDestroy(() => { card = null; });
</script>

{#if platformKey === "Riot Games" && uniqueId}
  <section class="riot-card" class:riot-card--empty={!card?.linked}>
    <header class="riot-card__head">
      <h3>{$t("Riot_CardTitle")}</h3>
      <div class="riot-card__headActions">
        {#if card?.linked && !editing}
          <button type="button" class="btnicontext" disabled={loading} on:click={() => void refresh()}>
            {$t("Riot_Refresh")}
          </button>
        {/if}
        <button type="button" class="btnicontext" on:click={() => (editing ? (editing = false) : void beginEdit())}>
          {editing ? $t("Button_Cancel") : card?.linked ? $t("Riot_Edit") : $t("Riot_Link")}
        </button>
      </div>
    </header>

    {#if editing}
      <div class="riot-card__form">
        <label class="riot-card__field">
          <span>{$t("Riot_RiotId")}</span>
          <input type="text" bind:value={riotIdInput} placeholder="GameName#TAG" spellcheck="false" />
        </label>
        <label class="riot-card__field">
          <span>{$t("Riot_Region")}</span>
          <select bind:value={regionInput}>
            {#each regions as r (r.platform)}
              <option value={r.platform}>{r.display}</option>
            {/each}
          </select>
        </label>
        <button type="button" class="btnicontext" disabled={saving} on:click={() => void save()}>
          {$t("Riot_Save")}
        </button>
        <p class="riot-card__hint">{$t("Riot_ClearHint")}</p>

        <hr />

        <label class="riot-card__field">
          <span>{$t("Riot_ApiKey")}</span>
          <input
            type="password"
            bind:value={keyInput}
            placeholder={hasKey ? $t("Riot_ApiKey_Stored") : "RGAPI-..."}
            spellcheck="false"
            autocomplete="off"
          />
        </label>
        <button type="button" class="btnicontext" disabled={savingKey} on:click={() => void saveKey()}>
          {keyInput.trim() === "" && hasKey ? $t("Riot_ApiKey_Clear") : $t("Riot_Save")}
        </button>
        <p class="riot-card__hint">
          {keyStoreAvailable ? $t("Riot_ApiKey_Hint") : $t("Riot_ApiKey_NoStore")}
        </p>
      </div>
    {:else if !card?.linked}
      <p class="riot-card__hint">{$t("Riot_NotLinked")}</p>
    {:else}
      <div class="riot-card__body">
        <div class="riot-card__identity">
          {#if card.iconUrl && !$streamerMode}
            <img class="riot-card__icon" src={card.iconUrl} alt="" draggable="false" />
          {/if}
          <div>
            <!-- Streamer mode exists to keep exactly this off a broadcast. -->
            <p class="riot-card__name">{$streamerMode ? $t("StreamerMode_HiddenAccount") : card.riotId}</p>
            {#if card.level > 0}
              <p class="riot-card__level">{$t("Riot_Level", { level: card.level })}</p>
            {/if}
          </div>
        </div>

        {#if card.error}
          <p class="riot-card__error">{card.error}</p>
        {:else if !card.hasKey}
          <p class="riot-card__hint">{$t("Riot_NoKeyHint")}</p>
        {:else if card.ranks.length === 0}
          <p class="riot-card__hint">{$t("Riot_Unranked")}</p>
        {/if}

        {#if card.ranks.length > 0}
          <ul class="riot-card__ranks">
            {#each card.ranks as rank (rank.queue)}
              <li class="riot-card__rank">
                {#if rank.emblemUrl}
                  <img src={rank.emblemUrl} alt="" draggable="false" />
                {/if}
                <span class="riot-card__rankText">{rank.display}</span>
                {#if rank.hasGames}
                  <span class="riot-card__winrate">
                    {$t("Riot_WinRate", { rate: rank.winRate, wins: rank.wins, losses: rank.losses })}
                  </span>
                {/if}
              </li>
            {/each}
          </ul>
        {/if}

        {#if card.links.length > 0}
          <div class="riot-card__links">
            {#each card.links as link (link.url)}
              <a class="riot-card__link" href={link.url} target="_blank" rel="noreferrer noopener">
                <span class="riot-card__linkTitle">{titleLabel(link.title)}</span>
                {link.site}
              </a>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </section>
{/if}

<style lang="scss">
  .riot-card {
    margin: 0 0 0.75rem;
    padding: 0.75rem 0.9rem;
    border: 1px solid var(--borderColor, rgb(255 255 255 / 12%));
    border-radius: var(--ui-radius, 4px);
    background: rgb(255 255 255 / 4%);
  }

  .riot-card__head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;

    h3 {
      margin: 0;
      font-size: 0.95rem;
    }
  }

  .riot-card__headActions {
    display: flex;
    gap: 0.4rem;
  }

  .riot-card__body {
    display: grid;
    gap: 0.6rem;
    margin-top: 0.6rem;
  }

  .riot-card__identity {
    display: flex;
    align-items: center;
    gap: 0.6rem;
  }

  .riot-card__icon {
    width: 56px;
    height: 56px;
    border-radius: 50%;
    object-fit: cover;
  }

  .riot-card__name {
    margin: 0;
    font-weight: 600;
  }

  .riot-card__level,
  .riot-card__hint {
    margin: 0.1rem 0 0;
    font-size: 0.85rem;
    opacity: 0.75;
  }

  .riot-card__error {
    margin: 0;
    font-size: 0.85rem;
    color: var(--danger, #ff6b6b);
  }

  .riot-card__ranks {
    display: grid;
    gap: 0.35rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .riot-card__rank {
    display: flex;
    align-items: center;
    gap: 0.5rem;

    img {
      width: 24px;
      height: 24px;
    }
  }

  .riot-card__winrate {
    font-size: 0.8rem;
    opacity: 0.7;
  }

  .riot-card__links {
    display: flex;
    flex-wrap: wrap;
    gap: 0.35rem;
  }

  .riot-card__link {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    padding: 0.2rem 0.5rem;
    border: 1px solid var(--borderColor, rgb(255 255 255 / 12%));
    border-radius: var(--ui-radius, 4px);
    font-size: 0.82rem;
    text-decoration: none;
  }

  .riot-card__linkTitle {
    font-size: 0.7rem;
    text-transform: uppercase;
    opacity: 0.65;
  }

  .riot-card__form {
    display: grid;
    gap: 0.5rem;
    margin-top: 0.6rem;
    max-width: 26rem;

    hr {
      width: 100%;
      margin: 0.3rem 0;
      border: 0;
      border-top: 1px solid var(--borderColor, rgb(255 255 255 / 12%));
    }
  }

  .riot-card__field {
    display: grid;
    gap: 0.2rem;

    span {
      font-size: 0.82rem;
      opacity: 0.8;
    }

    input,
    select {
      padding: 0.35rem 0.5rem;
    }
  }
</style>
