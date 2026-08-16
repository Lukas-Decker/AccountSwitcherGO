<script lang="ts">
  /*
    Links a saved account to a Riot ID.

    Both halves are asked for together, and the region is picked from a list.
    Two sequential prompts made the region a memory test with no feedback: the
    platform ids are not the names anyone uses, "tr" is wrong where "tr1" is
    right, and a typo only surfaced later as a card that would not load.
  */
  import { createEventDispatcher, onMount, tick } from "svelte";
  import { t } from "../../stores/i18n";

  export let riotId = "";
  export let region = "";
  /** Platform id and label for every region the backend knows. */
  export let regions: { platform: string; display: string }[] = [];
  export let positiveLabel = "";
  export let negativeLabel = "";

  const dispatch = createEventDispatcher<{ resolve: { riotId: string; region: string } | null }>();

  let idInput: HTMLInputElement | undefined;
  let value = riotId;
  let chosen = region;

  // An account with no region yet gets the first one rather than an empty
  // selection, so the field always holds something valid.
  $: if (!chosen && regions.length > 0) {
    chosen = regions[0].platform;
  }

  function save(): void {
    dispatch("resolve", { riotId: value.trim(), region: chosen });
  }

  function cancel(): void {
    dispatch("resolve", null);
  }

  onMount(() => {
    void tick().then(() =>
      requestAnimationFrame(() => {
        idInput?.focus();
        idInput?.select();
      }),
    );
  });
</script>

<div class="modal-block">
  <div class="riot-link">
    <label class="riot-link__label" for="riot-link-id">{$t("Riot_RiotId")}</label>
    <input
      id="riot-link-id"
      bind:this={idInput}
      bind:value
      type="text"
      class="modal-input"
      autocomplete="off"
      spellcheck="false"
      placeholder="GameName#TAG"
      on:keydown={(e) => e.key === "Enter" && save()}
    />

    <label class="riot-link__label" for="riot-link-region">{$t("Riot_Region")}</label>
    <select id="riot-link-region" class="modal-input riot-link__select" bind:value={chosen}>
      {#each regions as r (r.platform)}
        <!-- Both halves shown: the label is what the user recognises, the
             platform id is what they would otherwise have had to guess. -->
        <option value={r.platform}>{r.display} ({r.platform})</option>
      {/each}
    </select>
  </div>

  <p class="riot-link__hint">{$t("Riot_ClearHint")}</p>

  <div class="modal-inline-actions settingsCol inputAndButton">
    <button type="button" class="btnicontext" on:click={cancel}>{negativeLabel}</button>
    <span class="modal-actions-spacer"></span>
    <button type="button" class="btnicontext modal-primary" on:click={save}>{positiveLabel}</button>
  </div>
</div>

<style lang="scss">
  .riot-link {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr);
    align-items: center;
    gap: 0.5rem 0.75rem;
    padding: 0.25rem 0 0.5rem;
  }

  .riot-link__label {
    padding: 0;
    white-space: nowrap;
  }

  .riot-link__select {
    height: 2.2rem;
    cursor: pointer;
  }

  .riot-link__hint {
    margin: 0 0 0.5rem;
    font-size: 0.85rem;
    opacity: 0.75;
  }
</style>
