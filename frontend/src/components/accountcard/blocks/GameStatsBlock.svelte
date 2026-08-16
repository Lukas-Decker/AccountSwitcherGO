<script lang="ts" generics="TAccount">
  import { t } from "../../../stores/i18n";
  import { sanitizeHtml } from "../../../lib/sanitizeHtml";
  import type { CardBlockProps } from "../blockRegistry";

  export let block: CardBlockProps<TAccount>;

  $: stats = block.gameStats;
  $: hasStats = !!stats && Object.keys(stats).length > 0;
</script>

{#if hasStats && stats}
  <div class="acc_inline_gamestats" aria-label={$t("Context_ManageGameStats")}>
    <span class="acc_inline_gamestats_metrics">
      {#each Object.values(stats) as metrics}
        {#each Object.values(metrics) as dto}
          <span class="acc_inline_gamestats_metric">
            {#if dto.indicatorMarkup}
              <span class="acc_inline_gamestats_ind">{@html sanitizeHtml(dto.indicatorMarkup, "gameStats")}</span>
            {/if}
            <span class="acc_inline_gamestats_val">{@html sanitizeHtml(dto.statValue, "gameStats")}</span>
          </span>
        {/each}
      {/each}
    </span>
  </div>
{/if}
