/*
  State behind the app settings grid.

  Every switch in the grid is one of these controllers: it holds the value, the
  in-flight flag, how to read it, how to write it and what to say afterwards.
  The cards then only have to render, which is why several cards can share one
  hydration pass instead of each fetching its own copy.

  This is a module singleton because the settings are global to the app. The
  grid re-hydrates on mount, so a value changed elsewhere (the tray menu, the
  command palette) is picked up the next time settings are opened.
*/

import { derived, get, writable, type Readable, type Writable } from "svelte/store";
import * as PlatformService from "../../bindings/account-switcher/internal/platform/platformservice.js";
import { t } from "../stores/i18n";
import { pushToast } from "../stores/toast";
import { formatToastWithError } from "./formatWailsError";
import { initOfflineMode, offlineMode, setUserOfflineMode } from "../stores/offlineMode";
import { setAutoStreamerMode, setStreamerMode } from "../stores/streamerMode";
import {
  animationsEnabled,
  loadAnimationsEnabled,
  setAnimationsEnabled,
} from "../stores/animationSettings";
import {
  applyControllerSupportEnabled,
  loadControllerSupportEnabled,
  setControllerSupportEnabled,
} from "../stores/controllerSupport";
import {
  commandPaletteHotkey,
  loadCommandPaletteHotkey,
  normalizeCommandPaletteHotkey,
} from "../stores/commandPalette";
import { roundedCorners, setRoundedCorners } from "../stores/roundedCorners";

export interface ToggleState {
  value: boolean;
  busy: boolean;
}

export interface SettingToggle extends Readable<ToggleState> {
  value: Writable<boolean>;
  busy: Writable<boolean>;
  /** Reads the current state from the backend. Failures fall back to off. */
  load: () => Promise<void>;
  /** Writes a new state, showing the switch as flipped while it saves. */
  apply: (next: boolean) => Promise<void>;
}

interface ToastSpec {
  message: string;
  duration: number;
}

interface ToggleOptions {
  read: () => Promise<boolean>;
  write: (next: boolean) => Promise<void>;
  /** Key of the label used in the generic "Saved ..." toast. */
  labelKey: string;
  /** Returns true when the change must not happen, e.g. offline mode is on. */
  blocked?: () => boolean;
  /** Runs after a successful write; used where the backend owns the truth. */
  after?: (next: boolean) => Promise<void> | void;
  /** Replaces the generic saved toast for settings with real consequences. */
  toast?: (next: boolean) => ToastSpec;
}

function createToggle(options: ToggleOptions): SettingToggle {
  const value = writable(false);
  const busy = writable(false);
  const state = derived([value, busy], ([$value, $busy]) => ({ value: $value, busy: $busy }));

  async function apply(next: boolean): Promise<void> {
    if (get(busy) || next === get(value) || options.blocked?.()) {
      return;
    }
    const previous = get(value);
    // Flip first: a switch that waits for a disk write feels broken.
    value.set(next);
    busy.set(true);
    try {
      await options.write(next);
      await options.after?.(next);
      const spec = options.toast?.(next) ?? {
        message: get(t)("Toast_SavedItem", { item: get(t)(options.labelKey) }),
        duration: 4000,
      };
      pushToast({ type: "success", message: spec.message, duration: spec.duration });
    } catch (e) {
      value.set(previous);
      pushToast({
        type: "error",
        message: formatToastWithError(get(t)("Toast_SaveFailed"), e),
        duration: 8000,
      });
    } finally {
      busy.set(false);
    }
  }

  return {
    subscribe: state.subscribe,
    value,
    busy,
    load: async () => {
      try {
        value.set(await options.read());
      } catch {
        value.set(false);
      }
    },
    apply,
  };
}

/** True on Windows, where the tray, protocol and shortcut settings apply. */
export function isWindowsHost(): boolean {
  return /windows|win32/i.test(navigator.userAgent);
}

export const exitToTray = createToggle({
  read: () => PlatformService.GetExitToTray(),
  write: (v) => PlatformService.SetExitToTray(v),
  labelKey: "Settings_ExitToTray",
});

export const startTrayWithWindows = createToggle({
  read: () => PlatformService.GetStartTrayWithWindows(),
  write: (v) => PlatformService.SetStartTrayWithWindows(v),
  labelKey: "Settings_Tray_StartWindows",
});

