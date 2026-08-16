<script lang="ts">
  export let label = "";
  export let onClick: (ev: Event) => void = () => {};
  export let disabled = false;
</script>

<!-- Funnel and label sit together at the grid's left edge: the state readout
     and the control that changes it must not live at opposite ends of the bar. -->
<button type="button" class="tag-filter-bar" class:tag-filter-bar--disabled={disabled} on:click={disabled ? undefined : onClick} disabled={disabled}>
  <span class="tag-filter-bar__icon" aria-hidden="true">
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width="18"
      height="18"
      fill="currentColor"
    >
      <path d="M10 18h4v-2h-4v2zM3 6v2h18V6H3zm3 7h12v-2H6v2z" />
    </svg>
  </span>
  <span class="tag-filter-bar__label">{label}</span>
</button>

<style lang="scss">
  .tag-filter-bar {
    display: flex;
    align-items: center;
    justify-content: flex-start;
    gap: 0.35rem;
    width: 100%;
    margin: 0 0 0.375rem;
    padding: 0.3rem 0.375rem;
    border: 0px solid transparent;
    background: var(--backdrop-dark-20);
    color: inherit;
    font: inherit;
    cursor: pointer;
    box-sizing: border-box;
    &::active {
      border: 1px solid var(--overlay-white-12)!important;
    }
  }

  .tag-filter-bar--disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .tag-filter-bar__label {
    text-align: start;
    font-size: 0.6875rem;
  }

  .tag-filter-bar__icon {
    display: flex;
    flex-shrink: 0;
    opacity: 0.85;
  }
</style>
