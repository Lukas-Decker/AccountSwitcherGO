<script lang="ts">
  /*
    One checkbox setting on the shared settings row: label (and optional
    explanation) on the left, the control on the right, the same shape App
    settings uses. The explanation renders under the label rather than hiding
    in a hover tooltip, so it is read before the choice is made.
  */
  export let id: string;
  export let checked: boolean = false;
  export let disabled: boolean = false;
  export let label: string = "";
  export let tooltip: string = "";

  import SettingsRow from "./app/SettingsRow.svelte";
</script>

<SettingsRow {label} hint={tooltip} controlId={id} {disabled}>
  <span class="form-check">
    <input
      type="checkbox"
      {id}
      bind:checked
      {disabled}
      on:change
    />
    <label class="form-check-label" for={id}></label>
  </span>
</SettingsRow>
