<script lang="ts">
  import { t } from "../../stores/i18n";
  import { route } from "../../stores/nav";

  export let name: string;
  /** Reflects the stored toggle so the button says which card is in use. */
  export let overrideEnabled = false;
</script>

<h2 class="SettingsHeader">{$t("CardEditor_Title")}</h2>

<div class="cardsection">
  <p class="cardsection__state">
    {overrideEnabled ? $t("CardEditor_State_Own") : $t("CardEditor_State_Global")}
  </p>
  <button
    type="button"
    class="cardsection__open"
    on:click={() => route.set({ page: "card-editor", platformName: name })}
  >
    {$t("CardEditor_Open")}
  </button>
</div>

<style lang="scss">
  .cardsection {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 0.75rem;
  }

  .cardsection__state {
    margin: 0;
    font-size: 0.85rem;
    opacity: 0.75;
  }

  .cardsection__open {
    padding: 0.35rem 0.9rem;
    border: 1px solid var(--role-field-border, var(--button-bg));
    border-radius: 3px;
    background: var(--button-bg);
    color: inherit;
    font: inherit;
    cursor: pointer;

    &:hover {
      background: var(--button-bg-hover, var(--button-bg));
    }
  }
</style>
