<script lang="ts">
  /*
    One setting: its name, an optional explanation, and the control that
    changes it.

    Wide controls (sliders, text fields, button groups) pass stacked so the
    control drops onto its own line instead of squeezing the label.
  */
  import { onDestroy } from "svelte";
  import {
    matchesSettingsQuery,
    nextMatchId,
    useCardTitle,
    useRowRegistry,
    useSettingsQuery,
  } from "../../../lib/settingsSearch";

  export let label: string;
  /** Shown under the label, and searched along with it. */
  export let hint = "";
  /** Id of the control this row labels, so the text is clickable. */
  export let controlId = "";
  /** Puts the control on its own line under the text. */
  export let stacked = false;
  /** Dims the row when the setting cannot be changed right now. */
  export let disabled = false;
  /** Extra search terms that are not in the visible text. */
  export let keywords = "";

  const query = useSettingsQuery();
  const cardTitle = useCardTitle();
  const registry = useRowRegistry();
  const id = nextMatchId("setting-row");

  $: matched = matchesSettingsQuery(`${$cardTitle} ${label} ${hint} ${keywords}`, $query);
  $: registry?.report(id, matched);

  onDestroy(() => registry?.remove(id));
</script>

<div
  class="setting-row"
  class:setting-row--stacked={stacked}
  class:setting-row--disabled={disabled}
  class:setting-row--hidden={!matched}
>
  <div class="setting-row__text">
    {#if controlId}
      <label class="setting-row__label" for={controlId}>{label}</label>
    {:else}
      <span class="setting-row__label">{label}</span>
    {/if}
    {#if hint}
      <p class="setting-row__hint">{hint}</p>
    {/if}
    <slot name="under" />
  </div>
  <div class="setting-row__control">
    <slot />
  </div>
</div>
