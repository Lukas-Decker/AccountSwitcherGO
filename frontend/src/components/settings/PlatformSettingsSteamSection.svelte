<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { viewportDropdown } from "../../lib/actions/viewportDropdown";
  import { t } from "../../stores/i18n";
  import {
    closingValues,
    startingValues,
    closingLabel,
    startingLabel,
    overrideStates,
    withLaunchArgFlag,
  } from "../../lib/platformSettingsShared";
  import SettingsRow from "./app/SettingsRow.svelte";
  import SharedSettingCheckbox from "./SharedSettingCheckbox.svelte";
  import ProcessMethodDropdown from "./ProcessMethodDropdown.svelte";
  import type { Settings } from "../../../bindings/account-switcher/internal/steam/models";

  export let name: string;
  export let steamSettings: Settings;
  export let hasDesktopShortcut: boolean = false;
  export let silentOn: boolean = false;
  export let oldUiOn: boolean = false;
  export let closingMethodUiLocked: boolean = false;

  const dispatch = createEventDispatcher();
  const ARG_SILENT = "-silent";
  const ARG_VGUI = "-vgui";
  let stateOpen = false;

  function overrideLabel(v: number): string {
    const row = overrideStates.find((x) => x.v === v);
    return row ? $t(row.key) : $t("NoDefault");
  }
</script>

<h2 class="SettingsHeader">{$t("Settings_Header_GeneralSettings")}</h2>
<SharedSettingCheckbox
  id="ps-desktop-shortcut"
  checked={hasDesktopShortcut}
  label={$t("Settings_Shortcut", { platform: name })}
  on:change={() => dispatch("toggleDesktopShortcut")}
/>
<SharedSettingCheckbox
  id="ps-run-admin"
  checked={steamSettings.RunAsAdmin}
  label={$t("Settings_Admin", { platform: name })}
  on:change={() => {
    steamSettings.RunAsAdmin = !steamSettings.RunAsAdmin;
    dispatch("save");
  }}
/>
<SharedSettingCheckbox
  id="ps-autostart"
  checked={steamSettings.AutoStart}
  label={$t("Settings_AutoStart", { platform: name })}
  on:change={() => {
    steamSettings.AutoStart = !steamSettings.AutoStart;
    dispatch("save");
  }}
/>
<SharedSettingCheckbox
  id="ps-forget"
  checked={steamSettings.ForgetAccountEnabled}
  label={$t("Settings_ForgetAccountEnabled")}
  on:change={() => {
    steamSettings.ForgetAccountEnabled = !steamSettings.ForgetAccountEnabled;
    dispatch("save");
  }}
/>

<h2 class="SettingsHeader">{$t("Settings_Header_AccountDisplay")}</h2>
<SharedSettingCheckbox
  id="ps-show-miniprofile"
  checked={steamSettings.Steam_ShowMiniProfile}
  label={$t("Steam_ShowMiniProfile")}
  tooltip={$t("Tooltip_SteamShowMiniProfile")}
  on:change={() => {
    steamSettings.Steam_ShowMiniProfile = !steamSettings.Steam_ShowMiniProfile;
    dispatch("save");
  }}
/>
<SharedSettingCheckbox
  id="ps-show-avatar-frame"
  checked={steamSettings.Steam_ShowAvatarFrame}
  label={$t("Steam_ShowAvatarFrame")}
  tooltip={$t("Tooltip_SteamShowAvatarFrame")}
  on:change={() => {
    steamSettings.Steam_ShowAvatarFrame = !steamSettings.Steam_ShowAvatarFrame;
    dispatch("save");
  }}
/>

<h2 class="SettingsHeader">{$t("Settings_Header_TraySettings")}</h2>
<SharedSettingCheckbox
  id="ps-tray-name"
  checked={steamSettings.Steam_TrayAccountName}
  label={$t("Steam_Tray_AccountName")}
  on:change={() => {
    steamSettings.Steam_TrayAccountName = !steamSettings.Steam_TrayAccountName;
    dispatch("save");
  }}
/>
<SettingsRow label={$t("Settings_TrayMax")} controlId="ps-tray-max">
  <input
    id="ps-tray-max"
    type="number"
    min="0"
    max="365"
    bind:value={steamSettings.TrayAccNumber}
    on:change={() => dispatch("save")}
  />
</SettingsRow>

