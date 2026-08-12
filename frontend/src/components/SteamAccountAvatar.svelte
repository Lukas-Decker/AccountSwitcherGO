<script lang="ts">
  import { get } from "svelte/store";
  import { offlineMode } from "../stores/offlineMode";
  import { avatarSalt, streamerMode } from "../stores/streamerMode";
  import { accountAvatarSrc } from "../lib/accountAvatarSrc";
  import { isProfileVideoUrl } from "../lib/profileImageDrop";
  import { miniProfileHover } from "../lib/actions/miniProfileHover";
  import type { SteamAccountRow } from "../lib/steam/types";

  export let account: SteamAccountRow;
  export let epoch = 0;
  export let fallback = "";
  export let boundary: HTMLElement | null = null;

  function steamListAvatarUrl(): string | undefined {
    const acc = account;
    const primary = acc.imageUrl?.trim() || undefined;
    const fb = acc.staticImageUrl?.trim() || undefined;
    if ($offlineMode) {
      if (fb) return fb;
      if (primary && !isProfileVideoUrl(primary)) return primary;
      return undefined;
    }
    return primary ?? fb;
  }

  $: avatarSrc = accountAvatarSrc({
    streamer: $streamerMode,
    salt: $avatarSalt,
    platformKey: "Steam",
    accountKey: account.accountName || account.steamId64 || "",
    imageUrl: steamListAvatarUrl(),
    pending: account.avatarPending === true,
    epoch,
    offline: $offlineMode,
    fallback,
  });
  $: avatarIsVideo = !$offlineMode && !$streamerMode && isProfileVideoUrl(avatarSrc);
  // The hover card is a slab of the account's Steam profile - persona name, level,
  // games. Exactly what streamer mode exists to keep off the screen.
  $: miniProfileEnabled =
    !$streamerMode && !!(account.showMiniProfile && (account.miniProfileHtml ?? "").trim() !== "");
</script>

<span class="steam-acc-avatar-wrap">
  {#if avatarIsVideo}
    <video
      class="steam-acc-avatar"
      class:status_vac={account.showVac && account.vac}
      class:status_limited={account.showLimited && account.ltd}
      src={avatarSrc}
      autoplay loop muted playsinline
      aria-hidden="true" draggable="false"
      use:miniProfileHover={{
        html: account.miniProfileHtml ?? "",
        boundary,
        offline: $offlineMode,
        enabled: miniProfileEnabled,
      }}
    ></video>
  {:else}
    <img
      class="steam-acc-avatar"
      class:status_vac={account.showVac && account.vac}
      class:status_limited={account.showLimited && account.ltd}
      src={avatarSrc}
      alt="" draggable="false"
      use:miniProfileHover={{
        html: account.miniProfileHtml ?? "",
        boundary,
        offline: $offlineMode,
        enabled: miniProfileEnabled,
      }}
    />
  {/if}
  {#if account.showAvatarFrame && (account.avatarFrameUrl ?? "").trim() !== "" && !$offlineMode && !$streamerMode}
    <img class="steam-acc-avatar-frame" src={account.avatarFrameUrl ?? ""} alt="" draggable="false" />
  {/if}
</span>