export const minimizeOnSwitch = createToggle({
  read: () => PlatformService.GetMinimizeOnSwitch(),
  write: (v) => PlatformService.SetMinimizeOnSwitch(v),
  labelKey: "Settings_MinimizeOnSwitch",
});

export const startProgramCentered = createToggle({
  read: () => PlatformService.GetStartProgramCentered(),
  write: (v) => PlatformService.SetStartProgramCentered(v),
  labelKey: "Settings_StartCentered",
});

export const desktopShortcut = createToggle({
  read: () => PlatformService.GetDesktopHomeShortcutExists(),
  write: (v) => PlatformService.SetDesktopHomeShortcut(v),
  labelKey: "Settings_DesktopShortcut",
  // The shortcut file is the source of truth, not the value we just sent.
  after: async () => {
    try {
      desktopShortcut.value.set(await PlatformService.GetDesktopHomeShortcutExists());
    } catch {
      /* leave the optimistic value in place */
    }
  },
});

export const protocolEnabled = createToggle({
  read: () => PlatformService.GetProtocolEnabled(),
  write: (v) => PlatformService.SetProtocolEnabled(v),
  labelKey: "Settings_Protocol",
  toast: (next) => ({
    message: get(t)(next ? "Toast_ProtocolEnabled" : "Toast_ProtocolDisabled"),
    duration: 6000,
  }),
});

export const streamerMode = createToggle({
  read: () => PlatformService.GetStreamerMode(),
  write: (v) => setStreamerMode(v),
  labelKey: "Settings_StreamerMode",
});

export const autoStreamerMode = createToggle({
  read: () => PlatformService.GetAutoStreamerMode(),
  write: (v) => setAutoStreamerMode(v),
  labelKey: "Settings_AutoStreamerMode",
});

export const hideFromScreenshots = createToggle({
  read: () => PlatformService.GetHideFromScreenshots(),
  write: (v) => PlatformService.SetHideFromScreenshots(v),
  labelKey: "Settings_HideFromScreenshots",
});

export const offlineModeToggle = createToggle({
  read: async () => {
    await initOfflineMode();
    return get(offlineMode);
  },
  write: (v) => setUserOfflineMode(v),
  labelKey: "Settings_OfflineMode",
  // Rich presence needs the network, so it goes down with the connection.
  after: (next) => {
    if (next) {
      discordRpc.value.set(false);
    }
  },
  toast: (next) => ({
    message: get(t)(next ? "Toast_OfflineModeEnabled" : "Toast_OfflineModeDisabled"),
    duration: 6000,
  }),
});

export const discordRpc = createToggle({
  read: () => PlatformService.GetDiscordRpc(),
  write: (v) => PlatformService.SetDiscordRpc(v),
  labelKey: "Settings_DiscordRpc",
  blocked: () => get(offlineMode),
});

export const animations = createToggle({
  read: async () => {
    await loadAnimationsEnabled();
    return get(animationsEnabled);
  },
  write: (v) => setAnimationsEnabled(v),
  labelKey: "Settings_AnimationsEnabled",
});

export const controllerSupport = createToggle({
  read: () => loadControllerSupportEnabled(),
  write: (v) => setControllerSupportEnabled(v),
  labelKey: "Settings_ControllerSupport",
});

export const roundedCornersToggle = createToggle({
  read: async () => get(roundedCorners),
  write: (v) => setRoundedCorners(v),
  labelKey: "Settings_RoundedCorners",
});

export const prereleaseUpdates = createToggle({
  read: () => PlatformService.GetPrereleaseUpdates(),
  write: (v) => PlatformService.SetPrereleaseUpdates(v),
  labelKey: "Settings_PrereleaseUpdates",
});

export const debugLogging = createToggle({
  read: () => PlatformService.GetDebugLogging(),
  write: (v) => PlatformService.SetDebugLogging(v),
  labelKey: "Settings_DebugLogging",
});

export const skipElevatePrompt = createToggle({
  read: () => PlatformService.GetSkipElevatePrompt(),
  write: (v) => PlatformService.SetSkipElevatePrompt(v),
  labelKey: "Settings_SkipElevatePrompt",
});

/** Discord names the presence after the app the id belongs to. */
export const discordAppId = writable("");

