import type { ComponentType } from "svelte";
import type { GameStatMetricDTO, PlatformAccountAdapter } from "../PlatformAccountAdapter";
import { CORE_BLOCK_KINDS, type CardBlockKind } from "./blockKinds";

import AvatarBlock from "./blocks/AvatarBlock.svelte";
import AccountLoginBlock from "./blocks/AccountLoginBlock.svelte";
import DisplayNameBlock from "./blocks/DisplayNameBlock.svelte";
import TagsBlock from "./blocks/TagsBlock.svelte";
import NoteBlock from "./blocks/NoteBlock.svelte";
import GameStatsBlock from "./blocks/GameStatsBlock.svelte";
import PlatformIdBlock from "./blocks/PlatformIdBlock.svelte";
import LastUsedBlock from "./blocks/LastUsedBlock.svelte";
import StatusLineBlock from "./blocks/StatusLineBlock.svelte";

/**
 * What every block is handed. Passed as one object rather than as separate
 * props so each block takes the same single prop whether or not it reads all
 * of it, which keeps the blocks interchangeable to the renderer.
 */
export interface CardBlockProps<TAccount> {
  acc: TAccount;
  adapter: PlatformAccountAdapter<TAccount>;
  rid: string;
  epoch: number;
  /** Ties the name block to the hidden radio's accessible name. */
  labelId: string;
  gameStats?: Record<string, Record<string, GameStatMetricDTO>>;
  /** Clamps small tooltips to the scrolling list. */
  boundary?: HTMLElement;
  /**
   * Clamps large hover surfaces, such as Steam's miniprofile. Deliberately the
   * page rather than the list: a popover the size of a profile card has to be
   * allowed outside the list's own bounds or it gets squashed.
   */
  hoverBoundary?: HTMLElement;
}

export interface CardBlockDef {
  kind: CardBlockKind;
  /** i18n key naming this block in the card editor. */
  labelKey: string;
  /**
   * Whether this block can put an identifier on screen. Streamer mode reads
   * this rather than trusting each block to remember to censor itself, so a
   * custom layout cannot surface an ID that streamer mode is meant to hide.
   */
  censorable: boolean;
  component: ComponentType;
}

const REGISTRY: Record<CardBlockKind, CardBlockDef> = {
  avatar: { kind: "avatar", labelKey: "CardBlock_Avatar", censorable: false, component: AvatarBlock as ComponentType },
  accountLogin: { kind: "accountLogin", labelKey: "CardBlock_AccountLogin", censorable: true, component: AccountLoginBlock as ComponentType },
  displayName: { kind: "displayName", labelKey: "CardBlock_DisplayName", censorable: false, component: DisplayNameBlock as ComponentType },
  tags: { kind: "tags", labelKey: "CardBlock_Tags", censorable: false, component: TagsBlock as ComponentType },
  note: { kind: "note", labelKey: "CardBlock_Note", censorable: false, component: NoteBlock as ComponentType },
  gameStats: { kind: "gameStats", labelKey: "CardBlock_GameStats", censorable: false, component: GameStatsBlock as ComponentType },
  platformId: { kind: "platformId", labelKey: "CardBlock_PlatformId", censorable: true, component: PlatformIdBlock as ComponentType },
  lastUsed: { kind: "lastUsed", labelKey: "CardBlock_LastUsed", censorable: false, component: LastUsedBlock as ComponentType },
  statusLine: { kind: "statusLine", labelKey: "CardBlock_StatusLine", censorable: false, component: StatusLineBlock as ComponentType },
};

export function blockDef(kind: CardBlockKind): CardBlockDef {
  return REGISTRY[kind];
}

/**
 * The blocks a platform can draw: the core set every platform has, plus
 * whatever its adapter declares.
 */
export function availableBlockKinds<T>(adapter: PlatformAccountAdapter<T>): CardBlockKind[] {
  const extra = adapter.cardBlocks ?? [];
  return [...CORE_BLOCK_KINDS, ...extra.filter((k) => !CORE_BLOCK_KINDS.includes(k))];
}

/**
 * Resolves the component for a block, letting a platform replace the avatar
 * with one that can carry more than an image.
 */
export function blockComponent<T>(kind: CardBlockKind, adapter: PlatformAccountAdapter<T>): ComponentType {
  if (kind === "avatar" && adapter.avatarComponent) return adapter.avatarComponent;
  return REGISTRY[kind].component;
}
