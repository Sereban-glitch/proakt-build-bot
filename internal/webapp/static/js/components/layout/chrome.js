/**
 * components/layout/chrome.js — каркас экрана: шапка и таб-бар.
 */

import { icon } from '../ui/icon.js';
import { haptics } from '../../services/tg.js';

/** Header({ title, subtitle, back }) — back: () => history.back()-подобный колбэк. */
export function Header({ title, subtitle = '', back } = {}) {
  const h = document.createElement('header');
  h.className = 'app-header';
  if (back) {
    const b = document.createElement('button');
    b.className = 'icon-btn';
    b.setAttribute('aria-label', 'Назад');
    b.appendChild(icon('back', 20));
    b.addEventListener('click', () => { haptics.tap(); back(); });
    h.appendChild(b);
  }
  const wrap = document.createElement('div');
  wrap.className = 'grow';
  const t = document.createElement('div');
  t.className = 'title ellipsis';
  t.textContent = title;
  wrap.appendChild(t);
  if (subtitle) {
    const s = document.createElement('div');
    s.className = 'subtitle ellipsis';
    s.textContent = subtitle;
    wrap.appendChild(s);
  }
  h.appendChild(wrap);
  return h;
}

/** Табы главного экрана. active: dashboard|objects|estimates|acts|price|report */
export function TabBar(active, navigate) {
  const items = [
    { id: 'dashboard', label: 'Сводка', ic: 'home' },
    { id: 'objects', label: 'Объекты', ic: 'objects' },
    { id: 'estimates', label: 'Сметы', ic: 'estimate' },
    { id: 'acts', label: 'Акты', ic: 'acts' },
    { id: 'price', label: 'Прайс', ic: 'price' },
    { id: 'report', label: 'Отчёт', ic: 'report' },
  ];
  const nav = document.createElement('nav');
  nav.className = 'tabbar';
  const inner = document.createElement('div');
  inner.className = 'tabbar-inner';
  inner.setAttribute('role', 'tablist');
  for (const it of items) {
    const b = document.createElement('button');
    b.className = `tab${active === it.id ? ' active' : ''}`;
    b.setAttribute('role', 'tab');
    b.setAttribute('aria-selected', String(active === it.id));
    b.setAttribute('aria-label', it.label);
    b.appendChild(icon(it.ic, 21));
    const lbl = document.createElement('span');
    lbl.textContent = it.label;
    b.appendChild(lbl);
    b.addEventListener('click', () => {
      if (active === it.id) return;
      haptics.select();
      navigate(it.id);
    });
    inner.appendChild(b);
  }
  nav.appendChild(inner);
  return nav;
}

export { icon };
