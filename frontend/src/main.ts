import App from './App.svelte'
import './styles/context_menu.scss'
import './styles/normalize.scss'
import './styles/style.scss'
import './styles/theme.scss'
import './styles/overlayReceivers.scss'
import './styles/UI.scss'
import './styles/modal-primary.scss'
import './styles/acclist.scss'
import './styles/rounded.scss'
import './styles/rtl.scss'
import { initI18n } from './stores/i18n'
import { initOfflineMode } from './stores/offlineMode'
import { resolveInitialRoute, installHashSync } from './stores/nav'
import { initTheme } from './lib/themes'
import { initRoundedCorners } from './stores/roundedCorners'

const app = void (async () => {
  // Lets the UI be driven in a plain browser, where Wails' native IPC is not
  // available. Opt-in via ?wailsStub, and compiled out of production builds.
  if (import.meta.env.DEV && new URLSearchParams(location.search).has('wailsStub')) {
    const { installWailsStub, applyDevCardPreset } = await import('./dev/wailsStub')
    installWailsStub()
    await applyDevCardPreset()
  }

  await initI18n()
  await initOfflineMode()
  await initTheme()
  await initRoundedCorners()
  await resolveInitialRoute()
  installHashSync()
  new App({ target: document.getElementById('app')! })
})()

export default app
