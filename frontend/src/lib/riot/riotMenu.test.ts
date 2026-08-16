import { describe, expect, it } from "vitest";

import { buildRiotMenu, formatBalance } from "./riotMenu";

type Card = Parameters<typeof buildRiotMenu>[0];

const tr = (key: string, vars?: Record<string, string | number>) =>
  vars ? `${key}(${Object.entries(vars).map(([k, v]) => `${k}=${v}`).join(",")})` : key;

const deps = { tr, locale: "en-US", openUrl: () => {}, editLink: () => {}, refresh: () => {} };

function card(overrides: Partial<NonNullable<Card>> = {}): Card {
  return {
    riotId: "Someone#EUW",
    region: "euw1",
    linked: true,
    hasKey: false,
    level: 30,
    iconId: 0,
    iconUrl: "",
    ranks: [],
    links: [],
    error: "",
    source: "client",
    capturedAt: "",
    hasWallet: false,
    blueEssence: 0,
    riotPoints: 0,
    walletAt: "",
    ...overrides,
  } as NonNullable<Card>;
}

function labels(c: Card): string[] {
  return (buildRiotMenu(c, deps).children ?? []).map((child) =>
    "label" in child && child.label ? child.label : "",
  );
}

it("shows a balance of zero, which is why hasWallet exists at all", () => {
  // The obvious version of this check is `if (card.blueEssence)`, which hides
  // the balance of an account that has spent everything.
  const rows = labels(card({ hasWallet: true, blueEssence: 0, riotPoints: 0 }));
  expect(rows).toContain("Riot_BlueEssence(amount=0)");
  expect(rows).toContain("Riot_RiotPoints(amount=0)");
});

it("dates the balances only when they are older than the rest of the card", () => {
  // The balances come from the client and the level can come from the API, so
  // they carry separate times; when both came from one snapshot, dating them
  // twice would put the same age on screen two rows apart.
  const at = new Date(Date.now() - 3 * 86400_000).toISOString();
  const wallet = { hasWallet: true, blueEssence: 10, riotPoints: 20, walletAt: at };

  const separate = labels(card({ ...wallet, source: "api" }));
  expect(separate.some((l) => l.startsWith("Riot_Wallet_AsOf"))).toBe(true);

  const together = labels(card({ ...wallet, source: "snapshot", capturedAt: at }));
  expect(together.some((l) => l.startsWith("Riot_Wallet_AsOf"))).toBe(false);
  expect(together.some((l) => l.startsWith("Riot_AsOf"))).toBe(true);
});

describe("formatBalance", () => {
  it("falls back to the bare number rather than showing nothing", () => {
    expect(formatBalance(27220, "not a locale")).toBe("27220");
  });
});
