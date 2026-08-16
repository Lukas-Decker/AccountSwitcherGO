<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import AccountCardPreview from "./AccountCardPreview.svelte";
  import { blockDef } from "./blockRegistry";
  import { t } from "../../stores/i18n";
  import { presetLayout, seedCustomLayout } from "../../lib/accountCard/presets";
  import { resolveLayout } from "../../lib/accountCard/resolve";
  import {
    flatToLayout,
    layoutToFlat,
    moveBlock,
    type FlatBlock,
  } from "../../lib/accountCard/flatLayout";
  import {
    CARD_BOUNDS,
    type AccountCardConfig,
    type CardBlockDisplay,
    type CardBlockKind,
    type CardSizePreset,
    type StatusBadgeStyle,
  } from "../../lib/accountCard/types";

  export let config: AccountCardConfig;
  /** What the platform being configured can draw. */
  export let available: readonly CardBlockKind[];
  export let disabled = false;

  const dispatch = createEventDispatcher<{ change: AccountCardConfig }>();

  const PRESETS: CardSizePreset[] = ["small", "medium", "large", "custom"];
  const BADGE_STYLES: StatusBadgeStyle[] = ["border", "corner", "block"];

  $: isCustom = config.preset === "custom";
  $: activeLayout = resolveLayout(config, available);

  // The blocks as the editor lists them: whatever the active shape places,
  // followed by anything the platform offers that it does not.
  $: baseLayout = isCustom
    ? (config.custom ?? seedCustomLayout("small"))
    : presetLayout(config.preset);
  $: flat = withOverrides(layoutToFlat(baseLayout, available));

  function withOverrides(list: FlatBlock[]): FlatBlock[] {
    return list.map((b) => ({
      ...b,
      enabled: config.blocks?.[b.kind] ?? b.enabled,
      display: config.displays?.[b.kind] ?? b.display,
    }));
  }

  function emit(next: AccountCardConfig): void {
    dispatch("change", next);
  }

  function choosePreset(preset: CardSizePreset): void {
    if (preset === "custom" && !config.custom) {
      // Start from what they were already looking at, rather than a blank card.
      const from = config.preset === "custom" ? "small" : config.preset;
      emit({ ...config, preset, custom: seedCustomLayout(from) });
      return;
    }
    emit({ ...config, preset });
  }

  function toggleBlock(kind: CardBlockKind, enabled: boolean): void {
    emit({ ...config, blocks: { ...(config.blocks ?? {}), [kind]: enabled } });
  }

  function setDisplay(kind: CardBlockKind, display: string): void {
    emit({ ...config, displays: { ...(config.displays ?? {}), [kind]: display as CardBlockDisplay } });
  }

  function setBadgeStyle(style: StatusBadgeStyle): void {
    emit({ ...config, statusBadgeStyle: style });
  }

  /** Ordering, joining and dimensions only exist for the custom shape. */
  function updateCustom(next: FlatBlock[]): void {
    emit({ ...config, custom: flatToLayout(next, baseLayout) });
  }

  function move(index: number, delta: -1 | 1): void {
    updateCustom(moveBlock(flat, index, delta));
  }

  function toggleJoin(index: number, join: boolean): void {
    const next = flat.map((b, i) => (i === index ? { ...b, joinPrevious: join } : { ...b }));
    updateCustom(next);
  }

  function setMetric(key: "minWidth" | "maxWidth" | "minHeight" | "avatarEm" | "fontScale", value: number): void {
    emit({ ...config, custom: { ...baseLayout, [key]: value } });
  }

  function presetLabel(preset: CardSizePreset): string {
    return $t(`CardSize_${preset.charAt(0).toUpperCase()}${preset.slice(1)}`);
  }
</script>

