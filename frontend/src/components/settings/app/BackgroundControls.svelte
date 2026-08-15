<script lang="ts">
  /*
    Background image controls for either the app or the current platform.

    Only rendered once an image is actually set, since every control here acts
    on that image.
  */
  import { get } from "svelte/store";
  import { route } from "../../../stores/nav";
  import { t } from "../../../stores/i18n";
  import { pushToast } from "../../../stores/toast";
  import { formatToastWithError } from "../../../lib/formatWailsError";
  import {
    normalizeBackgroundAlignment,
    normalizeBackgroundFit,
  } from "../../../lib/backgroundDisplay";
  import * as PlatformService from "../../../../bindings/account-switcher/internal/platform/platformservice.js";
  import { appBgInfo, platformBgInfo } from "../../../stores/backgroundImage";
  import SettingsRow from "./SettingsRow.svelte";
  import SettingsSelect from "./SettingsSelect.svelte";

  export let target: "app" | "platform" = "app";
  /** Offered next to Clear when the active theme brings its own background. */
  export let onResetToTheme: (() => void) | null = null;

  const SAVE_DELAY_MS = 300;

  let opacityTimer: ReturnType<typeof setTimeout>;
  let blurTimer: ReturnType<typeof setTimeout>;

  $: store = target === "app" ? appBgInfo : platformBgInfo;
  $: info = target === "app" ? $appBgInfo : $platformBgInfo;
  $: alignment = normalizeBackgroundAlignment(info.alignment);
  $: fit = normalizeBackgroundFit(info.fit);

  $: alignmentOptions = [
    { value: "center", label: $t("Settings_BgAlign_Center") },
    { value: "left", label: $t("Settings_BgAlign_Left") },
    { value: "right", label: $t("Settings_BgAlign_Right") },
    { value: "top", label: $t("Settings_BgAlign_Top") },
    { value: "bottom", label: $t("Settings_BgAlign_Bottom") },
  ];

  $: fitOptions = [
    { value: "cover", label: $t("Settings_BgFit_Cover") },
    { value: "contain", label: $t("Settings_BgFit_Contain") },
    { value: "fill", label: $t("Settings_BgFit_Fill") },
    { value: "none", label: $t("Settings_BgFit_None") },
    { value: "scale-down", label: $t("Settings_BgFit_ScaleDown") },
  ];

  function platformName(): string {
    return ($route as { platformName?: string }).platformName ?? "";
  }

  function showSaveError(error: unknown): void {
    pushToast({
      type: "error",
      message: formatToastWithError(get(t)("Toast_SaveFailed"), error),
      duration: 8000,
    });
  }

  async function clearBackground(): Promise<void> {
    try {
      if (target === "app") {
        await PlatformService.ClearAppBackground();
        appBgInfo.set(await PlatformService.GetAppBackground());
        return;
      }
      const name = platformName();
      await PlatformService.ClearPlatformBackground(name);
      platformBgInfo.set(await PlatformService.GetPlatformBackground(name));
    } catch (e) {
      showSaveError(e);
    }
  }

  /** Slider moves are shown live and written once the user settles. */
  function onSlide(
    e: Event,
    field: "opacity" | "blur",
    save: (value: number) => Promise<void>,
  ): void {
    const value = parseFloat((e.currentTarget as HTMLInputElement).value);
    store.update((state) => ({ ...state, [field]: value }));
    const timer = field === "opacity" ? opacityTimer : blurTimer;
    clearTimeout(timer);
    const next = setTimeout(() => void save(value).catch(showSaveError), SAVE_DELAY_MS);
    if (field === "opacity") {
      opacityTimer = next;
    } else {
      blurTimer = next;
    }
  }

  function saveOpacity(value: number): Promise<void> {
    return target === "app"
      ? PlatformService.SetAppBackgroundOpacity(value)
      : PlatformService.SetPlatformBackgroundOpacity(platformName(), value);
  }

  function saveBlur(value: number): Promise<void> {
    return target === "app"
      ? PlatformService.SetAppBackgroundBlur(value)
      : PlatformService.SetPlatformBackgroundBlur(platformName(), value);
  }

  async function pickAlignment(value: string): Promise<void> {
    const next = normalizeBackgroundAlignment(value);
    const previous = alignment;
    store.update((state) => ({ ...state, alignment: next }));
    try {
      if (target === "app") {
        await PlatformService.SetAppBackgroundAlignment(next);
      } else {
        await PlatformService.SetPlatformBackgroundAlignment(platformName(), next);
      }
    } catch (e) {
      store.update((state) => ({ ...state, alignment: previous }));
      showSaveError(e);
    }
  }

  async function pickFit(value: string): Promise<void> {
    const next = normalizeBackgroundFit(value);
    const previous = fit;
    store.update((state) => ({ ...state, fit: next }));
    try {
      if (target === "app") {
        await PlatformService.SetAppBackgroundFit(next);
      } else {
        await PlatformService.SetPlatformBackgroundFit(platformName(), next);
      }
    } catch (e) {
      store.update((state) => ({ ...state, fit: previous }));
      showSaveError(e);
    }
  }

  $: clearLabel =
    target === "app" ? $t("Settings_ClearBackground") : $t("Settings_ClearPlatformBackground");
  $: rowLabel =
    target === "app" ? $t("Settings_BackgroundImage") : $t("Settings_BackgroundImage_Platform");
</script>

<SettingsRow label={rowLabel} keywords="background image wallpaper">
  {#if onResetToTheme}
    <button type="button" class="settings-btn settings-btn--ghost" on:click={onResetToTheme}>
      {$t("Settings_ResetToThemeBackground")}
    </button>
  {/if}
  <button type="button" class="settings-btn" on:click={() => void clearBackground()}>
    {clearLabel}
  </button>
</SettingsRow>

<SettingsRow
  label={$t("Settings_BgAlignAria")}
  controlId={`bg-align-${target}`}
  keywords="background position"
>
  <SettingsSelect
    id={`bg-align-${target}`}
    label={$t("Settings_BgAlignAria")}
    options={alignmentOptions}
    value={alignment}
    on:select={(e) => void pickAlignment(e.detail)}
  />
</SettingsRow>

<SettingsRow
  label={$t("Settings_BgFitAria")}
  controlId={`bg-fit-${target}`}
  keywords="background scale"
>
  <SettingsSelect
    id={`bg-fit-${target}`}
    label={$t("Settings_BgFitAria")}
    options={fitOptions}
    value={fit}
    on:select={(e) => void pickFit(e.detail)}
  />
</SettingsRow>

<SettingsRow
  label={$t("Settings_BgOpacity")}
  controlId={`bg-opacity-${target}`}
  stacked
  keywords="background transparency"
>
  <input
    id={`bg-opacity-${target}`}
    class="settings-slider"
    type="range"
    min="0"
    max="1"
    step="0.01"
    value={info.opacity}
    on:input={(e) => onSlide(e, "opacity", saveOpacity)}
  />
  <span class="settings-value">{Math.round(info.opacity * 100)}%</span>
</SettingsRow>

<SettingsRow
  label={$t("Settings_BgBlur")}
  controlId={`bg-blur-${target}`}
  stacked
  keywords="background blur"
>
  <input
    id={`bg-blur-${target}`}
    class="settings-slider"
    type="range"
    min="0"
    max="40"
    step="0.5"
    value={info.blur}
    on:input={(e) => onSlide(e, "blur", saveBlur)}
  />
  <span class="settings-value">{info.blur.toFixed(1)}px</span>
</SettingsRow>
