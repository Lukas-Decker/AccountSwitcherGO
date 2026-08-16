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
    gap: 0.375rem 0.5625rem;
    padding: 0.1875rem 0 0.375rem;
  }

  .riot-link__label {
    padding: 0;
    white-space: nowrap;
  }

  /*
     A shared floor rather than a height on either field.

     Setting height on the select was the bug: .modal-input pairs 8px of padding
     with a 24px line box, so 1.65rem left an 8.4px content box and cut the text
     through the middle. Chromium also ignores line-height on a select, so the
     two fields size by different rules and do not agree on their own: the input
     lands on 42px here and 36px on a page that loads Settings.scss, the select
     on 34px. min-height on both makes them the same height everywhere while
     still letting either grow if it needs to.
  */
  .riot-link input,
  .riot-link select {
    min-height: 2.625rem;
  }

  .riot-link__select {
    cursor: pointer;
  }

  .riot-link__hint {
    margin: 0 0 0.375rem;
    font-size: 0.6375rem;
    opacity: 0.75;
  }
</style>
