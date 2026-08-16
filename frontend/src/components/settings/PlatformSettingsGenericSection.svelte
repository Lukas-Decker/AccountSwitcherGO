<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { t } from "../../stores/i18n";
  import {
    closingValues,
    startingValues,
    closingLabel,
    startingLabel,
  } from "../../lib/platformSettingsShared";
  import SettingsRow from "./app/SettingsRow.svelte";
  import SharedSettingCheckbox from "./SharedSettingCheckbox.svelte";
  import ProcessMethodDropdown from "./ProcessMethodDropdown.svelte";
  import type { PlatformSettings } from "../../../bindings/account-switcher/internal/platform/models";

  export let name: string;
  export let genericPS: PlatformSettings;
  export let hasDesktopShortcut: boolean = false;
  export let closingMethodUiLocked: boolean = false;
  export let hasRemoteProfileImages: boolean = false;

  const dispatch = createEventDispatcher();

  function pullAccountImagesOnSwitch(): boolean {
    const g = genericPS as unknown as Record<string, unknown>;
    return g.PullAccountImagesOnSwitch !== false;
  }

  function handlePullAccountImagesChange(): void {
    const g = genericPS as unknown as Record<string, unknown>;
    g.PullAccountImagesOnSwitch = !pullAccountImagesOnSwitch();
    dispatch("save");
  }
</script>

<h2 class="SettingsHeader">{$t("Settings_Header_GeneralSettings")}</h2>
<SharedSettingCheckbox
  id="gp-desktop-shortcut"
  checked={hasDesktopShortcut}
  label={$t("Settings_Shortcut", { platform: name })}
  on:change={() => dispatch("toggleDesktopShortcut")}
/>
<SharedSettingCheckbox
  id="gp-run-admin"
  checked={genericPS.RunAsAdmin}
  label={$t("Settings_Admin", { platform: name })}
  on:change={() => {
    genericPS.RunAsAdmin = !genericPS.RunAsAdmin;
    dispatch("save");
  }}
/>
<SharedSettingCheckbox
  id="gp-autostart"
  checked={genericPS.AutoStart}
  label={$t("Settings_AutoStart", { platform: name })}
  on:change={() => {
    genericPS.AutoStart = !genericPS.AutoStart;
    dispatch("save");
  }}
/>
<SharedSettingCheckbox
  id="gp-forget"
  checked={genericPS.ForgetAccountEnabled}
  label={$t("Settings_ForgetAccountEnabled")}
  on:change={() => {
    genericPS.ForgetAccountEnabled = !genericPS.ForgetAccountEnabled;
    dispatch("save");
  }}
/>

<h2 class="SettingsHeader">{$t("Settings_Header_LaunchOptions")}</h2>
<SettingsRow
  label={$t("Settings_LaunchArgumentsForPlatform", { platform: name })}
  hint={$t("Settings_LaunchArguments_Hint")}
  controlId="gp-launch-args"
  disabled={!genericPS.AutoStart}
  stacked
>
  <input
    id="gp-launch-args"
    class="settings-input"
    type="text"
    spellcheck="false"
    autocomplete="off"
    disabled={!genericPS.AutoStart}
    bind:value={genericPS.LaunchArguments}
    on:input={() => dispatch("save")}
  />
</SettingsRow>

<h2 class="SettingsHeader">{$t("Settings_Header_ProcessManagement")}</h2>
{#if !closingMethodUiLocked}
  <ProcessMethodDropdown
    values={closingValues}
    current={genericPS.ClosingMethod}
    label={$t("Settings_Header_ClosingMethod", { platform: name })}
    labelFn={closingLabel}
    tooltip={$t("Tooltip_ClosingMethod")}
    on:select={(e) => {
      genericPS.ClosingMethod = e.detail.value;
      dispatch("save");
    }}
  />
{/if}

<!-- How long the platform gets to close on its own, and what happens when it
     does not. Terminating a launcher mid-write loses the session the switcher is
     about to read, so whether to go that far is the user's call. -->
<SettingsRow
  label={$t("Settings_CloseTimeout")}
  hint={$t("Settings_CloseTimeout_Hint")}
  controlId="gp-close-timeout"
>
  <input
    id="gp-close-timeout"
    type="number"
    min="0"
    max="120"
    value={genericPS.CloseTimeoutSeconds ?? 0}
    on:change={(e) => {
      const n = Number.parseInt(e.currentTarget.value, 10);
      genericPS.CloseTimeoutSeconds = Number.isFinite(n) && n > 0 ? Math.min(n, 120) : 0;
      dispatch("save");
    }}
  />
</SettingsRow>

<SharedSettingCheckbox
  id="gs-force-close"
  checked={genericPS.ForceCloseAfterTimeout !== false}
  label={$t("Settings_ForceClose")}
  tooltip={$t("Tooltip_ForceClose")}
  on:change={() => {
    // Absent means forced (the historical behaviour), so the toggle flips
    // between an explicit false and an explicit true.
    genericPS.ForceCloseAfterTimeout = genericPS.ForceCloseAfterTimeout === false;
    dispatch("save");
  }}
/>

<ProcessMethodDropdown
  values={startingValues}
  current={genericPS.StartingMethod}
  label={$t("Settings_Header_StartingMethod", { platform: name })}
  labelFn={startingLabel}
  tooltip={$t("Tooltip_StartingMethod")}
  on:select={(e) => {
    genericPS.StartingMethod = e.detail.value;
    dispatch("save");
  }}
/>

<h2 class="SettingsHeader">{$t("Settings_Header_TraySettings")}</h2>
<SettingsRow label={$t("Settings_TrayMax")} controlId="gp-tray-max">
  <input
    id="gp-tray-max"
    type="number"
    min="0"
    max="365"
    bind:value={genericPS.TrayAccNumber}
    on:change={() => dispatch("save")}
  />
</SettingsRow>

{#if hasRemoteProfileImages}
  <h2 class="SettingsHeader">{$t("Settings_Header_ProfileImages")}</h2>
  <SharedSettingCheckbox
    id="gp-pull-account-images"
    checked={pullAccountImagesOnSwitch()}
    label={$t("Settings_PullAccountImages")}
    on:change={handlePullAccountImagesChange}
  />
  <SettingsRow label={$t("Settings_ProfileImageExpiryDays")} controlId="gp-image-expiry">
    <input
      id="gp-image-expiry"
      type="number"
      min="1"
      max="365"
      bind:value={genericPS.ProfileImageExpiryDays}
      on:change={() => dispatch("save")}
    />
  </SettingsRow>
  <div class="buttoncol">
    <button type="button" on:click={() => dispatch("refreshBasicProfileImages")}>
      {$t("Button_RefreshImages")}
    </button>
    <button type="button" on:click={() => dispatch("clearBasicProfileImages")}>
      {$t("Button_ClearCachedProfileImages")}
    </button>
  </div>
{/if}
