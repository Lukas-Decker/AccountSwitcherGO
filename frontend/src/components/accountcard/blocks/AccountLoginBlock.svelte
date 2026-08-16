<script lang="ts" generics="TAccount">
  import type { CardBlockProps } from "../blockRegistry";

  export let block: CardBlockProps<TAccount>;

  $: login = block.adapter.accountLogin(block.acc);
  $: show = block.adapter.shouldShowAccountLogin?.(block.acc) === true && !!login;
</script>

{#if show}
  <p class="streamerCensor">{login}</p>
{/if}
