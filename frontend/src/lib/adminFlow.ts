import { get } from "svelte/store";
import * as PlatformService from "../../bindings/account-switcher/internal/platform/platformservice.js";
import { t } from "../stores/i18n";
import { openConfirm, openConfirmWithOptOut } from "../stores/modal";
import { pushToast } from "../stores/toast";
import { formatToastWithError, formatWailsError } from "./formatWailsError";

const needsAdminMarker = "NEEDS_ADMIN:";

/** Error payload includes NEEDS_ADMIN: prefix from the Go backend, or Win32 elevation (740). */
export function isNeedsAdminError(err: unknown): boolean {
  const s = formatWailsError(err);
  if (s.includes(needsAdminMarker)) {
    return true;
  }
  // Wails/Go may surface fork/exec with JSON cause: {"Err":740,...} (ERROR_ELEVATION_REQUIRED).
  if (/"Err"\s*:\s*740\b/.test(s)) {
    return true;
  }
  return false;
}


/**
 * Asks whether to restart elevated, unless the user has said not to ask again.
 *
 * The opt-out is a standing yes, so it is stored only when the answer was yes:
 * ticking the box and then declining would otherwise record a preference the
 * user never expressed. The prompt itself disables the negative button once the
 * box is ticked, so that combination cannot be submitted in the first place.
 */
async function confirmElevate(): Promise<boolean> {
  const tr = get(t);
  try {
    if (await PlatformService.GetSkipElevatePrompt()) {
      return true;
    }
  } catch {
    // Unreadable preference means ask, which is the safe direction.
  }
  const { ok, checked: remember } = await openConfirmWithOptOut({
    title: tr("Modal_Title_ConfirmAction"),
    body: tr("Prompt_RestartAsAdmin"),
    positiveLabel: tr("Ok"),
    negativeLabel: tr("No"),
    checkboxLabel: tr("Prompt_RestartAsAdmin_NeverAsk"),
  });
  if (ok && remember) {
    try {
      await PlatformService.SetSkipElevatePrompt(true);
    } catch {
      // Failing to remember is not a reason to refuse the restart itself.
    }
  }
  return ok;
}

export async function preflightAdminForPlatform(platformKey: string): Promise<void> {
  const key = String(platformKey ?? "").trim();
  if (!key) {
    return;
  }
  try {
    const r = await PlatformService.CheckAdminForPlatform(key);
    if (!r.needsAdmin) {
      return;
    }
    const tr = get(t);
    const ok = await confirmElevate();
    if (ok) {
      await PlatformService.RestartAsAdmin([`--page=${key}`]);
      return;
    }
    pushToast({
      type: "info",
      title: "",
      message: tr("Toast_RestartAsAdmin"),
      duration: 8000,
    });
  } catch (e) {
    pushToast({
      type: "error",
      title: "",
      message: formatWailsError(e) || String(e),
      duration: 6000,
    });
  }
}

export async function offerRestartIfNeedsAdmin(err: unknown, platformKey: string): Promise<void> {
  if (!isNeedsAdminError(err)) {
    return;
  }
  const key = String(platformKey ?? "").trim();
  const tr = get(t);
  const ok = await confirmElevate();
  if (ok && key) {
    await PlatformService.RestartAsAdmin([`--page=${key}`]);
  }
}

export async function reportLaunchFailure(err: unknown, platformKey: string): Promise<void> {
  await offerRestartIfNeedsAdmin(err, platformKey);
  const tr = get(t);
  if (isNeedsAdminError(err)) {
    pushToast({
      type: "error",
      title: "",
      message: tr("Toast_RestartAsAdmin"),
      duration: 8000,
    });
    return;
  }
  pushToast({
    type: "error",
    message: formatToastWithError(tr("Toast_LaunchFailed"), err),
    duration: 8000,
  });
}
