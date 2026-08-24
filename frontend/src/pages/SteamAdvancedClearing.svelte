<script lang="ts">
  import { onMount } from "svelte";
  import { tick } from "svelte";
  import { get } from "svelte/store";
  import { route, previousPage, appBarTitle } from "../stores/nav";
  import { t } from "../stores/i18n";
  import { openExternalUrl } from "../lib/openExternalUrl";
  import { sanitizeHtml } from "../lib/sanitizeHtml";
  import { pushToast } from "../stores/toast";
  import { activeModal, openConfirm } from "../stores/modal";
  import { RunAdvancedClearingAction } from "../../bindings/account-switcher/internal/steam/steamservice.js";
  import "../styles/Settings.scss";

  const WIKI_URL =
    "https://github.com/KeinNameVorhanden/AccountSwitcherGO/wiki/Platform:-Steam#steam-cleaning";

  /** Action IDs match Go `RunAdvancedClearingAction`. */
  const generalActions = [
    "clear_logs",
    "clear_dumps",
    "clear_htmlcache",
    "clear_ui_logs",
    "clear_appcache",
    "clear_httpcache",
    "clear_depotcache",
  ];

  const loginActions = [
    "delete_loginusers_vdf",
    "clear_ssfn",
    "delete_config_vdf",
    "reg_autologinuser",
    "reg_lastgamenameused",
    "reg_pseudouuid",
    "reg_rememberpassword",
  ];

  /* The label leads with the outcome; the path is the second, smaller line.
     Spelled out per id rather than built from it so key greps find them. */
  const actionLabelKey: Record<string, string> = {
    clear_logs: "Cleaning_Action_clear_logs",
    clear_dumps: "Cleaning_Action_clear_dumps",
    clear_htmlcache: "Cleaning_Action_clear_htmlcache",
    clear_ui_logs: "Cleaning_Action_clear_ui_logs",
    clear_appcache: "Cleaning_Action_clear_appcache",
    clear_httpcache: "Cleaning_Action_clear_httpcache",
    clear_depotcache: "Cleaning_Action_clear_depotcache",
    delete_loginusers_vdf: "Cleaning_Action_delete_loginusers_vdf",
    clear_ssfn: "Cleaning_Action_clear_ssfn",
    delete_config_vdf: "Cleaning_Action_delete_config_vdf",
    reg_autologinuser: "Cleaning_Action_reg_autologinuser",
    reg_lastgamenameused: "Cleaning_Action_reg_lastgamenameused",
    reg_pseudouuid: "Cleaning_Action_reg_pseudouuid",
    reg_rememberpassword: "Cleaning_Action_reg_rememberpassword",
  };

  const pathText: Record<string, string> = {
    clear_logs: "..\\Steam\\logs",
    clear_dumps: "..\\Steam\\dumps",
    clear_htmlcache: "%Local%\\Steam\\htmlcache",
    clear_ui_logs: "..\\Steam\\*.log",
    clear_appcache: "..\\Steam\\appcache",
    clear_httpcache: "..\\Steam\\appcache\\httpcache",
    clear_depotcache: "..\\Steam\\depotcache",
    delete_loginusers_vdf: "..\\Steam\\config\\loginusers.vdf",
    clear_ssfn: "..\\Steam\\ssfn*",
    delete_config_vdf: "..\\Steam\\config\\config.vdf",
    reg_autologinuser: "HKCU\\..\\AutoLoginUser",
    reg_lastgamenameused: "HKCU\\..\\LastGameNameUsed",
    reg_pseudouuid: "HKCU\\..\\PseudoUUID",
    reg_rememberpassword: "HKCU\\..\\RememberPassword",
  };

  function isRegistryAction(id: string): boolean {
    return id.startsWith("reg_");
  }

  let acceptedRisk = false;
  let registrySupported = false;
  let busy = false;
  let logLines: string[] = [];
  let logEl: HTMLDivElement | null = null;

  $: appBarTitle.set($t("Title_Steam_Cleaning"));

  const i18nLogPrefix = "i18n:";
  const i18nLogSep = "\u001f";

  function isWindowsClient(): boolean {
    if (typeof navigator === "undefined") return false;
    const uaData = (navigator as { userAgentData?: { platform?: string } }).userAgentData;
    const platform = (uaData?.platform || navigator.platform || navigator.userAgent || "").toLowerCase();
    return platform.includes("win");
  }

  onMount(() => {
    previousPage.set({ page: "platform-settings", platformName: "Steam" });
    // UI visibility is client-platform based; backend still enforces OS support.
    registrySupported = isWindowsClient();
  });

  function showAction(_id: string): boolean {
    return true;
  }

  async function scrollLogToBottom(): Promise<void> {
    await tick();
    if (logEl) {
      logEl.scrollTop = logEl.scrollHeight;
    }
  }

  function translateLogLine(line: string): string {
    if (!line.startsWith(i18nLogPrefix)) {
      return line;
    }
    const parts = line.slice(i18nLogPrefix.length).split(i18nLogSep);
    const key = parts.shift() ?? "";
    const vars: Record<string, string | number> = {};
    for (let i = 0; i < parts.length; i += 2) {
      const name = parts[i];
      if (!name) continue;
      vars[name] = parts[i + 1] ?? "";
    }
    return get(t)(key, vars);
  }

  async function runAction(id: string): Promise<void> {
    if (!acceptedRisk || busy) return;
    // The one action that empties the switcher itself gets its warning at the
    // moment of deciding, not only in the paragraph at the top of the page.
    if (id === "delete_loginusers_vdf") {
      const ok = await openConfirm({
        title: $t("Cleaning_ConfirmLoginusers_Title"),
        body: `<p>${$t("Cleaning_ConfirmLoginusers_Body")}</p>`,
        style: "yesno",
        positiveLabel: $t("Yes"),
        negativeLabel: $t("No"),
      });
      if (!ok) return;
    }
    busy = true;
    try {
      const res = await RunAdvancedClearingAction(id);
      const lines = res?.lines?.length
        ? res.lines.map(translateLogLine)
        : [$t("SteamAdvanced_NoOutput")];
      logLines = [...logLines, ...lines, ""];
      await scrollLogToBottom();
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      logLines = [...logLines, $t("SteamAdvanced_LogError", { message: msg }), ""];
      await scrollLogToBottom();
      pushToast({ type: "error", title: "", message: msg, duration: 10000 });
    } finally {
      busy = false;
    }
  }

  function clearLog(): void {
    logLines = [];
  }

  function onWiki(): void {
    void openExternalUrl(WIKI_URL);
  }

  function onClose(): void {
    route.set({ page: "platform-settings", platformName: "Steam" });
  }

  function onWindowKeyDown(e: KeyboardEvent): void {
    if (e.key !== "Escape") return;
    if (get(activeModal)) return;
    onClose();
  }
