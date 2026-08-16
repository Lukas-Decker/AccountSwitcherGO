<script lang="ts" generics="TAccount">
  import { locale } from "../../../stores/i18n";
  import { formatLastLoginForLocale } from "../../../lib/formatLastLogin";
  import type { PlatformAccountAdapter } from "../../PlatformAccountAdapter";

  export let acc: TAccount;
  export let adapter: PlatformAccountAdapter<TAccount>;
</script>

{#if adapter.shouldShowLastUsed(acc)}
  <p class="acc_lastused">{formatLastLoginForLocale(adapter.lastUsed(acc), $locale)}</p>
{/if}
