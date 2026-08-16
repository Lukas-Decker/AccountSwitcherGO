<script lang="ts" generics="TAccount">
  import type { PlatformAccountAdapter } from "../../PlatformAccountAdapter";

  export let acc: TAccount;
  export let adapter: PlatformAccountAdapter<TAccount>;
</script>

{#if adapter.shouldShowNote(acc)}
  <p class="acc_note">{adapter.note(acc)}</p>
{/if}
