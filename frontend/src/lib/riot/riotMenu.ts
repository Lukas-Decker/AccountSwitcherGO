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

/**
 * Formats a currency balance with the reader's own grouping.
 *
 * Balances run to five and six figures, where 27220 and 272200 are hard to tell
 * apart at a glance; the separator is what makes them readable, and which
 * separator that is depends on the locale.
 */
export function formatBalance(amount: number, locale: string): string {
  if (!Number.isFinite(amount)) return "";
  try {
    return new Intl.NumberFormat(locale.replace(/_/g, "-")).format(amount);
  } catch {
    // An unusable locale tag is not a reason to show nothing.
    return String(amount);
  }
}

export interface RiotMenuDeps {
  tr: Translate;
  /** Current interface locale, used to group the digits of a balance. */
  locale: string;
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

  // Not loaded and not linked are different answers, and saying the second when
  // the first is true tells the user their account is unlinked when it is not.
  // A menu is built synchronously the instant it opens, so "not yet" is a real
  // state rather than a transient that can be waited out.
  if (card === null) {
    children.push({ label: tr("Riot_Loading"), disabled: true });
    children.push({ label: tr("Riot_Edit"), action: () => deps.editLink() });
    return { label: tr("Riot_CardTitle"), children };
  }

  if (!card.linked) {
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

  const walletAge = card.walletAt ? describeAge(card.walletAt, tr) : "";
  const snapshotAge =
    card.source === "snapshot" && card.capturedAt ? describeAge(card.capturedAt, tr) : "";

  // Balances, which only the League Client can report. They are shown whenever
  // any were ever read, since a stored balance is the only one there will be
  // while the client is closed. hasWallet rather than a truthiness check on the
  // amounts: an account really can hold zero of either.
  if (card.hasWallet) {
    children.push({
      label: tr("Riot_BlueEssence", { amount: formatBalance(card.blueEssence, deps.locale) }),
      disabled: true,
    });
    children.push({
      label: tr("Riot_RiotPoints", { amount: formatBalance(card.riotPoints, deps.locale) }),
      disabled: true,
    });
    // Balances have their own age, because they have their own source: the level
    // beside them can be a minute old from the API while these are a week old
    // from the last time the client was open. Said only when it differs from the
    // card's own age below, or the same figure is dated twice.
    if (walletAge && walletAge !== snapshotAge) {
      children.push({ label: tr("Riot_Wallet_AsOf", { age: walletAge }), disabled: true });
    }
  }

  // Where the figures came from, so a snapshot is never mistaken for live data.
  if (snapshotAge) {
    children.push({ label: tr("Riot_AsOf", { age: snapshotAge }), disabled: true });
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
