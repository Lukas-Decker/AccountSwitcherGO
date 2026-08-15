<script lang="ts">
  /*
    Dropdown used for theme, accent and language.

    A native <select> cannot show a colour swatch next to an option, so this is
    a button and a list. It flips above the trigger when there is more room up
    there, which matters for the cards near the bottom of the page.
  */
  import { createEventDispatcher, onDestroy } from "svelte";

  /** `color` draws a swatch before the label, used by the accent picker. */
  type SelectOption = { value: string; label: string; color?: string };

  export let id: string;
  export let options: SelectOption[] = [];
  export let value = "";
  export let label: string;
  /** Shown on the trigger when the value is not one of the options. */
  export let fallbackLabel = "";
  /** Swatch on the trigger, for the accent picker. */
  export let triggerColor = "";

  const dispatch = createEventDispatcher<{ select: string }>();

  const VIEWPORT_MARGIN = 8;
  const MAX_HEIGHT_RATIO = 0.6;

  let open = false;
  let root: HTMLElement;
  let menu: HTMLElement | undefined;
  let placement: "below" | "above" = "below";
  let maxHeight = 0;

  $: current = options.find((option) => option.value === value);
  $: triggerLabel = current?.label || fallbackLabel || value;
  $: swatch = triggerColor || current?.color || "";

  /** Puts the menu wherever there is room, and caps it to what is left. */
  function place(): void {
    if (!open || !root || !menu) {
      return;
    }
    const trigger = root.getBoundingClientRect();
    const viewport = window.innerHeight || document.documentElement.clientHeight;
    const cap = Math.floor(viewport * MAX_HEIGHT_RATIO);
    const above = Math.max(0, Math.floor(trigger.top - VIEWPORT_MARGIN));
    const below = Math.max(0, Math.floor(viewport - trigger.bottom - VIEWPORT_MARGIN));
    const wanted = Math.min(menu.scrollHeight, cap);
    placement = wanted <= below || below >= above ? "below" : "above";
    maxHeight = Math.min(cap, placement === "below" ? below : above);
  }

  function toggleOpen(): void {
    open = !open;
    if (open) {
      requestAnimationFrame(place);
    }
  }

  function pick(option: SelectOption): void {
    open = false;
    dispatch("select", option.value);
  }

  function onWindowPointerDown(e: PointerEvent): void {
    if (!open || !root) {
      return;
    }
    if (!root.contains(e.target as Node)) {
      open = false;
    }
  }

  function onWindowKeyDown(e: KeyboardEvent): void {
    if (open && e.key === "Escape") {
      e.stopPropagation();
      open = false;
    }
  }

  onDestroy(() => {
    open = false;
  });
</script>

<svelte:window
  on:pointerdown={onWindowPointerDown}
  on:keydown={onWindowKeyDown}
  on:resize={place}
  on:scroll|capture={place}
/>

<div class="settings-select" class:is-open={open} bind:this={root}>
  <button
    {id}
    type="button"
    class="settings-select__trigger"
    aria-haspopup="listbox"
    aria-expanded={open}
    aria-label={label}
    on:click={toggleOpen}
  >
    {#if swatch}
      <span class="settings-swatch" style={`--settings-swatch-color: ${swatch}`}></span>
    {/if}
    <span class="settings-select__label">{triggerLabel}</span>
    <span class="settings-select__caret" aria-hidden="true"></span>
  </button>

  {#if open}
    <ul
      class="settings-select__menu"
      role="listbox"
      aria-label={label}
      bind:this={menu}
      style={`${placement === "below" ? "top" : "bottom"}: calc(100% + 2px); ${
        placement === "below" ? "bottom" : "top"
      }: auto; max-height: ${maxHeight}px;`}
    >
      {#each options as option (option.value)}
        <li role="none">
          <button
            type="button"
            role="option"
            aria-selected={option.value === value}
            class="settings-select__option"
            class:is-active={option.value === value}
            on:click={() => pick(option)}
          >
            {#if option.color}
              <span class="settings-swatch" style={`--settings-swatch-color: ${option.color}`}></span>
            {/if}
            <span>{option.label}</span>
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</div>
