<script lang="ts" generics="TAccount">
  import { censoredName } from "../../../stores/streamerMode";
  import type { PlatformAccountAdapter } from "../../PlatformAccountAdapter";

  export let acc: TAccount;
  export let adapter: PlatformAccountAdapter<TAccount>;
  /** Ties the card to the hidden radio's accessible name. */
  export let labelId: string;
</script>

<h6 id={labelId} class="displayName">{$censoredName(adapter.name(acc))}</h6>
