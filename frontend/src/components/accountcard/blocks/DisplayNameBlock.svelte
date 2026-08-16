<script lang="ts" generics="TAccount">
  import { censoredName } from "../../../stores/streamerMode";
  import type { CardBlockProps } from "../blockRegistry";

  export let block: CardBlockProps<TAccount>;
</script>

<h6 id={block.labelId} class="displayName">{$censoredName(block.adapter.name(block.acc))}</h6>
