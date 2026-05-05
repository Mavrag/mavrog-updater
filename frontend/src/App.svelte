<script lang="ts">
  import { onMount } from 'svelte';
  import { marked } from 'marked';
  import appIcon from './assets/images/appicon.png';
  import {
    GetStatus,
    CheckForUpdate,
    InstallUpdate,
    SetAutoCheck,
    PickAddonsFolder,
    OpenURL,
    OpenAddonsFolder,
    CheckSelfUpdate,
    ApplySelfUpdate,
  } from '../wailsjs/go/main/App';
  import { EventsOn } from '../wailsjs/runtime/runtime';
  import type { main } from '../wailsjs/go/models';

  let status: main.AppStatus | null = null;
  let update: main.UpdateInfo | null = null;
  let selfUpdate: main.SelfUpdateInfo | null = null;
  let checking = false;
  let installing = false;
  let progress = 0;
  let logLines: string[] = [];
  let errorMsg = '';
  let view: 'home' | 'changelog' | 'settings' = 'home';
  let autoCheck = true;

  function pushLog(msg: string) {
    logLines = [...logLines.slice(-50), msg];
  }

  async function refreshStatus() {
    try {
      status = await GetStatus();
      autoCheck = status.autoCheck;
    } catch (e: any) {
      errorMsg = String(e);
    }
  }

  async function check() {
    if (checking) return;
    checking = true;
    errorMsg = '';
    try {
      update = await CheckForUpdate();
    } catch (e: any) {
      errorMsg = String(e?.message ?? e);
    } finally {
      checking = false;
    }
  }

  async function install() {
    if (installing) return;
    installing = true;
    errorMsg = '';
    progress = 0;
    try {
      const v = await InstallUpdate();
      pushLog(`Installed version ${v}`);
      await refreshStatus();
      await check();
    } catch (e: any) {
      errorMsg = String(e?.message ?? e);
    } finally {
      installing = false;
    }
  }

  async function pickFolder() {
    try {
      const p = await PickAddonsFolder();
      if (p) await refreshStatus();
    } catch (e: any) {
      errorMsg = String(e?.message ?? e);
    }
  }

  async function toggleAuto() {
    autoCheck = !autoCheck;
    await SetAutoCheck(autoCheck);
  }

  async function checkSelf() {
    try {
      selfUpdate = await CheckSelfUpdate();
    } catch (e) {
      // Silent: updater repo may not yet have releases.
      selfUpdate = null;
    }
  }

  async function applySelf() {
    if (!selfUpdate?.updateAvailable) return;
    try {
      await ApplySelfUpdate();
    } catch (e: any) {
      errorMsg = String(e?.message ?? e);
    }
  }

  marked.setOptions({ gfm: true, breaks: true });

  function renderMd(src: string): string {
    if (!src) return '';
    return marked.parse(src) as string;
  }

  function fmtBytes(n: number): string {
    if (!n) return '';
    if (n < 1024) return n + ' B';
    if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB';
    return (n / 1024 / 1024).toFixed(2) + ' MB';
  }

  onMount(async () => {
    EventsOn('install:progress', (p: any) => {
      progress = Math.max(0, Math.min(100, p?.percent ?? 0));
    });
    EventsOn('selfupdate:progress', (p: any) => {
      progress = Math.max(0, Math.min(100, p?.percent ?? 0));
    });
    EventsOn('log', (msg: string) => pushLog(msg));

    await refreshStatus();
    if (status?.autoCheck) {
      await check();
    }
    checkSelf();
  });
</script>

