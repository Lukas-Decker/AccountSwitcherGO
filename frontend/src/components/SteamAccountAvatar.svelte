<script lang="ts">
  import { offlineMode } from "../stores/offlineMode";
  import { avatarSalt, streamerMode } from "../stores/streamerMode";
  import { accountAvatarSrc } from "../lib/accountAvatarSrc";
  import { isProfileVideoUrl } from "../lib/profileImageDrop";
  import { miniProfileHover } from "../lib/actions/miniProfileHover";
  import type { SteamAccountRow } from "../lib/steam/types";
  import type { CardBlockProps } from "./accountcard/blockRegistry";

  /** Steam's avatar is a card block: it carries frames, video and a hover card. */
  export let block: CardBlockProps<SteamAccountRow>;

  $: account = block.acc;

  function steamListAvatarUrl(): string | undefined {
    const primary = account.imageUrl?.trim() || undefined;
    const fb = account.staticImageUrl?.trim() || undefined;
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
    epoch: block.epoch,
    offline: $offlineMode,
    fallback: block.adapter.profileFallback,
  });
  $: avatarIsVideo = !$offlineMode && !$streamerMode && isProfileVideoUrl(avatarSrc);
  // The hover card is a slab of the account's Steam profile - persona name, level,
  // games. Exactly what streamer mode exists to keep off the screen.
  $: miniProfileEnabled =
    !$streamerMode && !!(account.showMiniProfile && (account.miniProfileHtml ?? "").trim() !== "");
  $: hasFrame =
    account.showAvatarFrame === true &&
    (account.avatarFrameUrl ?? "").trim() !== "" &&
    !$offlineMode &&
    !$streamerMode;
  $: hoverOpts = {
    html: account.miniProfileHtml ?? "",
    boundary: block.hoverBoundary ?? block.boundary ?? null,
    offline: $offlineMode,
    enabled: miniProfileEnabled,
  };
</script>

<span class="steam-acc-avatar-wrap">
  {#if avatarIsVideo}
    <video
      class="steam-acc-avatar"
      class:steam-acc-avatar--framed={hasFrame}
      class:status_vac={account.vac}
      class:status_limited={account.ltd}
      src={avatarSrc}
      autoplay loop muted playsinline
      aria-hidden="true" draggable="false"
      use:miniProfileHover={hoverOpts}
    ></video>
  {:else}
    <img
      class="steam-acc-avatar"
      class:steam-acc-avatar--framed={hasFrame}
      class:status_vac={account.vac}
      class:status_limited={account.ltd}
      src={avatarSrc}
      alt="" draggable="false"
      use:miniProfileHover={hoverOpts}
    />
  {/if}
  <!-- The frame layer is always present, empty when this account has none.
       Only some Steam accounts have a frame, and letting the layer come and go
       meant the avatar sat in a different sized box depending on whose card it
       was, so no amount of padding lined the two up. -->
  {#if hasFrame}
    <img class="steam-acc-avatar-frame" src={account.avatarFrameUrl ?? ""} alt="" draggable="false" />
  {:else}
    <span class="steam-acc-avatar-frame steam-acc-avatar-frame--empty" aria-hidden="true"></span>
  {/if}
</span>
