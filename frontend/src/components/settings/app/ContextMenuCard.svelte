<script lang="ts">
  /* How the right-click menu looks, everywhere in the app that opens one. */
  import { t } from "../../../stores/i18n";
  import {
    contextMenuStyle,
    resetContextMenuStyle,
    setContextMenuStyle,
    type ContextMenuDensity,
    type ContextMenuFont,
    type ContextMenuHeaderStyle,
    type ContextMenuWeight,
  } from "../../../lib/contextMenuStyle";
  import { settingsIcons } from "./settingsIcons";
  import SettingsCard from "./SettingsCard.svelte";
  import SettingsRow from "./SettingsRow.svelte";
  import SettingsSelect from "./SettingsSelect.svelte";

  const uid = "ctxstyle";

  $: densityOptions = [
    { value: "slim", label: $t("Settings_ContextMenu_Density_Slim") },
    { value: "normal", label: $t("Settings_ContextMenu_Density_Normal") },
    { value: "roomy", label: $t("Settings_ContextMenu_Density_Roomy") },
  ];
  $: weightOptions = [
    { value: "normal", label: $t("Settings_ContextMenu_Weight_Normal") },
    { value: "medium", label: $t("Settings_ContextMenu_Weight_Medium") },
    { value: "bold", label: $t("Settings_ContextMenu_Weight_Bold") },
  ];
  /* Handlers rather than inline casts: a TypeScript `as` inside a Svelte
     attribute expression is not something the template parser accepts. */
  function onDensity(e: CustomEvent<string>): void {
    setContextMenuStyle({ density: e.detail as ContextMenuDensity });
  }
  function onWeight(e: CustomEvent<string>): void {
    setContextMenuStyle({ weight: e.detail as ContextMenuWeight });
  }
  function onFont(e: CustomEvent<string>): void {
    setContextMenuStyle({ font: e.detail as ContextMenuFont });
  }
  function onHeaderStyle(e: CustomEvent<string>): void {
    setContextMenuStyle({ headerStyle: e.detail as ContextMenuHeaderStyle });
  }

  $: headerStyleOptions = [
    { value: "accent", label: $t("Settings_ContextMenu_HeaderStyle_Accent") },
    { value: "plain", label: $t("Settings_ContextMenu_HeaderStyle_Plain") },
    { value: "band", label: $t("Settings_ContextMenu_HeaderStyle_Band") },
  ];

  $: fontOptions = [
    { value: "app", label: $t("Settings_ContextMenu_Font_App") },
    { value: "system", label: $t("Settings_ContextMenu_Font_System") },
    { value: "mono", label: $t("Settings_ContextMenu_Font_Mono") },
  ];
</script>

<SettingsCard
  title={$t("Settings_Header_ContextMenu")}
  icon={settingsIcons.contextMenu}
  keywords="context menu right click slim compact density icons font bold italic style header title section collapse"
