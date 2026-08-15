<script lang="ts">
  /*
    One card in the settings grid: a titled box that holds setting rows.

    The card watches the rows inside it. When a search hides all of them the
    card hides too, and it reports that up to the grid so the grid can say
    "nothing found" instead of showing a page of gaps. Hiding is done with a
    class rather than an {#if}: the rows have to stay mounted, otherwise they
    would stop reporting matches and the card could never come back.
  */
  import { onDestroy } from "svelte";
  import { writable } from "svelte/store";
  import {
    createMatchRegistry,
    matchesSettingsQuery,
    nextMatchId,
    provideCardTitle,
    provideRowRegistry,
    useCardRegistry,
    useSettingsQuery,
  } from "../../../lib/settingsSearch";

  import type { SettingsIcon } from "./settingsIcons";

  /** Card heading, also searchable: matching it reveals every row inside. */
  export let title: string;
  /** Outline icon for the accent tile next to the title. */
  export let icon: SettingsIcon;
  /** Extra words that should find this card, e.g. product names. */
  export let keywords = "";

  const query = useSettingsQuery();
  const cardRegistry = useCardRegistry();
  const id = nextMatchId("settings-card");

  const titleStore = writable(title);
  provideCardTitle(titleStore);
  $: titleStore.set(`${title} ${keywords}`.trim());

  const rows = createMatchRegistry();
  provideRowRegistry(rows);

  const rowsMatched = rows.anyMatched;
  const rowsEmpty = rows.empty;

  // A card with no rows at all still answers for its own title.
  $: visible = $rowsEmpty ? matchesSettingsQuery(`${title} ${keywords}`, $query) : $rowsMatched;
  $: cardRegistry?.report(id, visible);

  onDestroy(() => cardRegistry?.remove(id));
</script>

<section class="settings-card" class:settings-card--hidden={!visible}>
  <header class="settings-card__head">
    <span class="settings-card__icon" aria-hidden="true">
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        {#each icon as d}<path {d} />{/each}
      </svg>
    </span>
    <h2 class="settings-card__title">{title}</h2>
  </header>
  <div class="settings-card__body">
    <slot />
  </div>
  <slot name="footer" />
</section>
