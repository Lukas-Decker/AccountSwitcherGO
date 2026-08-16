<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { t } from "../../stores/i18n";
  import { viewportDropdown } from "../../lib/actions/viewportDropdown";
  import SettingsRow from "./app/SettingsRow.svelte";

  export let values: readonly string[];
  export let current: string;
  export let label: string = "";
  export let labelFn: (v: string) => string = (v) => v;
  export let tooltip: string = "";
  export let disabled: boolean = false;

  const dispatch = createEventDispatcher();
  let open = false;

  // A dropdown that displays nothing gives no sign it is one, so an unset
  // value renders as an explicit "Default" instead of a bare caret.
  $: currentLabel = current ? labelFn(current) : $t("Settings_MethodNotSet");

  function toggle(): void {
    if (!disabled) open = !open;
  }

  function select(value: string): void {
    if (disabled) return;
    dispatch("select", { value });
    open = false;
  }
</script>

<SettingsRow {label} hint={tooltip} {disabled}>
  <div class="dropdown" class:show={open}>
    <button type="button" class="dropdown-toggle" on:click={toggle}>
      {currentLabel}
      <span class="caret" aria-hidden="true"></span>
    </button>
    {#if open}
      <ul class="custom-dropdown-menu dropdown-menu" use:viewportDropdown>
        {#each values as v}
          <li>
            <button type="button" class="dropdown-item" on:click={() => select(v)}>
              {labelFn(v)}
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</SettingsRow>
