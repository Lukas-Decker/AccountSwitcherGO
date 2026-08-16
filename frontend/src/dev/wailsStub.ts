/**
 * A stand-in for the Go backend, for driving the UI in a plain browser.
 *
 * Wails talks to Go over native IPC, so `vite dev` on its own boots the shell
 * and then every call fails: nothing renders and no screen can be checked.
 * This replaces the runtime's transport with one that answers from canned
 * data, which is enough to exercise the account list, the card and the
 * settings screens without a backend.
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

const TAGS = [
  { id: "t1", name: "main", colour: "#8aff80" },
  { id: "t2", name: "alt", colour: "#80ffea" },
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
      avatarFrameUrl: "",
      miniProfileHtml: "",
      showMiniProfile: false,
      showAvatarFrame: false,
      syncError: i === 2 ? "Could not reach the Steam Web API" : "",
      tags: i < 2 ? [TAGS[i]] : [],
      manualProfileImage: false,
    })),

  GetTagDefs: () => TAGS,
  GetAccountNote: () => "",
  GetPlatformExeIcon: () => "",
  HasGameStatsSupport: () => false,
};

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
