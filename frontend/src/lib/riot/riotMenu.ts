import type { MenuItemDef } from "../../stores/contextMenu";
import * as RiotService from "../../../bindings/account-switcher/internal/riotservice/service.js";

type Card = Awaited<ReturnType<typeof RiotService.GetCard>>;

/** Riot's own platform key, as the descriptor spells it. */
export const RIOT_PLATFORM_KEY = "Riot Games";

type Translate = (key: string, vars?: Record<string, string | number>) => string;

/**
 * How long ago a snapshot was taken, in the coarsest unit that is still honest.
 *
 * Coarse on purpose: the point is to stop a rank from three weeks ago reading as
 * current, not to report a duration to the minute.
 */
export function describeAge(iso: string, tr: Translate): string {
  const taken = Date.parse(iso);
  if (Number.isNaN(taken)) return "";
  const mins = Math.max(0, Math.round((Date.now() - taken) / 60000));
  if (mins < 60) return tr("Riot_Age_Minutes", { n: mins });
  const hours = Math.round(mins / 60);
  if (hours < 48) return tr("Riot_Age_Hours", { n: hours });
  return tr("Riot_Age_Days", { n: Math.round(hours / 24) });
}

export interface RiotMenuDeps {
  tr: Translate;
  openUrl: (url: string) => void;
  editLink: () => void;
  refresh: () => void;
}

/**
 * Builds the Riot submenu for one account.
 *
 * Everything is derived from a card the caller already fetched, so opening a
 * context menu never waits on a network call: an unreachable client or a spent
 * API quota costs a stale line of text, not a menu that hangs.
 */
export function buildRiotMenu(card: Card | null, deps: RiotMenuDeps): MenuItemDef {
  const { tr } = deps;
  const children: MenuItemDef[] = [];

  if (!card?.linked) {
    children.push({ label: tr("Riot_Link"), action: () => deps.editLink() });
    return { label: tr("Riot_CardTitle"), children };
  }

  // Identity and standings first, as unclickable rows: this is a summary, and a
  // row that looks like a button but does nothing is worse than one that plainly
  // is not one.
  children.push({ label: card.riotId, disabled: true });
  if (card.level > 0) {
    children.push({ label: tr("Riot_Level", { level: card.level }), disabled: true });
  }
  for (const rank of card.ranks ?? []) {
    children.push({ label: rank.display, disabled: true });
  }

  // Where the figures came from, so a snapshot is never mistaken for live data.
  if (card.source === "snapshot" && card.capturedAt) {
    children.push({ label: tr("Riot_AsOf", { age: describeAge(card.capturedAt, tr) }), disabled: true });
  } else if (card.linked && !card.hasKey && (card.ranks?.length ?? 0) === 0 && !card.capturedAt) {
    children.push({ label: tr("Riot_NoKeyHint"), disabled: true });
  }

  if (card.error) {
    children.push({ label: card.error, disabled: true });
  }

  children.push({ type: "separator" });

  // Links are grouped by title, since the same site appears under more than one.
  const byTitle = new Map<string, typeof card.links>();
  for (const link of card.links ?? []) {
    const list = byTitle.get(link.title) ?? [];
    list.push(link);
    byTitle.set(link.title, list);
  }
  for (const [title, links] of byTitle) {
    children.push({
      label: titleLabel(title),
      children: links.map((l) => ({ label: l.site, action: () => deps.openUrl(l.url) })),
    });
  }

  children.push({ type: "separator" });
  children.push({ label: tr("Riot_Refresh"), action: () => deps.refresh() });
  children.push({ label: tr("Riot_Edit"), action: () => deps.editLink() });

  return { label: tr("Riot_CardTitle"), children };
}

function titleLabel(title: string): string {
  switch (title) {
    case "tft":
      return "TFT";
    case "valorant":
      return "VALORANT";
    default:
      return "League";
  }
}
