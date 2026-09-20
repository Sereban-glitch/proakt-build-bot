/**
 * components/layout/chrome.js — каркас экрана: шапка и таб-бар.
 */

import { icon } from '../ui/icon.js';
import { openSheet } from '../ui/feedback.js';
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

/** Фирменная шапка рабочего стола — спокойнее обычного app-header. */
export function BrandHeader() {
  const h = document.createElement('header');
  h.className = 'brand-header';

  const mark = document.createElement('div');
  mark.className = 'brand-mark';
  mark.setAttribute('aria-hidden', 'true');
  const markImage = document.createElement('img');
  markImage.src = 'assets/proakt360-avatar.webp';
  markImage.alt = '';
  mark.appendChild(markImage);

  const copy = document.createElement('div');
  copy.className = 'grow';
  const eyebrow = document.createElement('div');
  eyebrow.className = 'brand-eyebrow';
  eyebrow.textContent = 'ЦИФРОВОЙ ПРОРАБ';
  const title = document.createElement('div');
  title.className = 'brand-title';
  title.textContent = 'ПрорАКТ 360';
  const sub = document.createElement('div');
  sub.className = 'brand-subtitle';
  sub.textContent = 'Сметы, объекты и оплаты';
  copy.append(eyebrow, title, sub);

  const status = document.createElement('div');
  status.className = 'brand-status';
  const dot = document.createElement('span');
  dot.className = 'brand-status-dot';
  status.append(dot, document.createTextNode('Рабочий стол'));

  h.append(mark, copy, status);
  return h;
}

function openMoreMenu(active, navigate) {
  const menu = document.createElement('div');
  menu.className = 'more-grid';
  let sheet;
  const items = [
    { id: 'acts', label: 'Акты', note: 'Работы и оплаты', ic: 'acts' },
    { id: 'price', label: 'Прайс', note: 'Цены на работы', ic: 'price' },
    { id: 'report', label: 'Отчёт', note: 'Долги и показатели', ic: 'report' },
    { id: 'templates', label: 'Шаблоны', note: 'Готовые наборы работ', ic: 'template' },
  ];
  for (const item of items) {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = `more-action${active === item.id ? ' active' : ''}`;
    const ic = document.createElement('span');
    ic.className = 'more-action-ic';
    ic.appendChild(icon(item.ic, 22));
    const copy = document.createElement('span');
    copy.className = 'more-action-copy';
    const title = document.createElement('b');
    title.textContent = item.label;
    const note = document.createElement('small');
    note.textContent = item.note;
    copy.append(title, note);
    btn.append(ic, copy, icon('chevron', 17));
    btn.addEventListener('click', () => {
      haptics.select();
      sheet.close();
      navigate(item.id);
    });
    menu.appendChild(btn);
  }
  const hint = document.createElement('p');
  hint.className = 'more-hint';
  hint.textContent = 'Основные разделы всегда внизу. Остальные инструменты собраны здесь, чтобы не перегружать экран.';
  sheet = openSheet({ title: 'Все инструменты', body: [menu, hint] });
}

/** Главная навигация: три рабочих раздела + компактное меню инструментов. */
export function TabBar(active, navigate) {
  const items = [
    { id: 'dashboard', label: 'Главная', ic: 'home' },
    { id: 'objects', label: 'Объекты', ic: 'objects' },
    { id: 'estimates', label: 'Сметы', ic: 'estimate' },
    { id: 'more', label: 'Ещё', ic: 'more' },
  ];
  const primary = ['dashboard', 'objects', 'estimates'];
  const selected = primary.includes(active) ? active : 'more';
  const nav = document.createElement('nav');
  nav.className = 'tabbar';
  const inner = document.createElement('div');
  inner.className = 'tabbar-inner';
  inner.setAttribute('role', 'tablist');
  for (const it of items) {
    const b = document.createElement('button');
    b.className = `tab${selected === it.id ? ' active' : ''}`;
    b.setAttribute('role', 'tab');
    b.setAttribute('aria-selected', String(selected === it.id));
    b.setAttribute('aria-label', it.label);
    b.appendChild(icon(it.ic, 21));
    const lbl = document.createElement('span');
    lbl.className = 'tab-label';
    lbl.textContent = it.label;
    b.appendChild(lbl);
    b.addEventListener('click', () => {
      if (it.id === 'more') {
        haptics.select();
        openMoreMenu(active, navigate);
        return;
      }
      if (selected === it.id) return;
      haptics.select();
      navigate(it.id);
    });
    inner.appendChild(b);
  }
  nav.appendChild(inner);
  return nav;
}

export { icon };
