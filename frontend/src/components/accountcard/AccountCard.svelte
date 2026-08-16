<script lang="ts" generics="TAccount">
  import AccountLiveSessionIndicator from "../AccountLiveSessionIndicator.svelte";
  import AccountTagBubbles from "../AccountTagBubbles.svelte";
  import AvatarBlock from "./blocks/AvatarBlock.svelte";
  import DisplayNameBlock from "./blocks/DisplayNameBlock.svelte";
  import NoteBlock from "./blocks/NoteBlock.svelte";
  import GameStatsBlock from "./blocks/GameStatsBlock.svelte";
  import LastUsedBlock from "./blocks/LastUsedBlock.svelte";
  import { t } from "../../stores/i18n";
  import { contextMenu as ctxMenuAction } from "../../lib/actions/contextMenu";
  import type { MenuItemDef } from "../../stores/contextMenu";
  import type { GameStatMetricDTO, PlatformAccountAdapter } from "../PlatformAccountAdapter";

  export let acc: TAccount;
  export let adapter: PlatformAccountAdapter<TAccount>;
  export let rid: string;

  /** Ids shared with the hidden radio the card labels. */
  export let radioId: string;
  export let labelId: string;
  export let descId: string;
  export let a11yDescription = "";

  export let epoch = 0;
  export let gameStats: Record<string, Record<string, GameStatMetricDTO>> | undefined = undefined;

  /** Clamps the live-session tooltip to the scrolling list. */
  export let boundary: HTMLElement | undefined = undefined;

  export let profileDropActive = false;
  export let dropTarget = false;

  export let contextMenuItems: () => MenuItemDef[];
  export let onContextMenuOpen: () => void;
  export let onDragOver: (e: DragEvent) => void;
  export let onDragLeave: (e: DragEvent) => void;
  export let onActivate: () => void;

  $: isCurrent = adapter.currentSession(acc);
  $: isBroken = adapter.savedDataBroken?.(acc) === true;
</script>

<!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
<label
  for={radioId}
  class="acc"
  class:currentAcc={isCurrent}
  class:acc--broken={isBroken}
  class:acc--profile-drop-target={profileDropActive}
  class:acc--drop-target={dropTarget}
  on:dragover={onDragOver}
  on:dragleave={onDragLeave}
  use:ctxMenuAction={{ items: contextMenuItems, beforeOpen: onContextMenuOpen }}
  on:dblclick|preventDefault={onActivate}
>
  <AccountLiveSessionIndicator
    active={isCurrent}
    tooltipText={$t("Tooltip_CurrentAccount")}
    {boundary}
  />

  {#if isBroken}
    <span class="acc_broken_badge">{$t("Security_AccountDataBroken")}</span>
  {/if}

  {#if profileDropActive}
    <div class="acc_profile_drop_overlay" class:acc_profile_drop_overlay--hover={dropTarget} aria-hidden="true">
      <div class="acc_profile_drop_overlay__center">
        <div class="acc_profile_drop_overlay__icon" aria-hidden="true">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="currentColor" d="M5 20h14v-2H5v2zM19 9h-4V3H9v6H5l7 7 7-7z"/></svg>
        </div>
        <span class="acc_profile_drop_overlay__label">{$t("Drop_SetAccountIcon")}</span>
      </div>
    </div>
  {/if}

  <slot name="account-avatar">
    <AvatarBlock {acc} {adapter} {rid} {epoch} />
  </slot>

  <slot name="account-before-name" />

  <DisplayNameBlock {acc} {adapter} {labelId} />

  {#if a11yDescription}
    <span id={descId} class="sr-only">{a11yDescription}</span>
  {/if}

  <AccountTagBubbles tags={adapter.tags(acc) ?? []} />

  <NoteBlock {acc} {adapter} />

  <GameStatsBlock stats={gameStats} />

  <slot name="account-after-stats" />

  <LastUsedBlock {acc} {adapter} />

  <slot name="account-footer" />
</label>