<h2 class="SettingsHeader">{$t("Settings_Header_LaunchOptions")}</h2>
<SharedSettingCheckbox
  id="ps-silent"
  checked={silentOn}
  disabled={!steamSettings.AutoStart}
  label={$t("Steam_StartSilent")}
  on:change={() => {
    steamSettings.LaunchArguments = withLaunchArgFlag(steamSettings.LaunchArguments ?? "", ARG_SILENT, !silentOn);
    dispatch("save");
  }}
/>
<SharedSettingCheckbox
  id="ps-steam-switcher"
  checked={steamSettings.ShowSteamSwitcher}
  label={$t("Settings_ShowSteamSwitcher")}
  on:change={() => {
    steamSettings.ShowSteamSwitcher = !steamSettings.ShowSteamSwitcher;
    dispatch("save");
  }}
/>
<SharedSettingCheckbox
  id="ps-collect"
  checked={steamSettings.CollectInfo}
  label={$t("Settings_SteamCollectInfo")}
  on:change={() => {
    steamSettings.CollectInfo = !steamSettings.CollectInfo;
    dispatch("save");
  }}
/>
<SharedSettingCheckbox
  id="ps-oldui"
  checked={oldUiOn}
  disabled={!steamSettings.AutoStart}
  label={$t("Steam_OldUi")}
  on:change={() => {
    steamSettings.LaunchArguments = withLaunchArgFlag(steamSettings.LaunchArguments ?? "", ARG_VGUI, !oldUiOn);
    dispatch("save");
  }}
/>
<SettingsRow
  label={$t("Settings_LaunchArgumentsForPlatform", { platform: name })}
  hint={$t("Settings_LaunchArguments_Hint")}
  controlId="ps-launch-args"
  disabled={!steamSettings.AutoStart}
  stacked
>
  <input
    id="ps-launch-args"
    class="settings-input"
    type="text"
    spellcheck="false"
    autocomplete="off"
    disabled={!steamSettings.AutoStart}
    bind:value={steamSettings.LaunchArguments}
    on:input={() => dispatch("save")}
  />
</SettingsRow>
<SettingsRow label={$t("Steam_OverrideDefaultState")}>
  <div class="dropdown" class:show={stateOpen}>
    <button type="button" class="dropdown-toggle" on:click={() => (stateOpen = !stateOpen)}>
      {overrideLabel(steamSettings.Steam_OverrideState)}
      <span class="caret" aria-hidden="true"></span>
    </button>
    {#if stateOpen}
      <ul class="custom-dropdown-menu dropdown-menu" use:viewportDropdown>
        {#each overrideStates as o}
          <li>
            <button
              type="button"
              class="dropdown-item"
              on:click={() => {
                steamSettings.Steam_OverrideState = o.v;
                stateOpen = false;
                dispatch("save");
              }}
            >
              {$t(o.key)}
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</SettingsRow>
<SettingsRow label={$t("Settings_ImageExpiry")} controlId="ps-image-expiry">
  <input
    id="ps-image-expiry"
    type="number"
    min="0"
    max="365"
    bind:value={steamSettings.Steam_ImageExpiryTime}
    on:change={() => dispatch("save")}
  />
</SettingsRow>
<SettingsRow
  label={$t("Settings_SteamAPIKey")}
  hint={$t("Settings_SteamAPIKey_Note")}
  controlId="ps-api-key"
  stacked
>
  <input
    id="ps-api-key"
    class="settings-input"
    type="text"
    spellcheck="false"
    bind:value={steamSettings.SteamWebApiKey}
    on:change={() => dispatch("save")}
  />
</SettingsRow>

<h2 class="SettingsHeader">{$t("Settings_Header_ProcessManagement")}</h2>
{#if !closingMethodUiLocked}
  <ProcessMethodDropdown
    values={closingValues}
    current={steamSettings.ClosingMethod}
    label={$t("Settings_Header_ClosingMethod", { platform: name })}
    labelFn={closingLabel}
    tooltip={$t("Tooltip_ClosingMethod")}
    on:select={(e) => {
      steamSettings.ClosingMethod = e.detail.value;
      dispatch("save");
    }}
  />
{/if}
<ProcessMethodDropdown
  values={startingValues}
  current={steamSettings.StartingMethod}
  label={$t("Settings_Header_StartingMethod", { platform: name })}
  labelFn={startingLabel}
  tooltip={$t("Tooltip_StartingMethod")}
  on:select={(e) => {
    steamSettings.StartingMethod = e.detail.value;
    dispatch("save");
  }}
/>