<div class="cardeditor" class:cardeditor--disabled={disabled}>
  <fieldset class="cardeditor__sizes" {disabled}>
    <legend class="cardeditor__legend">{$t("CardEditor_Size")}</legend>
    <div class="cardeditor__presets" role="radiogroup" aria-label={$t("CardEditor_Size")}>
      {#each PRESETS as preset (preset)}
        <button
          type="button"
          role="radio"
          aria-checked={config.preset === preset}
          class="cardeditor__preset"
          class:is-active={config.preset === preset}
          on:click={() => choosePreset(preset)}
        >
          {presetLabel(preset)}
        </button>
      {/each}
    </div>
    <p class="cardeditor__hint">
      {isCustom ? $t("CardEditor_Hint_Custom") : $t("CardEditor_Hint_Preset")}
    </p>
  </fieldset>

  <AccountCardPreview {config} {available} />

  <fieldset class="cardeditor__blocks" {disabled}>
    <legend class="cardeditor__legend">{$t("CardEditor_Blocks")}</legend>
    <ul class="cardeditor__list">
      {#each flat as block, index (block.kind)}
        {@const def = blockDef(block.kind)}
        <li class="cardeditor__row" class:is-joined={isCustom && block.joinPrevious}>
          <!-- The app draws checkboxes as the adjacent label; the input itself
               is transparent, so the pair has to stay together. -->
          <span class="cardeditor__toggle">
            <span class="form-check">
              <input
                type="checkbox"
                id={`cardblock-${block.kind}`}
                checked={block.enabled}
                on:change={(e) => toggleBlock(block.kind, e.currentTarget.checked)}
              />
              <label class="form-check-label" for={`cardblock-${block.kind}`}></label>
            </span>
            <label for={`cardblock-${block.kind}`}>{$t(def.labelKey)}</label>
          </span>

          {#if def.displays.length > 1}
            <select
              class="cardeditor__display"
              aria-label={$t("CardEditor_Display")}
              value={block.display ?? "text"}
              on:change={(e) => setDisplay(block.kind, e.currentTarget.value)}
            >
              {#each def.displays as mode (mode)}
                <option value={mode}>{$t(`CardDisplay_${mode}`)}</option>
              {/each}
            </select>
          {/if}

          {#if isCustom}
            <span class="cardeditor__join" title={$t("CardEditor_JoinLine")}>
              <span class="form-check">
                <input
                  type="checkbox"
                  id={`cardjoin-${block.kind}`}
                  checked={block.joinPrevious}
                  disabled={index === 0}
                  on:change={(e) => toggleJoin(index, e.currentTarget.checked)}
                />
                <label class="form-check-label" for={`cardjoin-${block.kind}`}></label>
              </span>
              <label for={`cardjoin-${block.kind}`}>{$t("CardEditor_JoinLine")}</label>
            </span>
            <span class="cardeditor__move">
              <button type="button" disabled={index === 0} on:click={() => move(index, -1)} aria-label={$t("Context_MoveLeft")}>&uarr;</button>
              <button type="button" disabled={index === flat.length - 1} on:click={() => move(index, 1)} aria-label={$t("Context_MoveRight")}>&darr;</button>
            </span>
          {/if}
        </li>
      {/each}
    </ul>
  </fieldset>

  <fieldset class="cardeditor__badges" {disabled}>
    <legend class="cardeditor__legend">{$t("CardEditor_Warnings")}</legend>
    <div class="cardeditor__presets" role="radiogroup" aria-label={$t("CardEditor_Warnings")}>
      {#each BADGE_STYLES as style (style)}
        <button
          type="button"
          role="radio"
          aria-checked={activeLayout.statusBadgeStyle === style}
          class="cardeditor__preset"
          class:is-active={activeLayout.statusBadgeStyle === style}
          on:click={() => setBadgeStyle(style)}
        >
          {$t(`CardBadges_${style}`)}
        </button>
      {/each}
    </div>
  </fieldset>

  {#if isCustom}
    <fieldset class="cardeditor__metrics" {disabled}>
      <legend class="cardeditor__legend">{$t("CardEditor_Dimensions")}</legend>
      <label>
        <span>{$t("CardEditor_MinWidth")}</span>
        <input
          type="number" min={CARD_BOUNDS.minWidth.min} max={CARD_BOUNDS.minWidth.max}
          value={baseLayout.minWidth}
          on:change={(e) => setMetric("minWidth", Number(e.currentTarget.value))}
        />
      </label>
      <label>
        <span>{$t("CardEditor_MaxWidth")}</span>
        <input
          type="number" min={CARD_BOUNDS.maxWidth.min} max={CARD_BOUNDS.maxWidth.max}
          value={baseLayout.maxWidth}
          on:change={(e) => setMetric("maxWidth", Number(e.currentTarget.value))}
        />
      </label>
      <label>
        <span>{$t("CardEditor_MinHeight")}</span>
        <input
          type="number" min={CARD_BOUNDS.minHeight.min} max={CARD_BOUNDS.minHeight.max}
          value={baseLayout.minHeight}
          on:change={(e) => setMetric("minHeight", Number(e.currentTarget.value))}
        />
      </label>
      <label>
        <span>{$t("CardEditor_AvatarSize")}</span>
        <input
          type="number" step="0.5" min={CARD_BOUNDS.avatarEm.min} max={CARD_BOUNDS.avatarEm.max}
          value={baseLayout.avatarEm}
          on:change={(e) => setMetric("avatarEm", Number(e.currentTarget.value))}
        />
      </label>
      <label>
        <span>{$t("CardEditor_TextScale")}</span>
        <input
          type="number" step="0.05" min={CARD_BOUNDS.fontScale.min} max={CARD_BOUNDS.fontScale.max}
          value={baseLayout.fontScale}
          on:change={(e) => setMetric("fontScale", Number(e.currentTarget.value))}
        />
      </label>
    </fieldset>
  {/if}
</div>

<style lang="scss">
  .cardeditor {
    display: flex;
    flex-direction: column;
    gap: 0.85rem;
  }

  .cardeditor--disabled {
    opacity: 0.55;
  }

  fieldset {
    min-width: 0;
    margin: 0;
    padding: 0;
    border: 0;
  }

  .cardeditor__legend {
    padding: 0;
    font-size: 0.85rem;
    font-weight: 600;
    letter-spacing: 0.03em;
    text-transform: uppercase;
    opacity: 0.7;
  }

  .cardeditor__presets {
    display: flex;
    flex-wrap: wrap;
    gap: 0.35rem;
    margin-top: 0.4rem;
  }

  .cardeditor__preset {
    padding: 0.35rem 0.9rem;
    border: 1px solid var(--role-field-border, var(--button-bg));
    border-radius: 3px;
    background: var(--button-bg);
    color: inherit;
    font: inherit;
    cursor: pointer;

    &:hover:not(:disabled) {
      background: var(--button-bg-hover, var(--button-bg));
    }

    &.is-active {
      border-color: var(--accent);
      background: var(--accent-fill-soft, var(--button-bg-hover));
    }
  }

  .cardeditor__hint {
    margin: 0.4rem 0 0;
    font-size: 0.8rem;
    opacity: 0.7;
  }

  .cardeditor__list {
    margin: 0.4rem 0 0;
    padding: 0;
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
  }

  .cardeditor__row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
    padding: 0.25rem 0.4rem;
    border-radius: 3px;

    &:hover {
      background: var(--surface-row-dark, rgba(255, 255, 255, 0.04));
    }
  }

  // Marks a block that shares a line with the one above it, so the list still
  // reads as rows rather than as a flat sequence.
  .cardeditor__row.is-joined {
    margin-left: 1.1rem;
    border-left: 2px solid var(--accent-overlay-border, var(--button-bg));
  }

  .cardeditor__toggle,
  .cardeditor__join {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;

    label:not(.form-check-label) {
      cursor: pointer;
    }
  }

  // The transparent input must not take up space beside the box it draws.
  .form-check {
    display: inline-flex;
    align-items: center;

    input[type="checkbox"] {
      position: absolute;
      width: 1px;
      height: 1px;
      opacity: 0;
    }
  }

  .cardeditor__toggle {
    flex: 1 1 8rem;
    min-width: 0;
  }

  .cardeditor__join {
    font-size: 0.78rem;
    opacity: 0.75;
  }

  .cardeditor__move button {
    padding: 0.1rem 0.45rem;
    border: 1px solid var(--role-field-border, var(--button-bg));
    border-radius: 3px;
    background: var(--button-bg);
    color: inherit;
    cursor: pointer;

    &:disabled {
      opacity: 0.4;
      cursor: default;
    }
  }

  .cardeditor__metrics {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr));
    gap: 0.5rem;

    label {
      display: flex;
      flex-direction: column;
      gap: 0.2rem;
      font-size: 0.82rem;
    }

    input {
      width: 100%;
      padding: 0.25rem 0.4rem;
    }
  }
</style>
