import {
  init as initTg,
  isDemo,
  haptics,
  disableVerticalSwipes,
  showBackButton,
  hideBackButton,
} from './services/tg.js';
import { initTheme } from './hooks/useTheme.js';

const VALID_TABS = new Set(['overview', 'work', 'finance', 'documents']);
const tabs = [...document.querySelectorAll('[data-tab]')];
const screens = [...document.querySelectorAll('[data-screen]')];
const sheet = document.getElementById('report-sheet');
const toast = document.getElementById('client-toast');
const demoSwitch = document.getElementById('demo-switch');

let toastTimer = 0;
let detachBack = null;

initTg();
initTheme();
disableVerticalSwipes();

if (!isDemo && demoSwitch) demoSwitch.hidden = true;

function currentTab() {
  const candidate = location.hash.replace(/^#\/?/, '');
  return VALID_TABS.has(candidate) ? candidate : 'overview';
}

function activateTab(id, { syncHash = true } = {}) {
  const tab = VALID_TABS.has(id) ? id : 'overview';

  for (const screen of screens) {
    const active = screen.dataset.screen === tab;
    screen.classList.toggle('active', active);
    screen.toggleAttribute('aria-hidden', !active);
  }

  for (const button of tabs) {
    const active = button.dataset.tab === tab;
    button.classList.toggle('active', active);
    if (active) button.setAttribute('aria-current', 'page');
    else button.removeAttribute('aria-current');
  }

  if (syncHash && location.hash !== `#${tab}`) history.replaceState(null, '', `#${tab}`);
  const label = tabs.find((button) => button.dataset.tab === tab)?.textContent.trim() || 'мой объект';
  document.title = `ПрорАКТ 360 — ${label}`;
  window.scrollTo({ top: 0, behavior: 'auto' });
}

function showToast(message) {
  clearTimeout(toastTimer);
  toast.textContent = message;
  toast.classList.add('show');
  toastTimer = window.setTimeout(() => toast.classList.remove('show'), 2600);
}

function openReport() {
  if (!sheet.hidden) return;
  haptics.select();
  sheet.hidden = false;
  document.body.classList.add('sheet-open');
  detachBack?.();
  detachBack = showBackButton(closeReport);
  requestAnimationFrame(() => sheet.querySelector('.sheet-close')?.focus({ preventScroll: true }));
}

function closeReport() {
  if (sheet.hidden) return;
  haptics.tap();
  sheet.hidden = true;
  document.body.classList.remove('sheet-open');
  detachBack?.();
  detachBack = null;
  hideBackButton();
}

function handleDemoAction(action) {
  haptics.tap();
  const messages = {
    question: 'В рабочей версии здесь откроется чат с Виталием',
    photos: 'Откроется полный фотоотчёт выбранного этапа',
    document: 'В рабочей версии откроется исходный PDF или XLSX',
    download: 'Документы будут скачаны одним архивом',
  };
  showToast(messages[action] || 'Функция будет подключена к данным объекта');
}

for (const button of tabs) {
  button.addEventListener('click', () => {
    if (button.dataset.tab === currentTab()) return;
    haptics.select();
    location.hash = button.dataset.tab;
  });
}

for (const opener of document.querySelectorAll('[data-open-report]')) {
  opener.addEventListener('click', openReport);
  opener.addEventListener('keydown', (event) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      openReport();
    }
  });
}

for (const closer of document.querySelectorAll('[data-close-report]')) {
  closer.addEventListener('click', closeReport);
}

for (const action of document.querySelectorAll('[data-action]')) {
  action.addEventListener('click', () => handleDemoAction(action.dataset.action));
}

document.addEventListener('keydown', (event) => {
  if (event.key === 'Escape' && !sheet.hidden) closeReport();
});

window.addEventListener('hashchange', () => activateTab(currentTab(), { syncHash: false }));

activateTab(currentTab());
