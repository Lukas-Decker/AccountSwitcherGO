<script lang="ts">
  import { get } from "svelte/store";
  import { onMount } from "svelte";
  import { previousPage, appBarTitle, navigateBackLikeButton } from "../stores/nav";
  import { activeModal } from "../stores/modal";
  import { t } from "../stores/i18n";
  import { controllerSpatialNavigation } from "../lib/actions/controllerSpatialNavigation";

  $: appBarTitle.set($t("Title_Settings"));
  onMount(() => {
    previousPage.set({ page: "home" });
  });

  function onWindowKeyDown(e: KeyboardEvent): void {
    if (e.key !== "Escape") {
      return;
    }
    if (get(activeModal)) {
      return;
    }
    // The search field clears itself on Escape and stops the event there.
    e.preventDefault();
    navigateBackLikeButton();
  }
</script>

<div class="main-content main-spacing" use:controllerSpatialNavigation>
  {#await import("../components/settings/app/AppSettingsGrid.svelte") then { default: AppSettingsGrid }}
    <AppSettingsGrid />
  {/await}
</div>
<svelte:window on:keydown={onWindowKeyDown} />