>
  <SettingsRow
    label={$t("Settings_ContextMenu_Density")}
    hint={$t("Settings_ContextMenu_Density_Hint")}
    controlId="{uid}-density"
    keywords="slim compact spacing row height"
  >
    <SettingsSelect
      id="{uid}-density"
      label={$t("Settings_ContextMenu_Density")}
      options={densityOptions}
      value={$contextMenuStyle.density}
      on:select={onDensity}
    />
  </SettingsRow>

  <SettingsRow
    label={$t("Settings_ContextMenu_Icons")}
    hint={$t("Settings_ContextMenu_Icons_Hint")}
    controlId="{uid}-icons"
    keywords="glyph symbol picture"
  >
    <span class="ctxstyle-check">
      <input
        id="{uid}-icons"
        type="checkbox"
        checked={$contextMenuStyle.showIcons}
        on:change={(e) => setContextMenuStyle({ showIcons: e.currentTarget.checked })}
      />
      <label for="{uid}-icons" class="ctxstyle-box" aria-hidden="true"></label>
    </span>
  </SettingsRow>

  <SettingsRow
    label={$t("Settings_ContextMenu_Headers")}
    hint={$t("Settings_ContextMenu_Headers_Hint")}
    controlId="{uid}-headers"
    keywords="header title section group collapse fold"
  >
    <span class="ctxstyle-check">
      <input
        id="{uid}-headers"
        type="checkbox"
        checked={$contextMenuStyle.showHeaders}
        on:change={(e) => setContextMenuStyle({ showHeaders: e.currentTarget.checked })}
      />
      <label for="{uid}-headers" class="ctxstyle-box" aria-hidden="true"></label>
    </span>
  </SettingsRow>

  <SettingsRow
    label={$t("Settings_ContextMenu_HeaderStyle")}
    hint={$t("Settings_ContextMenu_HeaderStyle_Hint")}
    controlId="{uid}-headerstyle"
    keywords="header title accent colour color band"
  >
    <SettingsSelect
      id="{uid}-headerstyle"
      label={$t("Settings_ContextMenu_HeaderStyle")}
      options={headerStyleOptions}
      value={$contextMenuStyle.headerStyle}
      on:select={onHeaderStyle}
    />
  </SettingsRow>

  <SettingsRow
    label={$t("Settings_ContextMenu_FontSize")}
    controlId="{uid}-size"
    keywords="text size bigger smaller"
  >
    <input
      id="{uid}-size"
      type="range"
      min="10"
      max="20"
      step="1"
      value={$contextMenuStyle.fontSize}
      on:input={(e) => setContextMenuStyle({ fontSize: Number(e.currentTarget.value) })}
    />
    <span class="ctxstyle-value">{$contextMenuStyle.fontSize}px</span>
  </SettingsRow>

  <SettingsRow
    label={$t("Settings_ContextMenu_Weight")}
    controlId="{uid}-weight"
    keywords="bold weight emphasis"
  >
    <SettingsSelect
      id="{uid}-weight"
      label={$t("Settings_ContextMenu_Weight")}
      options={weightOptions}
      value={$contextMenuStyle.weight}
      on:select={onWeight}
    />
  </SettingsRow>

  <SettingsRow
    label={$t("Settings_ContextMenu_Font")}
    controlId="{uid}-font"
    keywords="typeface family monospace"
  >
    <SettingsSelect
      id="{uid}-font"
      label={$t("Settings_ContextMenu_Font")}
      options={fontOptions}
      value={$contextMenuStyle.font}
      on:select={onFont}
    />
  </SettingsRow>

  <SettingsRow
    label={$t("Settings_ContextMenu_Italic")}
    controlId="{uid}-italic"
    keywords="italic slanted style"
  >
    <span class="ctxstyle-check">
      <input
        id="{uid}-italic"
        type="checkbox"
        checked={$contextMenuStyle.italic}
        on:change={(e) => setContextMenuStyle({ italic: e.currentTarget.checked })}
      />
      <label for="{uid}-italic" class="ctxstyle-box" aria-hidden="true"></label>
    </span>
  </SettingsRow>

  <SettingsRow
    label={$t("Settings_ContextMenu_Reset")}
    controlId="{uid}-reset"
    keywords="default restore"
  >
    <button id="{uid}-reset" type="button" class="settings-link" on:click={resetContextMenuStyle}>
      {$t("Settings_ContextMenu_Reset_Action")}
    </button>
  </SettingsRow>
</SettingsCard>

<style lang="scss">
  /* The app hides the native box and draws it through a sibling label, which is
     why the control is a pair rather than one element. */
  .ctxstyle-check {
    display: inline-flex;
    align-items: center;
  }

  .ctxstyle-box {
    width: 0.9rem;
    height: 0.9rem;
    flex: none;
  }

  .ctxstyle-value {
    margin-left: 0.5rem;
    font-variant-numeric: tabular-nums;
    opacity: 0.75;
  }
</style>
