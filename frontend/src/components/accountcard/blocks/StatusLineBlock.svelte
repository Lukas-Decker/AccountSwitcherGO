<script lang="ts" generics="TAccount">
  import CardIcon from "../CardIcon.svelte";
  import { cardIcons } from "../cardIcons";
  import type { CardBlockProps } from "../blockRegistry";

  export let block: CardBlockProps<TAccount>;

  $: status = block.adapter.statusLine?.(block.acc) ?? null;
  $: paths = status?.kind === "error" ? cardIcons.statusError : cardIcons.statusPending;
</script>

{#if status}
  {#if block.display === "icon"}
    <span
      class="acc_status acc_status--{status.kind}"
      title={status.title ?? status.text}
    >
      <CardIcon {paths} label={status.text} />
    </span>
  {:else if block.display === "iconText"}
    <span
      class="acc_status acc_status--{status.kind}"
      title={status.title ?? status.text}
    >
      <CardIcon {paths} />
      <span class="acc_status__text">{status.text}</span>
    </span>
  {:else if status.kind === "error"}
    <div class="steam_meta_err" title={status.title ?? status.text}>{status.text}</div>
  {:else}
    <div class="steam_meta_pending">{status.text}</div>
  {/if}
{/if}
