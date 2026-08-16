<script lang="ts" generics="TAccount">
  import { avatarSalt, streamerMode } from "../../../stores/streamerMode";
  import { offlineMode } from "../../../stores/offlineMode";
  import { accountAvatarSrc } from "../../../lib/accountAvatarSrc";
  import type { PlatformAccountAdapter } from "../../PlatformAccountAdapter";

  export let acc: TAccount;
  export let adapter: PlatformAccountAdapter<TAccount>;
  export let rid: string;
  export let epoch = 0;
</script>

<img
  src={accountAvatarSrc({
    streamer: $streamerMode,
    salt: $avatarSalt,
    platformKey: adapter.platformKey,
    accountKey: adapter.accountLogin(acc) || rid,
    imageUrl: adapter.imageUrl(acc),
    pending: adapter.imagePending(acc),
    epoch,
    offline: $offlineMode,
    fallback: adapter.profileFallback,
  })}
  alt=""
  draggable="false"
/>
