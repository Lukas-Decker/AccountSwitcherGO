/**
 * A stand-in for the Go backend, for driving the UI in a plain browser.
 *
 * Wails talks to Go over native IPC, so `vite dev` on its own boots the shell
 * and then every call fails: nothing renders and no screen can be checked.
 * This replaces the runtime's transport with one that answers from canned
 * data, which is enough to exercise every screen without a backend: home,
 * the account lists, the card editor, app, platform and Steam settings,
 * the games tab and advanced clearing. Saves are kept in memory so toggles
 * and edits round-trip within a session.
 *
 * Dev only, and opt-in: it is loaded behind `import.meta.env.DEV` and a
 * `?wailsStub` flag, so `wails3 dev` against the real backend is untouched.
 */

import { setTransport, objectNames } from "@wailsio/runtime";

/** Bound-method metadata, read out of the generated bindings at dev time. */
interface BindingInfo {
  name: string;
  returnType: string;
}

const CALL_BINDING = 0;

function buildBindingIndex(): Map<number, BindingInfo> {
  const index = new Map<number, BindingInfo>();
  // The generated bindings are the only place method IDs and their return
  // types are written down, so read them rather than restating them here.
  const sources = import.meta.glob("../../bindings/**/*.ts", {
    query: "?raw",
    import: "default",
    eager: true,
  }) as Record<string, string>;

  const re =
    /export function (\w+)\s*\([^)]*\)\s*:\s*\$CancellablePromise<([\s\S]*?)>\s*\{\s*return \$Call\.ByID\((\d+)/g;

  for (const source of Object.values(sources)) {
    let m: RegExpExecArray | null;
    while ((m = re.exec(source)) !== null) {
      index.set(Number(m[3]), { name: m[1], returnType: m[2].trim() });
    }
  }
  return index;
}

/**
 * A value shaped like the declared return type. Callers immediately hand the
 * result to a generated `$$createType`, which will throw on the wrong shape,
 * so guessing by type is what keeps unstubbed methods from breaking the page.
 */
function defaultForType(returnType: string): unknown {
  const t = returnType.trim();
  if (t === "void" || t === "undefined") return undefined;
  if (t.endsWith("[]") || t.startsWith("Array<")) return [];
  if (t.startsWith("{")) return {};
  if (t === "string") return "";
  if (t === "boolean") return false;
  if (t === "number") return 0;
  // Anything left is a generated model. The matching `$$createType` reads
  // fields off it with `in`, so an empty object survives where null throws.
  return {};
}

function nowIso(offsetMinutes: number): string {
  return new Date(Date.now() - offsetMinutes * 60_000).toISOString();
}

/** A small cast of accounts, chosen to exercise the card's edge cases. */
const BASIC_ACCOUNTS = [
  { uniqueId: "acc-1", displayName: "kestrel_9", note: "main / ranked", lastUsed: nowIso(120), currentSession: true },
  { uniqueId: "acc-2", displayName: "a_very_long_account_name_here", note: "", lastUsed: nowIso(1500), currentSession: false },
  { uniqueId: "acc-3", displayName: "smurf", note: "duo queue only", lastUsed: nowIso(8000), currentSession: false },
  { uniqueId: "acc-4", displayName: "trading", note: "", lastUsed: "", currentSession: false },
];

const STEAM_ACCOUNTS = [
  { steamId64: "76561198000000001", personaName: "kestrel_9", displayName: "kestrel_9", accountName: "kestrel_login", currentSession: true },
  { steamId64: "76561198000000002", personaName: "long name here", displayName: "a_very_long_account_name_here", accountName: "alt_login_02", currentSession: false },
  { steamId64: "76561198000000003", personaName: "smurf", displayName: "smurf", accountName: "smurf_login", currentSession: false },
];

/**
 * Stands in for a Steam avatar frame: an ornate ring with a transparent middle,
 * drawn to the edge of its box the way the real ones are. The card scales these
 * up by 1.22, so anything that reaches its own edge reaches past the avatar.
 */
const AVATAR_FRAME = `data:image/svg+xml;utf8,${encodeURIComponent(
  `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
     <circle cx="50" cy="50" r="47" fill="none" stroke="#e8c96a" stroke-width="5"/>
     <circle cx="50" cy="50" r="41" fill="none" stroke="#8a6d2b" stroke-width="2"/>
     <circle cx="50" cy="4" r="4" fill="#e8c96a"/>
     <circle cx="50" cy="96" r="4" fill="#e8c96a"/>
     <circle cx="4" cy="50" r="4" fill="#e8c96a"/>
     <circle cx="96" cy="50" r="4" fill="#e8c96a"/>
   </svg>`,
)}`;

const TAGS = [
  { id: "t1", name: "main", color: "#8aff80" },
  { id: "t2", name: "alt", color: "#80ffea" },
];

/**
 * Names double as icon asset lookups (public/img/platform/<name>.svg), so
 * they have to match files that actually exist. Seven enabled tiles leave
 * the home grid's last row partly filled, which is the wrap case worth
 * seeing; the disabled list feeds Manage Platforms' other column.
 */
const HOME_PLATFORMS = ["Steam", "BattleNet", "Epic Games", "Riot Games", "Discord", "Ubisoft", "GOG Galaxy"];
const DISABLED_PLATFORMS = ["Rockstar", "Origin", "EA Desktop"];

/** Mutable so Save/Set calls round-trip while the stub session lasts. */
const state = {
  homeOrder: [...HOME_PLATFORMS],
  disabled: [...DISABLED_PLATFORMS],
  cardConfig: { version: 1, preset: "medium", blocks: { note: false } } as Record<string, unknown>,
  platformSettings: new Map<string, Record<string, unknown>>(),
  steamSettings: null as Record<string, unknown> | null,
};

function defaultPlatformSettings(): Record<string, unknown> {
  return {
    RunAsAdmin: false,
    TrayAccNumber: 3,
    ForgetAccountEnabled: true,
    ClosingMethod: "Combined",
    CloseTimeoutSeconds: 0,
    ForceCloseAfterTimeout: true,
    StartingMethod: "Default",
    AutoStart: true,
    ShowShortNotes: true,
    ShowLastUsed: true,
    AccountNotes: {},
    LaunchArguments: "",
    PullAccountImagesOnSwitch: true,
    AccountCardCustomizationEnabled: false,
    AccountCard: null,
  };
}

function defaultSteamSettings(): Record<string, unknown> {
  return {
    ...defaultPlatformSettings(),
    FolderPath: "C:\\Program Files (x86)\\Steam",
    Steam_ShowSteamID: true,
    Steam_ShowVAC: true,
    Steam_ShowLimited: true,
    Steam_ShowLastLogin: true,
    Steam_ShowAccUsername: true,
    Steam_TrayAccountName: false,
    Steam_ImageExpiryTime: 7,
    Steam_OverrideState: -1,
    SteamWebApiKey: "",
    ShowSteamSwitcher: true,
    CollectInfo: true,
    Steam_ShowMiniProfile: false,
    Steam_ShowAvatarFrame: true,
  };
}

const OWNED_GAMES = [
  { appId: "730", name: "Counter-Strike 2", iconUrl: "", owners: ["76561198000000001", "76561198000000003"], installed: true },
  { appId: "570", name: "Dota 2", iconUrl: "", owners: ["76561198000000001"], installed: true },
  { appId: "271590", name: "Grand Theft Auto V", iconUrl: "", owners: ["76561198000000002"], installed: false },
  { appId: "1245620", name: "Elden Ring", iconUrl: "", owners: ["76561198000000002", "76561198000000003"], installed: false },
];

/**
 * Resolved games as the library service returns them, covering each case the
 * view has to render: an installed game whose installer is known, one shared by
 * several accounts with no installer named, and one owned but not installed.
 *
 * artUrl is a flat-colour 2:3 PNG inlined as a data URI. The real value points
 * into wwwroot, which no dev server serves, so without this every tile in the
 * browser preview falls back to its name and the artwork layout cannot be
 * checked at all.
 */
const RESOLVED_GAMES = [
  {
    platformKey: "Steam",
    gameId: "730",
    name: "Counter-Strike 2",
    artUrl: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACgAAAA8CAIAAACb22+3AAAAOElEQVR4nO3NQQkAAAgEsMtmNxsbwBgiDPZfqudExGKxWCwWi8VisVgsFovFYrFYLBaLxWLxp3gBFls6ERbNP9MAAAAASUVORK5CYII=",
    installed: true,
    installPath: "D:\SteamLibrary\steamapps\common\Counter-Strike Global Offensive",
    sizeOnDisk: 35000000000,
    owners: [
      {
        accountId: "76561198000000001",
        accountName: "Main",
        source: "steam:appmanifest",
        confidence: "exact",
        installedBy: true,
        playtimeMinutes: 74070,
        lastPlayed: "2026-08-20T18:00:00Z",
      },
      {
        accountId: "76561198000000003",
        accountName: "Smurf",
        source: "steam:localconfig",
        confidence: "strong",
        installedBy: false,
        playtimeMinutes: 640,
        lastPlayed: "2026-05-02T12:00:00Z",
      },
    ],
    sources: ["steam:appmanifest", "steam:localconfig"],
    artOptions: [
      { url: "", tier: "portrait", source: "steam:appmanifest" },
      { url: "", tier: "wide", source: "steam:applist" },
    ],
  },
  {
    platformKey: "Steam",
    gameId: "271590",
    name: "Grand Theft Auto V",
    artUrl: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACgAAAA8CAIAAACb22+3AAAAOUlEQVR4nO3NQQkAAAgEsGtlAfsXsIwxRBjsv0zXiYjFYrFYLBaLxWKxWCwWi8VisVgsFovF4k/xAiY0hvXIzg33AAAAAElFTkSuQmCC",
    installed: true,
    installPath: "C:\Program Files (x86)\Steam\steamapps\common\GTAV",
    sizeOnDisk: 95000000000,
    owners: [
      {
        accountId: "76561198000000002",
        accountName: "Alt",
        source: "steam:sharedconfig",
        confidence: "strong",
        installedBy: false,
        playtimeMinutes: 120,
        lastPlayed: "",
      },
      {
        accountId: "76561198000000003",
        accountName: "Smurf",
        source: "steam:userdata",
        confidence: "weak",
        installedBy: false,
        playtimeMinutes: 0,
        lastPlayed: "",
      },
    ],
    sources: ["steam:appmanifest", "steam:sharedconfig", "steam:userdata"],
  },
  {
    platformKey: "Steam",
    gameId: "1245620",
    name: "Elden Ring",
    artUrl: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACgAAAA8CAIAAACb22+3AAAAOElEQVR4nO3NMQ0AAAgDsPnXgzQE4IFnT5P+zU4qOqtYLBaLxWKxWCwWi8VisVgsFovFYrFYLH46NbqH4tyzr5oAAAAASUVORK5CYII=",
    installed: false,
    installPath: "",
    sizeOnDisk: 0,
    owners: [
      {
        accountId: "76561198000000002",
        accountName: "Alt",
        source: "steam:community-xml",
        confidence: "exact",
        installedBy: false,
        playtimeMinutes: 3300,
        lastPlayed: "",
      },
    ],
    sources: ["steam:community-xml"],
  },
  {
    platformKey: "Steam",
    gameId: "570",
    name: "Dota 2",
    artUrl: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACgAAAA8CAIAAACb22+3AAAAOElEQVR4nO3NQQkAAAgEsGtgPcvZ1RgiDPZfavpExGKxWCwWi8VisVgsFovFYrFYLBaLxWLxp3gBVnSGCKcFFisAAAAASUVORK5CYII=",
    installed: false,
    installPath: "",
    sizeOnDisk: 0,
    owners: [],
    sources: ["steam:userdata"],
  },
  {
    platformKey: "Steam",
    gameId: "9001",
    name: "Hentai Puzzle Deluxe",
    artUrl: "",
    installed: false,
    installPath: "",
    sizeOnDisk: 0,
    owners: [
      { accountId: "76561198000000002", accountName: "Alt", source: "steam:sharedconfig", confidence: "strong", installedBy: false, playtimeMinutes: 0, lastPlayed: "" },
    ],
    sources: ["steam:sharedconfig"],
    nsfw: true,
  },
  {
    platformKey: "Steam",
    gameId: "9002",
    name: "Something You Hid",
    artUrl: "",
    installed: false,
    installPath: "",
    sizeOnDisk: 0,
    owners: [
      { accountId: "76561198000000001", accountName: "Main", source: "steam:localconfig", confidence: "strong", installedBy: false, playtimeMinutes: 10, lastPlayed: "" },
    ],
    sources: ["steam:localconfig"],
    hidden: true,
  },
];

const SHORTCUTS = [
  { fileName: "Counter-Strike 2.url", displayName: "Counter-Strike 2", iconUrl: "", pinned: true, isPlatformExe: false, isUrl: true },
  { fileName: "Dota 2.url", displayName: "Dota 2", iconUrl: "", pinned: false, isPlatformExe: false, isUrl: true },
  { fileName: "Steam.exe", displayName: "Steam", iconUrl: "", pinned: false, isPlatformExe: true, isUrl: false },
];

/**
 * Canned results, keyed by the binding's function name. Anything not listed
 * falls back to a value shaped like its declared return type.
 */
const CANNED: Record<string, (args: unknown[]) => unknown> = {
  GetAccountsList: () =>
    BASIC_ACCOUNTS.map((a) => ({
      platformKey: "Steam",
      uniqueId: a.uniqueId,
      displayName: a.displayName,
      currentSession: a.currentSession,
      savedDataBroken: a.uniqueId === "acc-4",
    })),

  GetAccountsEnrichment: () =>
    BASIC_ACCOUNTS.map((a, i) => ({
      uniqueId: a.uniqueId,
      imageUrl: "",
      avatarPending: false,
      manualProfileImage: false,
      note: a.note,
      lastUsed: a.lastUsed,
      showLastUsed: true,
      savedDataBroken: a.uniqueId === "acc-4",
      tags: i < 2 ? [TAGS[i]] : [],
    })),

  GetSteamAccountsList: () => STEAM_ACCOUNTS,

  GetSteamAccountsEnrichment: () =>
    STEAM_ACCOUNTS.map((a, i) => ({
      steamId64: a.steamId64,
      displayName: a.displayName,
      lastLogin: nowIso(60 * (i + 1)),
      offline: false,
      imageUrl: "",
      staticImageUrl: "",
      avatarPending: false,
      metaPending: false,
      vac: i === 2,
      ltd: i === 1,
      showSteamId: true,
      showVac: true,
      showLimited: true,
      showLastLogin: true,
      showAccUsername: true,
      collectInfo: true,
      showShortNotes: true,
      note: i === 0 ? "main / ranked" : "",
      // The first account is the signed-in one, so this is the case the user
      // actually sees: a frame and the current-account ring on the same card.
      avatarFrameUrl: i === 0 ? AVATAR_FRAME : "",
      miniProfileHtml: "",
      showMiniProfile: false,
      showAvatarFrame: i === 0,
      syncError: i === 2 ? "Could not reach the Steam Web API" : "",
      tags: i < 2 ? [TAGS[i]] : [],
      manualProfileImage: false,
    })),

  // Stands in for a settings file that already has a card shape stored, so the
  // boot path that reads one can actually be exercised.
  GetAccountCardConfig: () => state.cardConfig,
  SetAccountCardConfig: (args) => {
    if (args[0] && typeof args[0] === "object") state.cardConfig = args[0] as Record<string, unknown>;
  },

  ListTagDefinitions: () => TAGS,
  GetAccountNote: () => "",
  GetPlatformExeIcon: () => "",

  // --- Home and Manage Platforms ---

  GetStartup: () => ({
    homePlatformOrder: state.homeOrder.filter((n) => !state.disabled.includes(n)),
    allPlatformNames: [...HOME_PLATFORMS, ...DISABLED_PLATFORMS],
    disabledPlatformNames: state.disabled,
    platformsFileMissing: false,
    platformAccountCounts: { "Steam": 3, "BattleNet": 4, "Epic Games": 2, "Riot Games": 1, "GOG Galaxy": 5 },
    platformTagCounts: { Steam: { tagCount: 2, taggedAccountCount: 2 } },
    language: "en-US",
    offlineMode: false,
    protocolEnabled: true,
    exitToTray: true,
    discordRpc: false,
    minimizeOnSwitch: true,
    startTrayWithWindows: false,
    startProgramCentered: false,
    streamerMode: false,
    autoStreamerMode: true,
    hideFromScreenshots: true,
    animationsEnabled: true,
    controllerSupportEnabled: true,
    prereleaseUpdates: false,
    commandPaletteHotkey: "Ctrl+K",
    themeAccentPreset: "",
    themeAccentCustom: "",
    themeHueRotate: 0,
    roundedCorners: true,
    discordAppId: "",
    appVersion: "1.4.0-dev",
  }),

  SaveHomeOrder: (args) => {
    if (Array.isArray(args[0])) state.homeOrder = args[0] as string[];
  },
  SetDisabledPlatforms: (args) => {
    if (Array.isArray(args[0])) state.disabled = args[0] as string[];
  },

  // Every canned platform "installs" instantly, so tiles navigate rather than
  // opening the locate-the-exe flow.
  ResolvePlatformLaunch: () => ({
    ok: true,
    needsManualLocate: false,
    foundViaShortcut: false,
    soughtExeName: "",
    initialPath: "",
  }),

  // --- App settings ---

  GetAppVersion: () => "1.4.0-dev",
  GetLanguage: () => "en-US",
  // A factor, not a percentage: 1 is 100%, and 0 would mean automatic.
  GetUIScale: () => 1,
  GetAnimationsEnabled: () => true,
  GetRoundedCorners: () => true,
  GetProtocolEnabled: () => true,
  GetExitToTray: () => true,
  GetMinimizeOnSwitch: () => true,
  GetAutoStreamerMode: () => true,
  GetHideFromScreenshots: () => true,
  GetUserDataLocation: () => "C:\\Users\\dev\\AppData\\Roaming\\AccountSwitcher",
  GetDefaultUserDataLocation: () => "C:\\Users\\dev\\AppData\\Roaming\\AccountSwitcher",
  CredentialStoreAvailable: () => true,
  GetSecurityStatus: () => ({
    appPasswordSet: false,
    appLocked: false,
    savedAccountDataEncrypted: false,
    operationBusy: false,
    quarantineCount: 0,
    interruptedRestorePending: false,
  }),

  // --- Platform and Steam settings ---

  GetPlatformSettings: (args) => {
    const name = String(args[0] ?? "");
    let s = state.platformSettings.get(name);
    if (!s) {
      s = defaultPlatformSettings();
      state.platformSettings.set(name, s);
    }
    return s;
  },
  SavePlatformSettings: (args) => {
    const name = String(args[0] ?? "");
    if (args[1] && typeof args[1] === "object") {
      state.platformSettings.set(name, args[1] as Record<string, unknown>);
    }
  },
  ResetPlatformSettings: (args) => {
    state.platformSettings.delete(String(args[0] ?? ""));
  },

  GetSteamSettings: () => {
    if (!state.steamSettings) state.steamSettings = defaultSteamSettings();
    return state.steamSettings;
  },
  SaveSteamSettings: (args) => {
    if (args[0] && typeof args[0] === "object") state.steamSettings = args[0] as Record<string, unknown>;
  },

  GetPlatformInstallFolder: (args) => {
    const name = String(args[0] ?? "");
    return name === "Steam" ? "C:\\Program Files (x86)\\Steam" : `C:\\Program Files\\${name}`;
  },

  GetSteamIDFormats: (args) => {
    const id64 = String(args[0] ?? "76561198000000001");
    const id32 = String(BigInt(id64) - 76561197960265728n);
    return {
      ID64: id64,
      ID3: `[U:1:${id32}]`,
      STEAMx: `STEAM_1:${BigInt(id32) % 2n}:${BigInt(id32) / 2n}`,
      ID32: id32,
      FriendCode: id32,
    };
  },

  // --- Games tab and shortcut bar ---

  GetOwnedGames: () => OWNED_GAMES,
  GetPlatformGames: (args) => ({
    platformKey: String(args[0] ?? "Steam"),
    games: RESOLVED_GAMES,
    warnings: [],
    unsupported: false,
    durationMs: 42,
    activeAccountId: "76561198000000001",
    accounts: [
      { accountId: "76561198000000001", accountName: "Main" },
      { accountId: "76561198000000002", accountName: "Alt" },
      { accountId: "76561198000000003", accountName: "Smurf" },
    ],
  }),
  GetGames: () => ({
    games: RESOLVED_GAMES,
    platforms: [{ platformKey: "Steam", games: RESOLVED_GAMES, warnings: [], unsupported: false, durationMs: 42 }],
    resolvedAt: "2026-08-24T00:00:00Z",
    usedNetwork: false,
  }),
  SupportedPlatforms: () => ["Steam"],
  GetInstalledGames: () => OWNED_GAMES.filter((g) => g.installed).map((g) => ({ appId: g.appId, name: g.name })),
  ListShortcuts: () => SHORTCUTS,

  // --- Steam advanced clearing ---

  AdvancedClearingRegistrySupported: () => true,
  RunAdvancedClearingAction: (args) => ({
    lines: [
      `[stub] ${String(args[0] ?? "?")}: deleted 3 files (1.2 MB)`,
      "[stub] done, nothing was actually touched",
    ],
  }),
};

/**
 * Lets a preset be chosen from the URL (`?card=medium`) while there is no
 * settings UI to choose it from yet. Persistence replaces this later.
 */
export async function applyDevCardPreset(): Promise<void> {
  const raw = new URLSearchParams(location.search).get("card");
  if (!raw) return;
  const preset = raw.trim().toLowerCase();
  if (!["small", "medium", "large"].includes(preset)) return;
  const { accountCardConfig } = await import("../stores/accountCard");
  accountCardConfig.set({ version: 1, preset: preset as "small" | "medium" | "large" });
  console.info(`[wailsStub] card preset forced to ${preset}`);
}

export function installWailsStub(): void {
  const bindings = buildBindingIndex();
  const unknown = new Set<number>();

  setTransport({
    async call(objectID: number, _method: number, _windowName: string, args: unknown) {
      // Everything that is not a bound service call (events, window, dialogs)
      // only needs to not reject for the shell to finish booting.
      if (objectID !== objectNames.Call) return null;

      const payload = (args ?? {}) as { methodID?: number; args?: unknown[] };
      const id = Number(payload.methodID);
      const info = bindings.get(id);

      if (!info) {
        if (!unknown.has(id)) {
          unknown.add(id);
          console.warn("[wailsStub] no binding found for method id", id);
        }
        return null;
      }

      const canned = CANNED[info.name];
      if (canned) return canned(payload.args ?? []);
      return defaultForType(info.returnType);
    },
  });

  console.info(`[wailsStub] active, ${bindings.size} bindings indexed`);
}
