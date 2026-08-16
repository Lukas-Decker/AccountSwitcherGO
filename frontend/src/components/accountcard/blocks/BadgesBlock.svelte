<script lang="ts" generics="TAccount">
  import { t } from "../../../stores/i18n";
  import type { CardBlockProps } from "../blockRegistry";

  export let block: CardBlockProps<TAccount>;

  $: badges = block.adapter.badges?.(block.acc) ?? [];
</script>

{#if badges.length > 0}
  <span class="acc_badges">
    {#each badges as badge (badge.id)}
      <span class="acc_badge acc_badge--{badge.tone}">{$t(badge.labelKey)}</span>
    {/each}
  </span>
{/if}

<style lang="scss">
  .acc_badges {
    display: inline-flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 0.15rem 0.1875rem;
    max-width: 100%;
    margin: 0.1125rem 0 0;
  }

  .acc_badge {
    padding: 0.02em 0.3em;
    border: 1px solid currentColor;
    border-radius: 2px;
    font-size: 0.62em;
    line-height: 1.4;
    letter-spacing: 0.02em;
    text-transform: uppercase;
    white-space: nowrap;
  }

  // The same two colours the avatar border uses for these flags, so the
  // presentation changes but what a colour means does not.
  .acc_badge--danger {
    color: var(--red);
  }

  .acc_badge--warning {
    color: var(--preview-control-border, var(--form-checkbox-border, var(--yellow)));
  }
</style>
