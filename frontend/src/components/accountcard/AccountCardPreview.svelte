<script lang="ts">
  import AccountCard from "./AccountCard.svelte";
  import { availableBlockKinds } from "./blockRegistry";
  import { previewAdapter, PREVIEW_ACCOUNT, type PreviewAccount } from "../../lib/accountCard/previewAdapter";
  import { colorCssVars, layoutCssVars, resolveLayout } from "../../lib/accountCard/resolve";
  import type { AccountCardConfig, CardBlockKind } from "../../lib/accountCard/types";

  export let config: AccountCardConfig;
  /** What the platform being configured can draw. */
  export let available: readonly CardBlockKind[];
  export let account: PreviewAccount = PREVIEW_ACCOUNT;

  $: adapter = previewAdapter(available, account);
  $: layout = resolveLayout(config, availableBlockKinds(adapter));

  // Scoped to this element rather than the document root: a preview must not
  // resize the real account list sitting behind the settings page.
  $: styleVars = Object.entries({ ...layoutCssVars(layout), ...colorCssVars(config) })
    .map(([k, v]) => `${k}: ${v}`)
    .join("; ");
</script>

<div class="acc-preview" style={styleVars} aria-hidden="true">
  <div class="acc-preview__stage">
    <div class="acc_list acc-preview__list">
      <div class="acc_list_item">
        <div class="acc_list_item_inner">
          <AccountCard
            acc={account}
            {adapter}
            {layout}
            rid="preview"
            radioId="acc-preview"
            labelId="acc-preview-label"
            descId="acc-preview-desc"
            contextMenuItems={() => []}
            onContextMenuOpen={() => {}}
            onDragOver={() => {}}
            onDragLeave={() => {}}
            onActivate={() => {}}
          />
        </div>
      </div>
    </div>
  </div>
</div>

<style lang="scss">
  .acc-preview {
    display: flex;
    justify-content: center;
    padding: 0.75rem;
    border: 1px solid var(--role-field-border, var(--button-bg));
    border-radius: 4px;
    background: var(--mainContentBackground, var(--code-background));
    overflow-x: auto;
    // Sized to the card it shows rather than the page it sits on, and sticky
    // so the feedback loop survives scrolling the block list it reflects.
    width: fit-content;
    min-width: 14rem;
    max-width: 100%;
    align-self: center;
    position: sticky;
    top: 0.25rem;
    z-index: 5;
  }

  // The list is a grid in the real page; here it only ever holds one card, so
  // it is sized to that card rather than filling the settings panel.
  .acc-preview__list {
    width: auto;
    height: auto;
    grid-template-columns: var(--acc-card-max-w);
    overflow: visible;
  }

  .acc-preview__stage {
    pointer-events: none;
  }
</style>
