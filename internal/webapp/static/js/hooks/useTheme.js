/**
 * hooks/useTheme.js — синхронизация CSS-переменных с Telegram themeParams.
 *
 * При каждом themeChanged (и один раз на старте) мапит themeParams → токены
 * из css/tokens.css. Вне Telegram оставляет фолбэк-палитру (demo).
 */

import { themeParams, colorScheme, onThemeChanged, isDemo, topInsetPx } from '../services/tg.js';

let detach = null;

/** Гарантирует безопасный контраст: тёмный текст на светлом акценте. */
function readableOn(hex) {
  const m = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex || '');
  if (!m) return '#06281a';
  const [r, g, b] = [1, 2, 3].map((i) => parseInt(m[i], 16) / 255);
  const lin = (c) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4);
  const L = 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b);
  return L > 0.55 ? '#0b2b19' : '#ffffff';
}

/** hex → rgba() со своей альфой (для мягких подложек статусов). */
function alpha(hex, a) {
  const m = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex || '');
  if (!m) return hex;
  const [r, g, b] = [1, 2, 3].map((i) => parseInt(m[i], 16));
  return `rgba(${r}, ${g}, ${b}, ${a})`;
}

export function applyTheme() {
  const t = themeParams();
  const scheme = colorScheme();
  const root = document.documentElement.style;
  const light = scheme === 'light';

  root.setProperty('color-scheme', scheme);
  root.setProperty('--bg', t.bg_color || '#17212b');
  root.setProperty('--text', t.text_color || '#f2f5f7');
  root.setProperty('--hint', t.hint_color || '#7c8894');
  root.setProperty('--accent', t.button_color || '#2ecc71');
  root.setProperty('--on-accent', readableOn(t.button_color || '#2ecc71'));
  root.setProperty('--accent-text', readableOn(t.button_color || '#2ecc71'));

  // поверхности: glass на secondary_bg (или подобранном по схеме)
  const base = t.secondary_bg_color || (light ? '#eef1f5' : '#0e1621');
  root.setProperty('--bg', t.bg_color || '#17212b');
  root.setProperty('--card', light ? 'rgba(255,255,255,.62)' : 'rgba(255,255,255,.05)');
  root.setProperty('--card-strong', light ? 'rgba(255,255,255,.85)' : 'rgba(255,255,255,.09)');
  root.setProperty('--border', light ? 'rgba(15,23,42,.08)' : 'rgba(255,255,255,.09)');
  root.setProperty('--border-strong', light ? 'rgba(15,23,42,.16)' : 'rgba(255,255,255,.17)');
  root.setProperty('--bg-glow-1', alpha(t.button_color || '#2ecc71', light ? .10 : .07));
  root.setProperty('--bg-glow-2', alpha(t.link_color || '#f1c40f', light ? .08 : .05));
  root.setProperty('--accent-soft', alpha(t.button_color || '#2ecc71', light ? .12 : .15));

  // статусы от teaseParams не приходят — фиксируем и подмешиваем через alpha
  const ok = '#2ecc71', warn = light ? '#d9a406' : '#f1c40f', danger = light ? '#e05540' : '#ff6b57';
  root.setProperty('--ok', ok);
  root.setProperty('--ok-soft', alpha(ok, light ? .12 : .14));
  root.setProperty('--warn', warn);
  root.setProperty('--warn-soft', alpha(warn, light ? .12 : .13));
  root.setProperty('--danger', danger);
  root.setProperty('--danger-soft', alpha(danger, light ? .11 : .14));
  root.setProperty('--info', t.link_color || '#58b0e3');
  root.setProperty('--info-soft', alpha(t.link_color || '#58b0e3', light ? .12 : .15));

  // тени: на светлой схеме мягче
  root.setProperty('--shadow-1', light
    ? '0 1px 2px rgba(15,23,42,.05), 0 10px 30px rgba(15,23,42,.08)'
    : '0 1px 2px rgba(0,0,0,.25), 0 8px 28px rgba(0,0,0,.22)');
  root.setProperty('--shadow-accent', `0 8px 22px ${alpha(t.button_color || '#2ecc71', light ? .25 : .3)}`);

  document.documentElement.dataset.scheme = scheme;
  void base;
}

/** Включает живую синхронизацию темы + зазор шапки Telegram. */
export function initTheme() {
  applyTheme();
  // зазор под нативной шапкой Telegram на новых клиентах
  const setGap = () => document.documentElement.style.setProperty('--tg-gap', `${topInsetPx()}px`);
  setGap();
  if (isDemo) return;
  if (detach) detach();
  detach = onThemeChanged(() => {
    applyTheme();
    setGap();
  });
}
