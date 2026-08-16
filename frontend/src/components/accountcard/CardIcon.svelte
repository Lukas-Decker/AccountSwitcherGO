<script lang="ts">
  import type { CardIcon } from "./cardIcons";

  export let paths: CardIcon;
  /** Icon-only blocks still need a name; text modes label themselves. */
  export let label = "";
</script>

<svg
  class="acc_icon"
  viewBox="0 0 24 24"
  fill="none"
  stroke="currentColor"
  stroke-width="2"
  stroke-linecap="round"
  stroke-linejoin="round"
  role={label ? "img" : "presentation"}
  aria-label={label || undefined}
  aria-hidden={label ? undefined : "true"}
>
  {#each paths as d}
    <path {d} />
  {/each}
</svg>

<style>
  .acc_icon {
    width: 1.15em;
    height: 1.15em;
    flex: none;
    vertical-align: -0.2em;
  }
</style>
