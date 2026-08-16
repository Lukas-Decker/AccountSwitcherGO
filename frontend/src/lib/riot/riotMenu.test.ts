import { describe, expect, it } from "vitest";

import { buildRiotMenu, formatBalance } from "./riotMenu";

type Card = Parameters<typeof buildRiotMenu>[0];

const tr = (key: string, vars?: Record<string, string | number>) =>
  vars ? `${key}(${Object.entries(vars).map(([k, v]) => `${k}=${v}`).join(",")})` : key;

const deps = {
  tr,
  locale: "en-US",
  openUrl: () => {},
  editLink: () => {},
  refresh: () => {},
};

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

describe("formatBalance", () => {
  it("groups the digits, because five and six figures look alike without it", () => {
    expect(formatBalance(27220, "en-US")).toBe("27,220");
    expect(formatBalance(272200, "en-US")).toBe("272,200");
  });

  it("groups them the way the reader's locale does", () => {
    expect(formatBalance(27220, "de-DE")).toBe("27.220");
  });

  it("survives an unusable locale tag rather than showing nothing", () => {
    expect(formatBalance(27220, "not a locale")).toBe("27220");
  });
});

describe("buildRiotMenu wallet rows", () => {
  it("leaves both rows out when no balance has ever been read", () => {
    const rows = labels(card());
    expect(rows.some((l) => l.startsWith("Riot_BlueEssence"))).toBe(false);
    expect(rows.some((l) => l.startsWith("Riot_RiotPoints"))).toBe(false);
  });

  it("shows both balances when the client has reported them", () => {
    const rows = labels(card({ hasWallet: true, blueEssence: 27220, riotPoints: 22670 }));
    expect(rows).toContain("Riot_BlueEssence(amount=27,220)");
    expect(rows).toContain("Riot_RiotPoints(amount=22,670)");
  });

  it("shows a balance of zero, which is a real balance and not a missing one", () => {
    const rows = labels(card({ hasWallet: true, blueEssence: 0, riotPoints: 0 }));
    expect(rows).toContain("Riot_BlueEssence(amount=0)");
    expect(rows).toContain("Riot_RiotPoints(amount=0)");
  });

  it("dates the balances when they are older than the rest of the card", () => {
    const rows = labels(
      card({
        hasWallet: true,
        blueEssence: 10,
        riotPoints: 20,
        walletAt: new Date(Date.now() - 3 * 86400_000).toISOString(),
        source: "api",
      }),
    );
    expect(rows.some((l) => l.startsWith("Riot_Wallet_AsOf"))).toBe(true);
  });

  it("does not date them twice when the whole card is one snapshot", () => {
    const at = new Date(Date.now() - 3 * 86400_000).toISOString();
    const rows = labels(
      card({ hasWallet: true, blueEssence: 10, riotPoints: 20, walletAt: at, source: "snapshot", capturedAt: at }),
    );
    expect(rows.filter((l) => l.startsWith("Riot_Wallet_AsOf"))).toHaveLength(0);
    expect(rows.some((l) => l.startsWith("Riot_AsOf"))).toBe(true);
  });

  it("says nothing about a wallet for an account that is not linked", () => {
    const rows = labels(card({ linked: false, hasWallet: true, blueEssence: 5, riotPoints: 5 }));
    expect(rows.some((l) => l.startsWith("Riot_BlueEssence"))).toBe(false);
  });
});
