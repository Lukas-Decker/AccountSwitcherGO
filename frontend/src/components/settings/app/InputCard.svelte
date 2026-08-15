<script lang="ts">
  /* Controller support and the hotkey that opens the command palette. */
  import { onDestroy } from "svelte";
  import { t } from "../../../stores/i18n";
  import {
    commandPaletteHotkey,
    formatCommandPaletteHotkeyEvent,
    setCommandPaletteHotkey,
  } from "../../../stores/commandPalette";
  import { controllerSupport } from "../../../lib/appSettingsModel";
  import { settingsIcons } from "./settingsIcons";
  import SettingsCard from "./SettingsCard.svelte";
  import SettingsRow from "./SettingsRow.svelte";
  import SettingsSwitch from "./SettingsSwitch.svelte";

  let capturing = false;

  function stopCapture(): void {
    if (!capturing) return;
    capturing = false;
    window.removeEventListener("keydown", onCaptureKeydown, true);
  }

  function startCapture(): void {
    if (capturing) return;
    capturing = true;
    // Capture phase: the combination must not reach the rest of the app.
    window.addEventListener("keydown", onCaptureKeydown, true);
  }

  function toggleCapture(): void {
    if (capturing) {
      stopCapture();
      return;
    }
    startCapture();
  }

  function onCaptureKeydown(e: KeyboardEvent): void {
    if (!capturing) return;
    e.preventDefault();
    e.stopPropagation();
    if (e.key === "Escape") {
      stopCapture();
      return;
    }
    const next = formatCommandPaletteHotkeyEvent(e);
    if (!next) return;
    stopCapture();
    void setCommandPaletteHotkey(next);
  }

  onDestroy(stopCapture);
</script>

<SettingsCard
  title={$t("Settings_Header_Input")}
  icon={settingsIcons.input}
  keywords="controller gamepad keyboard hotkey"
>
  <SettingsRow
    label={$t("Settings_ControllerSupport")}
    controlId="settings-controller"
    keywords="gamepad xbox playstation"
  >
    <SettingsSwitch
      id="settings-controller"
      toggle={controllerSupport}
      label={$t("Settings_ControllerSupport")}
    />
  </SettingsRow>

  <SettingsRow
    label={$t("Settings_CommandPaletteHotkey")}
    controlId="settings-hotkey"
    keywords="shortcut key combination palette search"
  >
    <button
      id="settings-hotkey"
      type="button"
      class="settings-btn"
      class:settings-btn--active={capturing}
      aria-pressed={capturing}
      on:click={toggleCapture}
    >
      {capturing ? $t("Settings_CommandPaletteHotkey_Prompt") : $commandPaletteHotkey}
    </button>
  </SettingsRow>
</SettingsCard>
