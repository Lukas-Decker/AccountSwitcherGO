<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type { ComponentType, SvelteComponent } from "svelte";
  import ModalBodyShell from "./ModalBodyShell.svelte";

  export let html: string | undefined = undefined;
  export let component: ComponentType<SvelteComponent> | undefined = undefined;
  export let componentProps: Record<string, unknown> | undefined = undefined;
  export let positiveLabel = "";
  export let negativeLabel: string | undefined = undefined;
  export let style: string = "";
  /** When set, an opt-out checkbox is shown under the body. */
  export let checkboxLabel: string | undefined = undefined;
  export let checked = false;

  const dispatch = createEventDispatcher<{ resolve: boolean | { ok: boolean; checked: boolean } }>();

  // Ticking the box is a standing yes, so declining in the same breath is
  // contradictory. Disabled rather than hidden, so the choice being removed
  // stays visible.
  $: negativeDisabled = checkboxLabel !== undefined && checked;

  function resolveWith(ok: boolean): void {
    dispatch("resolve", checkboxLabel === undefined ? ok : { ok, checked });
  }

  function positive(): void {
    resolveWith(true);
  }

  function negative(): void {
    if (negativeDisabled) return;
    resolveWith(false);
  }
</script>

<div class="modal-block">
  <ModalBodyShell
    {html}
    {component}
    {componentProps}
  />
  {#if checkboxLabel !== undefined}
    <div class="rowSetting modal-optout">
      <div class="form-check">
        <input id="modal-optout" type="checkbox" bind:checked />
        <label class="form-check-label" for="modal-optout"></label>
      </div>
      <label for="modal-optout">{checkboxLabel}</label>
    </div>
  {/if}
  <div class="modal-inline-actions settingsCol inputAndButton">
    <span class="modal-actions-spacer"></span>
    <button type="button" class="btnicontext modal-primary" on:click={positive}>
      {positiveLabel}
    </button>
    {#if style === "yesno"}
      <button type="button" class="btnicontext" on:click={negative} disabled={negativeDisabled}>
        {negativeLabel}
      </button>
    {/if}
  </div>
</div>

<style lang="scss">
  .modal-optout {
    margin: 0.3rem 0 0.15rem;
  }
</style>
