/**
 * main.js — точка входа Mini App: инициализация Telegram SDK, темы,
 * роутера и таб-бара. Вне Telegram честно показывает demo-чип.
 */

import { init as initTg, isDemo, showBackButton, hideBackButton, topInsetPx, disableVerticalSwipes } from './services/tg.js';
import { initTheme } from './hooks/useTheme.js';
import { run, register } from './app.js';
import { TabBar } from './components/layout/chrome.js';

const app = document.getElementById('app');

// 1. Telegram SDK и тема — до первого рендера (без «вспышки» неверной палитры)
initTg();
initTheme();
const gap = topInsetPx();
if (gap) document.documentElement.style.setProperty('--tg-gap', `${gap}px`);

// 2. demo-чип (открыто не из Telegram)
if (isDemo) {
  const chip = document.createElement('div');
  chip.className = 'demo-chip';
  chip.textContent = 'DEMO';
  document.body.appendChild(chip);
}

// 3. Контейнер экрана + постоянный таб-бар
const screen = document.createElement('div');
screen.className = 'screen';
app.replaceChildren(screen);

const TABS = ['dashboard', 'objects', 'estimates', 'acts', 'price', 'report'];
let tabBar = null;
function syncTabBar() {
  const seg = (location.hash || '#/dashboard').replace(/^#\/?/, '').split('/')[0] || 'dashboard';
  const active = TABS.includes(seg) ? seg : 'dashboard';
  const next = TabBar(active, (id) => { location.hash = '#/' + id; });
  tabBar?.remove();
  tabBar = next;
  document.body.appendChild(next);
}
window.addEventListener('hashchange', syncTabBar);

// 4. Роут: маршруты → ленивые модули представлений
register('dashboard', () => import('./views/dashboard.js'));
register('objects', () => import('./views/objects.js'));
register('objects/:id', () => import('./views/object-detail.js'));
register('estimates', () => import('./views/estimates.js'));
register('estimates/:id', () => import('./views/estimate-detail.js'));
register('templates', () => import('./views/templates.js'));
register('acts', () => import('./views/acts.js'));
register('acts/:id', () => import('./views/act-detail.js'));
register('price', () => import('./views/price.js'));
register('report', () => import('./views/report.js'));

// TMA 2026: длинные сметы скроллятся — свайп-закрытие мешает, отключаем
// (вне Telegram / старые клиенты — no-op)
disableVerticalSwipes();

syncTabBar();
run({
  app: screen,
  showBack: (fnOrNull) => {
    if (!fnOrNull) { hideBackButton(); return () => {}; }
    return showBackButton(fnOrNull);
  },
}).then(syncTabBar);
