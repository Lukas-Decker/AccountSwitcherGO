<script lang="ts" generics="TAccount">
  import type { CardBlockProps } from "../blockRegistry";

  export let block: CardBlockProps<TAccount>;

  $: status = block.adapter.statusLine?.(block.acc) ?? null;
</script>

{#if status}
  {#if status.kind === "error"}
    <div class="steam_meta_err" title={status.title ?? status.text}>{status.text}</div>
  {:else}
    <div class="steam_meta_pending">{status.text}</div>
  {/if}
{/if}
