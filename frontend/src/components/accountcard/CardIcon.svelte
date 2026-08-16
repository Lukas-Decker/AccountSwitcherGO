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
    width: 1.05em;
    height: 1.05em;
    flex: none;
    /* Digits carry their visual weight above the line box's centre, so an icon
       centred on the box reads as sitting low next to them. Nudged up to match
       what the eye lines up with rather than what the box says. */
    margin-top: -0.08em;
  }
</style>
