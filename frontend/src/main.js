import './style.css';
import './app.css';

import iconUrl from './assets/images/icon.png';
import { ListPorts, KillProcess, OpenFolder, GetSettings, SaveSettings, CloseWindow, StartHidden } from '../wailsjs/go/main/App';
import { WindowToggleMaximise } from '../wailsjs/runtime/runtime';

const state = {
  rows: [],
  query: '',
  source: 'all',
  loading: false,
  auto: true,
  interval: 5,
  showPaths: true,
  showSystem: true,
  killing: new Set(),
  collapsed: new Set(),
  lastUpdated: null,
  modal: null,
  closeAction: 'ask',
  startOnBoot: false,
};

const app = document.getElementById('app');

app.innerHTML = `
  <div class="wrap">
    <div class="titlebar" id="titlebar">
      <div class="tb-left">
        <img class="tb-icon" src="${iconUrl}" alt="" draggable="false"/>
        <span class="tb-title">Ports</span>
      </div>
      <div class="tb-center" id="tbStatus"></div>
      <div class="tb-right">
        <button class="tb-btn" id="settingsBtn" title="Settings"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg></button>
        <button class="tb-btn" id="aboutBtn" title="About"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg></button>
        <span class="tb-sep"></span>
        <button class="tb-btn" id="minBtn" title="Minimize"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="5" y1="12" x2="19" y2="12"/></svg></button>
        <button class="tb-btn" id="maxBtn" title="Maximize"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="5" y="5" width="14" height="14" rx="1"/></svg></button>
        <button class="tb-btn close" id="closeBtn" title="Close"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="6" y1="6" x2="18" y2="18"/><line x1="18" y1="6" x2="6" y2="18"/></svg></button>
      </div>
    </div>

    <header>
      <div class="controls">
        <div class="search-box">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
          <input id="search" class="search" type="text" placeholder="Filter by name, port, pid..." autocomplete="off" spellcheck="false"/>
        </div>
        <div class="chips" id="chips">
          <button class="chip active" data-src="all">All</button>
          <button class="chip" data-src="windows">Windows</button>
          <button class="chip" data-src="wsl">WSL</button>
        </div>
        <div class="spacer"></div>
        <button id="autoBtn" class="btn ghost toggle" title="Toggle auto refresh">Auto</button>
        <button id="refreshBtn" class="btn primary">Refresh</button>
      </div>
    </header>

    <main id="main">
      <div class="table-head">
        <span>Program</span>
        <span>Ports</span>
        <span>Path</span>
        <span></span>
      </div>
      <div id="list" class="list"></div>
      <div id="empty" class="empty hidden">
        <div class="empty-ring"></div>
        <div>No processes listening on ports</div>
        <div class="empty-sub">Press Refresh to scan again</div>
      </div>
    </main>

    <footer>
      <span id="count"></span>
      <span id="updated"></span>
    </footer>
  </div>

  <div id="modalBack" class="modal-back hidden">
    <div class="modal" id="modalSettings">
      <div class="modal-head">
        <h2>Settings</h2>
        <button class="tb-btn" data-close><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="6" y1="6" x2="18" y2="18"/><line x1="18" y1="6" x2="6" y2="18"/></svg></button>
      </div>
      <div class="modal-body">
        <label class="setting-row">
          <div><div class="s-title">Auto refresh</div><div class="s-sub">Periodically rescan listening ports</div></div>
          <input type="checkbox" id="setAuto" class="switch"/>
        </label>
        <label class="setting-row">
          <div><div class="s-title">Refresh interval</div><div class="s-sub">Seconds between scans</div></div>
          <input type="number" id="setInterval" class="s-input" min="2" max="120"/>
        </label>
        <label class="setting-row">
          <div><div class="s-title">Show executable paths</div><div class="s-sub">Display the binary location for each program</div></div>
          <input type="checkbox" id="setPaths" class="switch"/>
        </label>
        <label class="setting-row">
          <div><div class="s-title">Show system processes</div><div class="s-sub">Include Windows services and kernel listeners</div></div>
          <input type="checkbox" id="setSystem" class="switch"/>
        </label>
        <label class="setting-row">
          <div><div class="s-title">Start on launch</div><div class="s-sub">Start Ports hidden in the tray when you log in</div></div>
          <input type="checkbox" id="setStartBoot" class="switch"/>
        </label>
        <div class="setting-row" style="cursor:default">
          <div><div class="s-title">When closing the window</div><div class="s-sub">What happens when you click X</div></div>
          <select id="setCloseAction" class="s-select">
            <option value="ask">Ask me</option>
            <option value="hide">Hide to tray</option>
            <option value="quit">Quit</option>
          </select>
        </div>
      </div>
      <div class="modal-foot">
        <button class="btn" data-close>Cancel</button>
        <button class="btn primary" id="saveSettings">Save</button>
      </div>
    </div>

    <div class="modal about" id="modalAbout">
      <img class="about-icon-img" src="${iconUrl}" alt="" draggable="false"/>
      <h2>Ports</h2>
      <div class="about-ver">v1.0.0</div>
      <p class="about-desc">See every app holding a port on Windows and WSL, and kill it in one click.</p>
      <div class="about-creator">
        <div class="made-by">Made by</div>
        <div class="creator-name">Ainz</div>
        <a href="#" id="ainzLink" class="creator-site">ainz.uk</a>
      </div>
      <button class="btn" data-close style="margin-top:18px">Close</button>
    </div>

    <div class="modal confirm" id="modalConfirm">
      <div class="confirm-icon">&#9888;</div>
      <h2 id="confirmTitle">Kill process?</h2>
      <p class="confirm-desc" id="confirmText"></p>
      <div class="confirm-actions">
        <button class="btn" id="confirmCancel">Cancel</button>
        <button class="btn danger" id="confirmOk">Kill</button>
      </div>
    </div>

    <div class="modal confirm" id="modalClose">
      <div class="confirm-icon neutral">&#9635;</div>
      <h2>Close Ports?</h2>
      <p class="confirm-desc">Ports can keep running in the system tray so you can kill processes anytime, or exit completely.</p>
      <div class="confirm-actions">
        <button class="btn" id="closeHide">Hide to Tray</button>
        <button class="btn danger" id="closeQuit">Quit</button>
      </div>
      <label class="remember-row">
        <input type="checkbox" id="closeRemember" class="switch"/>
        <span>Remember my choice</span>
      </label>
    </div>
  </div>

  <div id="toast" class="toast hidden"></div>
`;

