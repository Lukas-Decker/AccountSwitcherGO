<script lang="ts" generics="TAccount">
  import AccountTagBubbles from "../../AccountTagBubbles.svelte";
  import type { CardBlockProps } from "../blockRegistry";

  export let block: CardBlockProps<TAccount>;
</script>

<AccountTagBubbles tags={block.adapter.tags(block.acc) ?? []} />
