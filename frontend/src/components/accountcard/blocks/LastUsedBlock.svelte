<script lang="ts" generics="TAccount">
  import { locale, t } from "../../../stores/i18n";
  import { formatLastLoginForLocale } from "../../../lib/formatLastLogin";
  import CardIcon from "../CardIcon.svelte";
  import { cardIcons } from "../cardIcons";
  import type { CardBlockProps } from "../blockRegistry";

  export let block: CardBlockProps<TAccount>;

  $: show = block.adapter.shouldShowLastUsed(block.acc);
  $: value = formatLastLoginForLocale(block.adapter.lastUsed(block.acc), $locale);
  $: label = $t("CardBlock_LastUsed");
</script>

{#if show}
  {#if block.display === "icon"}
    <span class="acc_lastused acc_lastused--icon" title={`${label}: ${value}`}>
      <CardIcon paths={cardIcons.lastUsed} label={`${label}: ${value}`} />
    </span>
  {:else if block.display === "iconText"}
    <span class="acc_lastused acc_lastused--icontext" title={`${label}: ${value}`}>
      <CardIcon paths={cardIcons.lastUsed} />
      <span>{value}</span>
    </span>
  {:else}
    <p class="acc_lastused">{value}</p>
  {/if}
{/if}