/** The artwork archive key. Empty leaves the archive off, which is the default:
 * it is a third-party service, and nothing reaches it until the user opts in by
 * pasting a key of their own. */
export const gameArtArchiveKey = writable("");
export const appVersion = writable("");
export const userDataPath = writable("");

export async function saveGameArtArchiveKey(value: string): Promise<void> {
  const next = value.trim();
  gameArtArchiveKey.set(next);
  try {
    await PlatformService.SetSteamGridDBAPIKey(next);
  } catch (e) {
    pushToast({
      type: "error",
      message: formatToastWithError(get(t)("Toast_SaveFailed"), e),
      duration: 8000,
    });
  }
}

export async function saveDiscordAppId(value: string): Promise<void> {
  const next = value.trim();
  discordAppId.set(next);
  try {
    await PlatformService.SetDiscordAppID(next);
  } catch (e) {
    pushToast({
      type: "error",
      message: formatToastWithError(get(t)("Toast_SaveFailed"), e),
      duration: 8000,
    });
  }
}

const windowsOnlyToggles = [startTrayWithWindows, desktopShortcut];

const alwaysLoadedToggles = [
  exitToTray,
  minimizeOnSwitch,
  startProgramCentered,
  protocolEnabled,
  streamerMode,
  autoStreamerMode,
  hideFromScreenshots,
  offlineModeToggle,
  discordRpc,
  animations,
  controllerSupport,
  roundedCornersToggle,
  prereleaseUpdates,
  debugLogging,
  skipElevatePrompt,
];

/** One backend round trip covers most of the grid; the rest load individually. */
async function hydrateFromSnapshot(): Promise<void> {
  const settings = await PlatformService.ReadSettings();
  offlineMode.set(settings.offlineMode);
  offlineModeToggle.value.set(settings.offlineMode);
  discordAppId.set(settings.discordAppId ?? "");
  gameArtArchiveKey.set(settings.steamGridDbApiKey ?? "");
  protocolEnabled.value.set(settings.protocolEnabled);
  exitToTray.value.set(settings.exitToTray);
  prereleaseUpdates.value.set(settings.prereleaseUpdates);
  discordRpc.value.set(settings.discordRpc);
  minimizeOnSwitch.value.set(settings.minimizeOnSwitch);
  startTrayWithWindows.value.set(settings.startTrayWithWindows);
  startProgramCentered.value.set(settings.startProgramCentered);
  streamerMode.value.set(settings.streamerMode);
  autoStreamerMode.value.set(settings.autoStreamerMode);
  hideFromScreenshots.value.set(settings.hideFromScreenshots);
  animationsEnabled.set(settings.animationsEnabled);
  animations.value.set(settings.animationsEnabled);
  controllerSupport.value.set(applyControllerSupportEnabled(settings.controllerSupportEnabled));
  roundedCornersToggle.value.set(settings.roundedCorners);
  commandPaletteHotkey.set(normalizeCommandPaletteHotkey(settings.commandPaletteHotkey));
  appVersion.set(settings.appVersion || "");
}

function hydrateIndividually(windowsHost: boolean): void {
  for (const toggle of alwaysLoadedToggles) {
    void toggle.load();
  }
  if (windowsHost) {
    for (const toggle of windowsOnlyToggles) {
      void toggle.load();
    }
  }
  void loadCommandPaletteHotkey();
  void PlatformService.GetDiscordAppID()
    .then((v) => discordAppId.set(v || ""))
    .catch(() => {});
  void PlatformService.GetAppVersion()
    .then((v) => appVersion.set(v || ""))
    .catch(() => appVersion.set(""));
}

/** Fills the whole grid. Safe to call again whenever the settings open. */
export async function hydrateAppSettings(): Promise<void> {
  const windowsHost = isWindowsHost();
  try {
    await hydrateFromSnapshot();
    // Not part of the snapshot: these are read from the OS, not the settings file.
    void debugLogging.load();
    void skipElevatePrompt.load();
    if (windowsHost) {
      void desktopShortcut.load();
    }
  } catch {
    hydrateIndividually(windowsHost);
  }
  void PlatformService.GetUserDataLocation()
    .then((v) => userDataPath.set(v || ""))
    .catch(() => userDataPath.set(""));
}
