<script lang="ts" generics="TAccount">
  import type { CardBlockProps } from "../blockRegistry";

  export let block: CardBlockProps<TAccount>;

  $: value = block.adapter.platformId?.(block.acc) ?? "";
  $: show = block.adapter.shouldShowPlatformId?.(block.acc) === true && !!value;
</script>

{#if show}
  <p class="streamerCensor steamId">{value}</p>
{/if}
