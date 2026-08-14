<script lang="ts">
  import { onMount } from "svelte";
  import { t } from "../../stores/i18n";
  import { pushToast } from "../../stores/toast";
  import { formatToastWithError } from "../../lib/formatWailsError";
  import * as RiotService from "../../../bindings/account-switcher/internal/riotservice/service.js";

  type KeyInfo = Awaited<ReturnType<typeof RiotService.KeyInfo>>;

  let info: KeyInfo | null = null;
  let storeAvailable = true;
  let keyInput = "";
  let busy = false;

  async function load(): Promise<void> {
    try {
      storeAvailable = await RiotService.CredentialStoreAvailable();
      info = await RiotService.KeyInfo();
    } catch (e) {
      pushToast({ type: "error", message: formatToastWithError($t("Riot_LoadFailed"), e), duration: 8000 });
    }
  }

  async function save(): Promise<void> {
    if (busy) return;
    busy = true;
    try {
      await RiotService.SetAPIKey(keyInput);
      keyInput = "";
      await load();
    } catch (e) {
      pushToast({ type: "error", message: formatToastWithError($t("Riot_KeySaveFailed"), e), duration: 8000 });
    } finally {
      busy = false;
    }
  }

  onMount(() => { void load(); });

  // The tier is inferred from the quota Riot hands back, never asserted: there is
  // no endpoint that says what kind of key you hold.
  $: tierLabel =
    info?.tier === "development"
      ? $t("Riot_Key_Development")
      : info?.tier === "elevated"
        ? $t("Riot_Key_Elevated")
        : $t("Riot_Key_Unknown");
</script>

<h2 class="SettingsHeader">{$t("Riot_CardTitle")}</h2>

<div class="riot-key">
  <label class="riot-key__field">
    <span>{$t("Riot_ApiKey")}</span>
    <input
      type="password"
      bind:value={keyInput}
      placeholder={info?.present ? $t("Riot_ApiKey_Stored") : "RGAPI-..."}
      spellcheck="false"
      autocomplete="off"
    />
  </label>
  <button type="button" class="btnicontext" disabled={busy} on:click={() => void save()}>
    {keyInput.trim() === "" && info?.present ? $t("Riot_ApiKey_Clear") : $t("Riot_Save")}
  </button>

  {#if info?.present}
    <p class="riot-key__status">
      <strong>{tierLabel}</strong>
      {#if info.appRateLimit}
        <span class="riot-key__limit">{$t("Riot_Key_Quota", { quota: info.appRateLimit })}</span>
      {/if}
    </p>
    {#if info.error}
      <p class="riot-key__error">{info.error}</p>
    {:else if info.tier === "development"}
      <p class="riot-key__hint">{$t("Riot_Key_Development_Hint")}</p>
    {:else if info.liveAllowed}
      <p class="riot-key__hint">{$t("Riot_Key_Elevated_Hint")}</p>
    {/if}
  {/if}

  <p class="riot-key__hint">
    {storeAvailable ? $t("Riot_ApiKey_Hint") : $t("Riot_ApiKey_NoStore")}
  </p>
  <p class="riot-key__hint">{$t("Riot_Key_ClientHint")}</p>
</div>

<style lang="scss">
  .riot-key {
    display: grid;
    gap: 0.45rem;
    max-width: 34rem;
    margin-bottom: 1rem;
  }

  .riot-key__field {
    display: grid;
    gap: 0.2rem;

    span {
      font-size: 0.82rem;
      opacity: 0.8;
    }

    input {
      padding: 0.35rem 0.5rem;
    }
  }

  .riot-key__status {
    margin: 0.2rem 0 0;
    font-size: 0.9rem;
  }

  .riot-key__limit {
    margin-left: 0.4rem;
    font-family: ui-monospace, monospace;
    font-size: 0.8rem;
    opacity: 0.7;
  }

  .riot-key__hint {
    margin: 0;
    font-size: 0.85rem;
    opacity: 0.75;
  }

  .riot-key__error {
    margin: 0;
    font-size: 0.85rem;
    color: var(--danger, #ff6b6b);
  }
</style>
