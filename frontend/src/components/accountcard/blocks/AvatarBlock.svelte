<script lang="ts" generics="TAccount">
  import { avatarSalt, streamerMode } from "../../../stores/streamerMode";
  import { offlineMode } from "../../../stores/offlineMode";
  import { accountAvatarSrc } from "../../../lib/accountAvatarSrc";
  import type { CardBlockProps } from "../blockRegistry";

  export let block: CardBlockProps<TAccount>;
</script>

<img
  src={accountAvatarSrc({
    streamer: $streamerMode,
    salt: $avatarSalt,
    platformKey: block.adapter.platformKey,
    accountKey: block.adapter.accountLogin(block.acc) || block.rid,
    imageUrl: block.adapter.imageUrl(block.acc),
    pending: block.adapter.imagePending(block.acc),
    epoch: block.epoch,
    offline: $offlineMode,
    fallback: block.adapter.profileFallback,
  })}
  alt=""
  draggable="false"
/>