const listEl = document.getElementById('list');
const emptyEl = document.getElementById('empty');
const searchEl = document.getElementById('search');
const countEl = document.getElementById('count');
const updatedEl = document.getElementById('updated');
const statusEl = document.getElementById('tbStatus');
const toastEl = document.getElementById('toast');
const refreshBtn = document.getElementById('refreshBtn');
const autoBtn = document.getElementById('autoBtn');
const chipsEl = document.getElementById('chips');
const modalBack = document.getElementById('modalBack');
const titlebar = document.getElementById('titlebar');

const SYSTEM_NAMES = new Set(['system', 'services.exe', 'lsass.exe', 'svchost.exe', 'wininit.exe', 'spoolsv.exe', 'system idle process', 'registry', 'smss.exe', 'csrss.exe', 'winlogon.exe', 'dwm.exe', 'fontdrvhost.exe', 'systemd-resolve', 'systemd-journald', 'systemd-udevd', 'systemd-logind']);

function esc(s) {
  return String(s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

let toastTimer = null;
function toast(msg, isErr) {
  toastEl.textContent = msg;
  toastEl.classList.toggle('error', !!isErr);
  toastEl.classList.remove('hidden');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => toastEl.classList.add('hidden'), 3200);
}

function shortPath(p) {
  if (!p) return '';
  return p.length > 46 ? p.slice(0, 20) + '...' + p.slice(-24) : p;
}

function portChip(p) {
  const addr = p.addr && !['*', '0.0.0.0', '::', '[::]'].includes(p.addr) ? `<em title="${esc(p.addr)}">${esc(shortPath(p.addr))}</em>` : '';
  return `<span class="port"><b>${p.port}</b><i>${esc(p.proto)}</i>${addr}</span>`;
}

function isSystem(row) {
  return SYSTEM_NAMES.has(row.name.toLowerCase());
}

function matchesQuery(row, q) {
  if (row.name.toLowerCase().includes(q)) return true;
  if (String(row.pid).includes(q)) return true;
  if ((row.path || '').toLowerCase().includes(q)) return true;
  return row.ports.some(p => String(p.port).includes(q) || (p.addr || '').toLowerCase().includes(q));
}

function groupKey(row) {
  return row.source + '|' + row.name.toLowerCase();
}

function groups() {
  const q = state.query.trim().toLowerCase();
  const filtered = state.rows.filter(r => {
    if (state.source !== 'all' && r.source !== state.source) return false;
    if (!state.showSystem && isSystem(r)) return false;
    if (q && !matchesQuery(r, q)) return false;
    return true;
  });
  const map = new Map();
  for (const r of filtered) {
    const k = groupKey(r);
    if (!map.has(k)) {
      map.set(k, {
        key: k,
        name: r.name,
        source: r.source,
        distro: r.distro || '',
        path: r.path || '',
        pids: [],
        ports: [],
      });
    }
    const g = map.get(k);
    g.pids.push(r.pid);
    for (const p of r.ports) {
      if (!g.ports.some(x => x.port === p.port && x.proto === p.proto)) g.ports.push(p);
    }
    if (r.path && (!g.path || g.path === r.path)) g.path = r.path;
    else if (r.path && g.path && r.path !== g.path) g.path = '';
  }
  g: for (const g of map.values()) {
    g.ports.sort((a, b) => a.port - b.port || a.proto.localeCompare(b.proto));
    g.pids.sort((a, b) => a - b);
  }
  return [...map.values()];
}

function render() {
  const gs = groups();
  emptyEl.classList.toggle('hidden', gs.length > 0 || state.loading);

  listEl.innerHTML = gs.map(g => {
    const collapsed = state.collapsed.has(g.key);
    const busy = g.pids.some(pid => state.killing.has(`${g.source}|${g.distro}|${pid}`));
    const srcBadge = g.source === 'wsl'
      ? `<span class="badge wsl">WSL</span>${g.distro ? `<span class="distro">${esc(g.distro)}</span>` : ''}`
      : `<span class="badge win">Windows</span>`;
    const ports = g.ports.slice(0, 10).map(portChip).join('');
    const more = g.ports.length > 10 ? `<span class="port more">+${g.ports.length - 10}</span>` : '';
    const pidLabel = g.pids.length === 1 ? `PID ${g.pids[0]}` : `${g.pids.length} processes`;
    const pathCell = state.showPaths
      ? (g.path
          ? `<span class="path" title="${esc(g.path)}">${esc(shortPath(g.path))}</span>`
          : `<span class="path none">unknown</span>`)
      : '';
    const folderBtn = g.path ? `<button class="btn icon" data-folder="${esc(g.key)}" title="Open containing folder"><svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg></button>` : '';
    return `
      <div class="group ${collapsed ? 'collapsed' : ''}" data-key="${esc(g.key)}">
        <div class="row group-head" data-toggle="${esc(g.key)}">
          <div class="cell name">
            <span class="chev">${collapsed ? '&#9656;' : '&#9662;'}</span>
            <div class="name-stack">
              <span class="pname" title="${esc(g.name)}">${esc(g.name)}</span>
              <span class="pid">${srcBadge} <span class="pid-label">${pidLabel}</span></span>
            </div>
          </div>
          <div class="cell ports">${ports}${more}</div>
          <div class="cell pathcell">${pathCell}</div>
          <div class="cell act">
            ${folderBtn}
            <button class="btn kill" data-kill="${esc(g.key)}" ${busy ? 'disabled' : ''}>${busy ? 'Killing...' : 'Kill'}</button>
          </div>
        </div>
        ${!collapsed ? `
        <div class="group-children">
          ${g.pids.length > 1 || (g.path && state.showPaths) ? `
          <div class="child head"><span>PID</span><span>Ports</span><span>Path</span></div>` : ''}
          ${childRows(g)}
        </div>` : ''}
      </div>`;
  }).join('');

  const total = gs.reduce((n, g) => n + g.pids.length, 0);
  countEl.textContent = `${gs.length} programs \u00b7 ${total} processes`;
}

function childRows(g) {
  return g.pids.map(pid => {
    const row = state.rows.find(r => r.source === g.source && r.distro === (g.distro || '') && r.pid === pid);
    if (!row) return '';
    const single = g.pids.length === 1;
    const path = row.path && state.showPaths ? `<span class="path" title="${esc(row.path)}">${esc(shortPath(row.path))}</span>` : '';
    return `
      <div class="child">
        <span class="child-pid">${single ? '' : pid}</span>
        <span class="child-ports">${row.ports.map(portChip).join('')}</span>
        <span class="child-path">${single ? '' : path}</span>
      </div>`;
  }).join('');
}

async function refresh() {
  if (state.loading) return;
  state.loading = true;
  refreshBtn.disabled = true;
  refreshBtn.textContent = 'Scanning...';
  try {
    const rows = await ListPorts();
    if (state.killing.size === 0) {
      state.rows = rows || [];
      state.lastUpdated = new Date();
      updatedEl.textContent = `Updated ${state.lastUpdated.toLocaleTimeString()}`;
    }
    statusEl.textContent = '';
  } catch (e) {
    statusEl.textContent = 'Scan failed';
    toast('Scan failed: ' + (e && e.message ? e.message : e), true);
  } finally {
    state.loading = false;
    refreshBtn.disabled = false;
    refreshBtn.textContent = 'Refresh';
    render();
  }
}

async function killGroup(key) {
  const g = state.rows.length ? groups().find(x => x.key === key) : null;
  if (!g) return;
  const label = `${g.name}${g.source === 'wsl' ? ' (' + g.distro + ')' : ''}`;
  const detail = g.pids.length === 1 ? `PID ${g.pids[0]}` : `${g.pids.length} processes (${g.pids.join(', ')})`;
  const confirmed = await confirmKill(`Kill "${label}"?`, `This will force-terminate ${detail}. The app will lose any unsaved work.`);
  if (!confirmed) return;

  const pidKeys = g.pids.map(pid => `${g.source}|${g.distro}|${pid}`);
  for (const k of pidKeys) state.killing.add(k);
  render();
  statusEl.textContent = `Killing ${g.name}...`;

  // Optimistic removal: drop the rows immediately so the UI reacts instantly.
  const killedKeys = new Set(g.pids.map(pid => rowKey(g.source, g.distro, pid)));
  const removed = state.rows.filter(r => killedKeys.has(r.key)).map(r => ({ ...r }));
  state.rows = state.rows.filter(r => !killedKeys.has(r.key));
  render();

  let failed = 0;
  for (const pid of g.pids) {
    try {
      await KillProcess(g.source, g.distro, pid);
    } catch (e) {
      failed++;
      toast(`Failed to kill PID ${pid}: ${e && e.message ? e.message : e}`, true);
    }
  }
  for (const k of pidKeys) state.killing.delete(k);

  if (failed > 0) {
    // Restore rows for processes that survived.
    state.rows.push(...removed);
    state.rows.sort((a, b) => a.source.localeCompare(b.source) || a.name.localeCompare(b.name));
  } else {
    toast(`Killed ${label}`);
  }
  statusEl.textContent = '';
  await refresh();
}

function rowKey(source, distro, pid) {
  return source === 'wsl' ? `wsl|${distro || ''}|${pid}` : `windows||${pid}`;
}

async function openFolder(key) {
  const g = groups().find(x => x.key === key);
  if (!g || !g.path) return;
  try {
    await OpenFolder(g.source, g.distro || '', g.path);
  } catch (e) {
    toast('Could not open folder: ' + (e && e.message ? e.message : e), true);
  }
}

function openModal(which) {
  state.modal = which;
  document.getElementById('modalSettings').classList.toggle('hidden', which !== 'settings');
  document.getElementById('modalAbout').classList.toggle('hidden', which !== 'about');
  document.getElementById('modalConfirm').classList.toggle('hidden', which !== 'confirm');
  document.getElementById('modalClose').classList.toggle('hidden', which !== 'close');
  modalBack.classList.remove('hidden');
}

function closeModal() {
  state.modal = null;
  modalBack.classList.add('hidden');
}

function confirmKill(title, text) {
  return new Promise(resolve => {
    document.getElementById('confirmTitle').textContent = title;
    document.getElementById('confirmText').textContent = text;
    openModal('confirm');
    const ok = document.getElementById('confirmOk');
    const cancel = document.getElementById('confirmCancel');
    const done = val => {
      ok.removeEventListener('click', onOk);
      cancel.removeEventListener('click', onCancel);
      closeModal();
      resolve(val);
    };
    const onOk = () => done(true);
    const onCancel = () => done(false);
    ok.addEventListener('click', onOk);
    cancel.addEventListener('click', onCancel);
  });
}

// handleCloseButton routes the X click through the saved preference or the
// close modal.
async function handleCloseButton() {
  let action = state.closeAction;
  if (action === 'ask') {
    action = await openCloseModal();
    if (!action) return;
  }
  applyCloseAction(action);
}

function openCloseModal() {
  return new Promise(resolve => {
    document.getElementById('closeRemember').checked = false;
    openModal('close');
    const hideBtn = document.getElementById('closeHide');
    const quitBtn = document.getElementById('closeQuit');
    const done = val => {
      hideBtn.removeEventListener('click', onHide);
      quitBtn.removeEventListener('click', onQuit);
      closeModal();
      resolve(val);
    };
    const onHide = () => {
      if (document.getElementById('closeRemember').checked) rememberClose('hide');
      done('hide');
    };
    const onQuit = () => {
      if (document.getElementById('closeRemember').checked) rememberClose('quit');
      done('quit');
    };
    const onHideRef = onHide;
    hideBtn.addEventListener('click', onHideRef);
    quitBtn.addEventListener('click', onQuit);
  });
}

async function rememberClose(action) {
  const s = { ...currentSettingsState(), closeAction: action };
  try { await SaveSettings(s); } catch { /* non-fatal */ }
  state.closeAction = action;
}

function currentSettingsState() {
  return {
    autoRefresh: state.auto,
    intervalSecs: state.interval,
    showPaths: state.showPaths,
    showSystem: state.showSystem,
    startOnBoot: state.startOnBoot,
    closeAction: state.closeAction,
  };
}

function applyCloseAction(action) {
  if (action === 'quit') {
    CloseWindow('quit');
  } else {
    CloseWindow('hide');
  }
}

async function loadSettings() {
  try {
    const s = await GetSettings();
    state.auto = !!s.autoRefresh;
    state.interval = s.intervalSecs || 5;
    state.showPaths = s.showPaths !== false;
    state.showSystem = s.showSystem !== false;
    state.closeAction = s.closeAction || 'ask';
    state.startOnBoot = !!s.startOnBoot;
  } catch { /* defaults */ }
  autoBtn.classList.toggle('active', state.auto);
  if (state.closeAction === 'ask' && await StartHidden()) {
    state.closeAction = 'hide';
  }
}

async function openSettings() {
  document.getElementById('setAuto').checked = state.auto;
  document.getElementById('setInterval').value = state.interval;
  document.getElementById('setPaths').checked = state.showPaths;
  document.getElementById('setSystem').checked = state.showSystem;
  document.getElementById('setStartBoot').checked = state.startOnBoot;
  document.getElementById('setCloseAction').value = state.closeAction;
  openModal('settings');
}

async function saveSettings() {
  const s = {
    autoRefresh: document.getElementById('setAuto').checked,
    intervalSecs: parseInt(document.getElementById('setInterval').value, 10) || 5,
    showPaths: document.getElementById('setPaths').checked,
    showSystem: document.getElementById('setSystem').checked,
    startOnBoot: document.getElementById('setStartBoot').checked,
    closeAction: document.getElementById('setCloseAction').value,
  };
  try {
    await SaveSettings(s);
  } catch (e) {
    toast('Could not save settings: ' + (e && e.message ? e.message : e), true);
  }
  const wasAuto = state.auto;
  state.auto = s.autoRefresh;
  state.interval = s.intervalSecs;
  state.showPaths = s.showPaths;
  state.showSystem = s.showSystem;
  state.closeAction = s.closeAction;
  state.startOnBoot = s.startOnBoot;
  autoBtn.classList.toggle('active', state.auto);
  if (state.auto !== wasAuto) statusEl.textContent = state.auto ? 'Auto refresh on' : 'Auto refresh off';
  closeModal();
  render();
}

listEl.addEventListener('click', e => {
  const killBtn = e.target.closest('button[data-kill]');
  if (killBtn) { killGroup(killBtn.dataset.kill); return; }
  const folderBtn = e.target.closest('button[data-folder]');
  if (folderBtn) { openFolder(folderBtn.dataset.folder); return; }
  const head = e.target.closest('[data-toggle]');
  if (head) {
    const key = head.dataset.toggle;
    state.collapsed.has(key) ? state.collapsed.delete(key) : state.collapsed.add(key);
    render();
  }
});

chipsEl.addEventListener('click', e => {
  const chip = e.target.closest('.chip');
  if (!chip) return;
  state.source = chip.dataset.src;
  chipsEl.querySelectorAll('.chip').forEach(c => c.classList.toggle('active', c === chip));
  render();
});

searchEl.addEventListener('input', () => { state.query = searchEl.value; render(); });
refreshBtn.addEventListener('click', refresh);
autoBtn.addEventListener('click', () => {
  state.auto = !state.auto;
  autoBtn.classList.toggle('active', state.auto);
  statusEl.textContent = state.auto ? 'Auto refresh on' : 'Auto refresh off';
});

document.getElementById('settingsBtn').addEventListener('click', openSettings);
document.getElementById('aboutBtn').addEventListener('click', () => openModal('about'));
document.getElementById('saveSettings').addEventListener('click', saveSettings);
document.getElementById('ainzLink').addEventListener('click', e => {
  e.preventDefault();
  window.open('https://ainz.uk', '_blank');
});

modalBack.addEventListener('click', e => {
  if (e.target === modalBack || e.target.closest('[data-close]')) {
    if (state.modal === 'confirm' || state.modal === 'close') return;
    closeModal();
  }
});

document.getElementById('minBtn').addEventListener('click', () => CloseWindow('hide'));
document.getElementById('maxBtn').addEventListener('click', () => WindowToggleMaximise());
document.getElementById('closeBtn').addEventListener('click', handleCloseButton);

document.getElementById('closeHide').addEventListener('click', () => applyCloseAction('hide'));
document.getElementById('closeQuit').addEventListener('click', () => applyCloseAction('quit'));

titlebar.addEventListener('dblclick', e => {
  if (e.target.closest('button')) return;
  WindowToggleMaximise();
});

let sinceLast = 0;
setInterval(() => {
  sinceLast++;
  if (state.auto && sinceLast >= state.interval && !state.loading) {
    sinceLast = 0;
    refresh();
  }
}, 1000);

loadSettings().then(refresh);
