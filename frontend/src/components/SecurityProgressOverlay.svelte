<script lang="ts">
  import { t } from "../stores/i18n";
  import { securityProgressActive, securityProgressMessage } from "../stores/security";
</script>

{#if $securityProgressActive}
  <div class="security-progress-overlay">
    <div class="security-progress-panel">
      <div class="security-progress-spinner" aria-hidden="true"></div>
      <div>
        <h2>{$t("Security_Progress_Title")}</h2>
        <p>{$securityProgressMessage ? $t($securityProgressMessage) : $t("Security_Progress_Working")}</p>
      </div>
    </div>
  </div>
{/if}

<style lang="scss">
  .security-progress-overlay {
    position: fixed;
    inset: 0;
    z-index: 10000;
    display: grid;
    place-items: center;
    background: rgba(0, 0, 0, 0.62);
  }

  .security-progress-panel {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    width: min(460px, calc(100vw - 1.5rem));
    padding: 0.8625rem;
    border: 1px solid var(--input-number-border);
    background: var(--program-bg);
    box-shadow: 0 18px 50px rgba(0, 0, 0, 0.35);
  }

  .security-progress-panel h2 {
    margin: 0 0 0.1875rem;
    font-size: 0.7875rem;
  }

  .security-progress-panel p {
    margin: 0;
    color: var(--whiteSecondary);
  }

  .security-progress-spinner {
    width: 34px;
    height: 34px;
    border: 3px solid rgba(255, 255, 255, 0.22);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: security-spin 0.9s linear infinite;
    flex: 0 0 auto;
  }

  @keyframes security-spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>