<main>
  <header>
    <div class="brand">
      <img class="logo" src={appIcon} alt="Mavrog" draggable="false" />
      <div>
        <div class="title">Mavrog Updater</div>
        <div class="subtitle">{status?.addonName ?? '...'}</div>
      </div>
    </div>
    <nav>
      <button class="ghost" class:active={view === 'home'} on:click={() => view = 'home'}>Home</button>
      <button class="ghost" class:active={view === 'changelog'} on:click={() => view = 'changelog'}>Changelog</button>
      <button class="ghost" class:active={view === 'settings'} on:click={() => view = 'settings'}>Settings</button>
    </nav>
  </header>

  {#if errorMsg}
    <div class="banner error">
      <span>⚠ {errorMsg}</span>
      <button class="ghost" on:click={() => errorMsg = ''}>Dismiss</button>
    </div>
  {/if}

  {#if selfUpdate?.updateAvailable}
    <div class="banner info">
      <span>Updater {selfUpdate.latestVersion} is available (you have {selfUpdate.currentVersion}).</span>
      <button class="primary" on:click={applySelf}>Update Updater</button>
    </div>
  {/if}

  {#if view === 'home'}
    <section class="hero">
      <div class="versions">
        <div class="vbox">
          <div class="label">Installed</div>
          <div class="value">{status?.installedVersion || '— not installed —'}</div>
        </div>
        <div class="arrow">→</div>
        <div class="vbox">
          <div class="label">Latest</div>
          <div class="value latest">
            {checking ? 'checking...' : (update?.latestVersion || '?')}
          </div>
          {#if update?.publishedAt}
            <div class="muted small">{update.publishedAt}</div>
          {/if}
        </div>
      </div>

      <div class="actions">
        <button on:click={check} disabled={checking || installing}>
          {checking ? 'Checking...' : 'Check for Updates'}
        </button>
        <button class="primary" on:click={install}
          disabled={installing || checking || !update?.updateAvailable || !status?.addonsPath}>
          {#if installing}
            Installing... {progress > 0 ? progress.toFixed(0) + '%' : ''}
          {:else if update?.updateAvailable}
            Install {update.latestVersion}
          {:else if update && !update.hasAsset}
            No release asset
          {:else}
            Up to date
          {/if}
        </button>
      </div>

      {#if installing && progress > 0}
        <div class="progress">
          <div class="bar" style="width: {progress}%"></div>
        </div>
      {/if}

      {#if update?.assetName}
        <div class="muted small center">
          Asset: {update.assetName} {update.assetSize ? '(' + fmtBytes(update.assetSize) + ')' : ''}
        </div>
      {/if}

      {#if !status?.addonsPath}
        <div class="warn">
          ⚠ AddOns folder not found. <button class="ghost" on:click={pickFolder}>Pick folder</button>
        </div>
      {/if}

      {#if logLines.length}
        <div class="log">
          {#each logLines as l}<div>{l}</div>{/each}
        </div>
      {/if}
    </section>
  {:else if view === 'changelog'}
    <section class="changelog">
      {#if !update}
        <p class="muted">Run a check first to see release notes.</p>
        <button on:click={check}>Check Now</button>
      {:else}
        <h2>{update.releaseName || update.latestVersion}</h2>
        <div class="muted small">{update.publishedAt}</div>
        <div class="md">{@html renderMd(update.changelog || '_(no notes provided)_')}</div>
        {#if update.htmlUrl}
          <button class="ghost" on:click={() => update && OpenURL(update.htmlUrl)}>Open on GitHub ↗</button>
        {/if}
      {/if}
    </section>
  {:else}
    <section class="settings">
      <div class="row">
        <div>
          <div class="label">AddOns Folder</div>
          <div class="path">{status?.addonsPath || '(not set)'}</div>
        </div>
        <div class="row-actions">
          <button on:click={pickFolder}>Change</button>
          <button class="ghost" on:click={OpenAddonsFolder} disabled={!status?.addonsPath}>Open</button>
        </div>
      </div>
      <div class="row">
        <div>
          <div class="label">Auto-check on launch</div>
          <div class="muted small">Automatically check GitHub when the app starts.</div>
        </div>
        <button on:click={toggleAuto} class:primary={autoCheck}>
          {autoCheck ? 'On' : 'Off'}
        </button>
      </div>
      <div class="row">
        <div>
          <div class="label">Repository</div>
          <div class="muted small">{status?.addonRepo}</div>
        </div>
        <button class="ghost" on:click={() => OpenURL(`https://github.com/${status?.addonRepo}`)}>
          Open ↗
        </button>
      </div>
      <div class="row">
        <div>
          <div class="label">Updater Version</div>
          <div class="muted small">
            {status?.appVersion}
            {#if selfUpdate?.updateAvailable}
              · update available: {selfUpdate.latestVersion}
            {/if}
          </div>
        </div>
        <button on:click={applySelf} disabled={!selfUpdate?.updateAvailable}>Update</button>
      </div>
    </section>
  {/if}
</main>

<style>
  main {
    display: flex;
    flex-direction: column;
    height: 100%;
    padding: 8px 10px 10px;
    gap: 7px;
    font-size: 12px;
  }

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    -webkit-app-region: drag;
  }
  nav { display: flex; gap: 4px; -webkit-app-region: no-drag; }
  nav .active { color: var(--text); background: var(--bg-3); }

  .brand { display: flex; align-items: center; gap: 8px; }
  .logo {
    width: 26px; height: 26px;
    border-radius: 6px;
    object-fit: cover;
    box-shadow: 0 2px 8px rgba(0,0,0,0.35);
    -webkit-user-drag: none;
  }
  .title { font-weight: 700; font-size: 12.5px; line-height: 1.1; }
  .subtitle { font-size: 10.5px; color: var(--muted); }

  .banner {
    display: flex; align-items: center; justify-content: space-between;
    padding: 7px 11px; border-radius: 8px; font-size: 12px;
    border: 1px solid var(--border);
  }
  .banner.error { background: rgba(239, 68, 68, 0.08); border-color: rgba(239, 68, 68, 0.4); }
  .banner.info  { background: rgba(245, 185, 66, 0.08); border-color: rgba(245, 185, 66, 0.4); }

  .hero {
    display: flex; flex-direction: column; align-items: center; gap: 9px;
    background: var(--bg-2);
    border: 1px solid var(--border);
    border-radius: 9px;
    padding: 12px 12px;
    flex: 1;
    overflow: auto;
  }
  .versions { display: flex; align-items: center; gap: 14px; }
  .vbox { text-align: center; min-width: 110px; }
  .label { font-size: 9.5px; text-transform: uppercase; letter-spacing: 0.08em; color: var(--muted); }
  .value { font-size: 17px; font-weight: 700; margin-top: 1px; line-height: 1.15; }
  .value.latest { color: var(--accent); }
  .arrow { font-size: 20px; color: var(--muted); }
  .actions { display: flex; gap: 8px; }
  .progress {
    width: 100%; max-width: 360px; height: 5px; background: var(--bg-3); border-radius: 3px; overflow: hidden;
  }
  .progress .bar { height: 100%; background: linear-gradient(90deg, var(--accent), var(--accent-2)); transition: width 0.15s; }
  .warn { color: var(--accent); font-size: 12px; }
  .muted { color: var(--muted); }
  .small { font-size: 11px; }
  .center { text-align: center; }

  .log {
    width: 100%; max-height: 110px; overflow: auto;
    background: #0c0e16; border: 1px solid var(--border); border-radius: 7px;
    padding: 6px 9px; font-family: ui-monospace, Consolas, monospace; font-size: 11px; color: #b9c0d4;
  }

  .changelog {
    background: var(--bg-2); border: 1px solid var(--border);
    border-radius: 9px; padding: 12px 14px; flex: 1; overflow: auto;
  }
  .changelog h2 { margin: 0 0 3px; font-size: 14px; }

  .md { font-size: 12.5px; line-height: 1.55; color: #d4d8e6; }
  .md :global(h1), .md :global(h2), .md :global(h3) {
    margin: 14px 0 6px; color: var(--text); font-weight: 700;
  }
  .md :global(h1) { font-size: 15px; }
  .md :global(h2) { font-size: 14px; color: var(--accent); }
  .md :global(h3) { font-size: 13px; }
  .md :global(ul), .md :global(ol) { margin: 4px 0 8px; padding-left: 20px; }
  .md :global(li) { margin: 2px 0; }
  .md :global(p)  { margin: 4px 0; }
  .md :global(strong) { color: var(--text); }
  .md :global(em)     { color: #b9c0d4; }
  .md :global(code) {
    background: #0c0e16; border: 1px solid var(--border); border-radius: 4px;
    padding: 1px 5px; font-family: ui-monospace, Consolas, monospace; font-size: 11.5px;
  }
  .md :global(pre) {
    background: #0c0e16; border: 1px solid var(--border); border-radius: 6px;
    padding: 8px 10px; overflow: auto;
  }
  .md :global(pre) :global(code) { background: transparent; border: 0; padding: 0; }
  .md :global(a) { color: var(--accent); }
  .md :global(hr) { border: none; border-top: 1px solid var(--border); margin: 12px 0; }
  .md :global(blockquote) {
    border-left: 3px solid var(--accent); margin: 6px 0; padding: 2px 10px; color: var(--muted);
  }

  .settings { display: flex; flex-direction: column; gap: 7px; }
  .settings .row {
    display: flex; align-items: center; justify-content: space-between;
    background: var(--bg-2); border: 1px solid var(--border);
    border-radius: 9px; padding: 10px 12px;
  }
  .row-actions { display: flex; gap: 6px; }
  .path { font-family: ui-monospace, Consolas, monospace; font-size: 11.5px; color: #c8cce0; margin-top: 3px; word-break: break-all; }
</style>