</script>

<div class="main-content main-spacing steam-adv-root">
  <!-- The window title already carries the app name; repeating it in the page
       broke the sentence-case pattern its sibling pages follow. -->
  <h1 class="SettingsHeader">{$t("Cleaning_PageTitle")}</h1>

  <h2 class="SettingsHeader">{$t("Cleaning_ImportantInfoHeader")}</h2>
  <div class="steam-adv-info">
    <!-- eslint-disable-next-line svelte/no-at-html-tags -->
    {@html sanitizeHtml($t("Cleaning_ImportantInfo"), "inline")}
    <!-- Reference material sits with the warning, before the decisions,
         not below all thirteen of them. -->
    <button type="button" class="fancyLinkBtn steam-adv-wiki" on:click={onWiki}>{$t("Button_WikiInfo")}</button>
  </div>

  <div class="rowSetting steam-adv-ack">
    <div class="form-check">
      <input id="steam-adv-understand" type="checkbox" bind:checked={acceptedRisk} />
      <label class="form-check-label" for="steam-adv-understand"></label>
    </div>
    <span class="steam-adv-ack-text">{$t("Cleaning_Understand")}</span>
  </div>

  <button
    type="button"
    class="danger steam-adv-kill"
    disabled={!acceptedRisk || busy}
    on:click={() => void runAction("close_steam")}
  >
    {$t("Cleaning_Button_KillProcess", { platform: "Steam" })}
  </button>

  <h2 class="SettingsHeader">{$t("Cleaning_Header_General")}</h2>
  <div class="steam-adv-grid">
    {#each generalActions as id}
      {#if showAction(id)}
        <button
          type="button"
          class="danger steam-adv-action"
          disabled={!acceptedRisk || busy || (isRegistryAction(id) && !registrySupported)}
          on:click={() => void runAction(id)}
        >
          <span class="steam-adv-action__label">{$t(actionLabelKey[id])}</span>
          <span class="steam-adv-action__path">{pathText[id]}</span>
        </button>
      {/if}
    {/each}
  </div>

  <h2 class="SettingsHeader">{$t("Cleaning_Header_LoginHistory")}</h2>
  <div class="steam-adv-grid">
    {#each loginActions as id}
      {#if showAction(id)}
        <button
          type="button"
          class="danger steam-adv-action"
          disabled={!acceptedRisk || busy || (isRegistryAction(id) && !registrySupported)}
          on:click={() => void runAction(id)}
        >
          <span class="steam-adv-action__label">{$t(actionLabelKey[id])}</span>
          <span class="steam-adv-action__path">{pathText[id]}</span>
        </button>
      {/if}
    {/each}
  </div>

  <div class="steam-adv-log-wrap">
    <div class="steam-adv-log" bind:this={logEl} role="log" aria-live="polite">
      {#if logLines.length === 0}
        <p class="steam-adv-log-empty">{$t("SteamAdvanced_LogPlaceholder")}</p>
      {:else}
        {#each logLines as line, i (i)}
          <div class="steam-adv-log-line">{line || "\u00a0"}</div>
        {/each}
      {/if}
    </div>
    <div class="steam-adv-log-actions">
      <button type="button" class="steam-adv-clear-log" on:click={clearLog}>{$t("Button_ClearLog")}</button>
    </div>
  </div>

  <div class="buttoncol col_close steam-adv-footer">
    <button type="button" class="btn_close" on:click={onClose}><span>{$t("Button_Close")}</span></button>
  </div>
</div>

<svelte:window on:keydown={onWindowKeyDown} />

<style lang="scss">
  .steam-adv-root {
    overflow-y: auto;
    flex: 1;
    min-height: 0;
    padding-bottom: 0.75rem;
  }

  .steam-adv-info {
    color: var(--text-white-90);
    font-size: 0.7125rem;
    line-height: 1.45;
    margin-bottom: 0.5625rem;
  }

  .steam-adv-ack {
    margin: 0.375rem 0 0.75rem;
    align-items: flex-start;
  }

  .steam-adv-ack-text {
    padding: 0 0.5em;
    line-height: 1.35;
  }

  /* No .buttoncol here, so these buttons carry their own disabled treatment. */
  .steam-adv-kill:disabled,
  .steam-adv-action:disabled {
    opacity: 0.5;
    cursor: default;
  }

  /* Full-width action grid: three or four across on a laptop, one column at
     the app's minimum width, no empty right-hand gutter. */
  .steam-adv-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(15.75rem, 1fr));
    gap: 0.375rem;
    margin-bottom: 0.5625rem;
  }

  .steam-adv-action {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.075rem;
    width: 100%;
    margin: 0;
    padding: 0.45em 0.75em;
    text-align: start;

    /* The global 2.5em button-span rule is a one-line centring device; these
       labels are two lines and set their own rhythm. */
    span {
      line-height: 1.35;
    }
  }

  .steam-adv-action__label {
    font-weight: 500;
  }

  .steam-adv-action__path {
    font-family: ui-monospace, monospace;
    font-size: 0.585rem;
    opacity: 0.65;
  }

  .steam-adv-log-wrap {
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
    width: 100%;
    margin-top: 0.75rem;
  }

  .steam-adv-log {
    flex: 1;
    min-height: 6rem;
    max-height: min(40vh, 16.5rem);
    overflow-y: auto;
    padding: 0.45rem 0.5625rem;
    background: var(--backdrop-dark-35);
    border: 1px solid var(--accent);
    border-radius: 4px;
    font-family: ui-monospace, monospace;
    font-size: 0.6rem;
    line-height: 1.35;
    color: var(--text-white-92);
  }

  .steam-adv-log-empty {
    margin: 0;
    opacity: 0.65;
    font-style: italic;
  }

  .steam-adv-log-line {
    white-space: pre-wrap;
    word-break: break-word;
  }

  .steam-adv-log-actions {
    display: flex;
    justify-content: flex-end;
  }

  .steam-adv-clear-log {
    position: relative;
    width: auto;
  }

  .fancyLinkBtn {
    position: relative;
    width: auto !important;
    background: transparent;
    border: none;
    color: var(--accent);
    text-decoration: underline;
    cursor: pointer;
    font: inherit;
    padding: 0.2625rem 0.375rem;
  }

  .fancyLinkBtn:hover {
    filter: brightness(1.15);
  }

  .steam-adv-wiki {
    display: block;
    margin-top: 0.35rem;
    padding-left: 0;
  }

  .steam-adv-footer {
    margin-top: 1.125rem;
    flex-wrap: wrap;
    justify-content: flex-end;
    align-items: center;
  }
</style>
