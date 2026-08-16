<script lang="ts" generics="TAccount">
  import { locale } from "../../../stores/i18n";
  import { formatLastLoginForLocale } from "../../../lib/formatLastLogin";
  import type { CardBlockProps } from "../blockRegistry";

  export let block: CardBlockProps<TAccount>;
</script>

{#if block.adapter.shouldShowLastUsed(block.acc)}
  <p class="acc_lastused">{formatLastLoginForLocale(block.adapter.lastUsed(block.acc), $locale)}</p>
{/if}
