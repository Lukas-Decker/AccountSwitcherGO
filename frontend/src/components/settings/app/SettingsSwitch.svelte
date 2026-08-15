<script lang="ts">
  /*
    On/off control for a boolean setting.

    A real button with role="switch" rather than a hidden checkbox: it is
    reachable by keyboard and by the controller spatial navigation, which looks
    for [role='switch'] among other things.
  */
  import type { SettingToggle } from "../../../lib/appSettingsModel";

  export let toggle: SettingToggle;
  export let id: string;
  /** Read out by screen readers, since the visible label lives in the row. */
  export let label: string;
  /** Blocks the switch, e.g. Discord presence while offline mode is on. */
  export let disabled = false;

  $: checked = $toggle.value;
  $: busy = $toggle.busy;
</script>

<button
  {id}
  type="button"
  role="switch"
  class="settings-switch"
  class:settings-switch--busy={busy}
  aria-checked={checked}
  aria-label={label}
  disabled={disabled || busy}
  on:click={() => void toggle.apply(!checked)}
>
  <span class="settings-switch__knob"></span>
</button>
