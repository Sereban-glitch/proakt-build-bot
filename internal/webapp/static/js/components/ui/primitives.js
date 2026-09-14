/**
 * components/ui/primitives.js — кирпичики UI: кнопка, бейдж, карточка,
 * ячейка списка, метрика, пустое состояние, скелетон, поиск.
 * Каждый компонент — чистая функция (el, props) → HTMLElement,
 * с колбэками: <Button onClick={...} loading disabled variant ...>.
 */

import { icon } from './icon.js';
import { haptics } from '../../services/tg.js';

// --- Button ---------------------------------------------------------------

/**
 * Button({ label, icon, variant='primary'|'secondary'|'danger'|'ghost',
 *          size='md'|'sm', block, loading, disabled, onClick, aria })
 */
export function Button({ label, iconName, variant = 'primary', size = 'md', block = false, onClick, aria } = {}) {
  const b = document.createElement('button');
  b.type = 'button';
  b.className = `btn btn-${variant}${size === 'sm' ? ' btn-sm' : ''}${block ? ' btn-block' : ''}`;
  if (aria) b.setAttribute('aria-label', aria);
  if (iconName) b.appendChild(icon(iconName, size === 'sm' ? 16 : 18));
  if (label) {
    const span = document.createElement('span');
    span.textContent = label;
    b.appendChild(span);
  }
  b.addEventListener('click', async (e) => {
    if (b.classList.contains('loading') || b.disabled) return;
    haptics.tap();
    if (onClick) {
      try { await onClick(e); }
      catch (err) {
        console.error('[button]', err);
        haptics.error();
      }
    }
  });
  return {
    el: b,
    setLoading(on) {
      b.classList.toggle('loading', Boolean(on));
      b.disabled = Boolean(on);
    },
    setDisabled(on) { b.disabled = Boolean(on); },
  };
}

// --- Badge ------------------------------------------------------------------

export function Badge({ text, tone = '', dot = false } = {}) {
  const s = document.createElement('span');
  s.className = `badge ${tone}`;
  if (dot) {
    const d = document.createElement('span');
    d.className = 'dot';
    s.appendChild(d);
  }
  s.appendChild(document.createTextNode(text));
  return s;
}

/** Статус акта: оплачен / частично / не оплачен / без суммы. */
export function actBadge(total, paid) {
  if (total < 0.01) return Badge({ text: 'без суммы', tone: 'warn' });
  if (paid >= total - 0.009) return Badge({ text: 'оплачен', tone: 'ok' });
  if (paid > 0.009) return Badge({ text: 'частично', tone: 'info' });
  return Badge({ text: 'не оплачен', tone: 'danger' });
}

// --- Card ---------------------------------------------------------------------

export function Card({ className = '', clickable, onClick } = {}) {
  const el = document.createElement('div');
  el.className = `card ${className}`.trim();
  if (clickable) {
    el.classList.add('clickable');
    el.setAttribute('role', 'button');
    el.tabIndex = 0;
    el.addEventListener('click', (e) => { haptics.tap(); onClick?.(e); });
    el.addEventListener('keydown', (e) => {
      if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onClick?.(e); }
    });
  }
  return el;
}

// --- Cell -----------------------------------------------------------------------

export function Cell({ icon: ic, iconTone, title, sub, value, note, badge, onClick, rightSlot } = {}) {
  const el = document.createElement(onClick ? 'button' : 'div');
  el.className = 'cell';
  if (onClick) {
    el.type = 'button';
    el.style.cssText = 'background:none;border:0;width:100%;font:inherit;';
    el.addEventListener('click', () => { haptics.tap(); onClick(); });
  }
  if (ic) {
    const wrap = document.createElement('span');
    wrap.className = 'cell-ic';
    if (iconTone) wrap.style.color = iconTone;
    wrap.appendChild(icon(ic, 20));
    el.appendChild(wrap);
  }
  const main = document.createElement('div');
  main.className = 'grow';
  const t = document.createElement('div');
  t.className = 'cell-title ellipsis';
  t.textContent = title;
  main.appendChild(t);
  if (sub) {
    const s = document.createElement('div');
    s.className = 'cell-sub';
    s.textContent = sub;
    main.appendChild(s);
  }
  el.appendChild(main);
  const right = document.createElement('div');
  right.className = 'cell-right';
  if (value) {
    const v = document.createElement('div');
    v.className = 'cell-value num';
    v.textContent = value;
    right.appendChild(v);
  }
  if (note) {
    const n = document.createElement('div');
    n.className = 'cell-note';
    n.textContent = note;
    right.appendChild(n);
  }
  if (badge) right.appendChild(badge);
  if (rightSlot) right.appendChild(rightSlot);
  if (right.childElementCount || right.textContent) el.appendChild(right);
  return el;
}

// --- Stat (метрика) ------------------------------------------------------------

export function Stat({ label, value, unit, sub, iconName, hero = false } = {}) {
  const el = document.createElement('div');
  el.className = `stat${hero ? ' hero' : ''}`;
  if (iconName) {
    const i = document.createElement('span');
    i.className = 'stat-icon';
    i.appendChild(icon(iconName, 22));
    el.appendChild(i);
  }
  const l = document.createElement('div');
  l.className = 'stat-label';
  l.textContent = label;
  el.appendChild(l);
  const v = document.createElement('div');
  v.className = 'stat-value num';
  v.textContent = value;
  if (unit) {
    const u = document.createElement('span');
    u.className = 'unit';
    u.textContent = unit;
    v.appendChild(u);
  }
  el.appendChild(v);
  if (sub) {
    const s = document.createElement('div');
    s.className = 'stat-sub num';
    s.textContent = sub;
    el.appendChild(s);
  }
  return el;
}

// --- Empty ------------------------------------------------------------------------

export function Empty({ iconName = 'empty', title, sub, action } = {}) {
  const el = document.createElement('div');
  el.className = 'empty';
  const i = document.createElement('div');
  i.className = 'empty-ic';
  i.appendChild(icon(iconName, 28));
  el.appendChild(i);
  const t = document.createElement('div');
  t.className = 'empty-title';
  t.textContent = title;
  el.appendChild(t);
  if (sub) {
    const s = document.createElement('div');
    s.className = 'empty-sub';
    s.textContent = sub;
    el.appendChild(s);
  }
  if (action) el.appendChild(action.el ?? action);
  return el;
}

// --- Skeleton ----------------------------------------------------------------------

export function skeletons(count = 4, kind = 'cell-sk') {
  const frag = document.createDocumentFragment();
  for (let i = 0; i < count; i++) {
    const s = document.createElement('div');
    s.className = `skeleton ${kind}`;
    frag.appendChild(s);
  }
  return frag;
}

// --- SearchBar ----------------------------------------------------------------------

export function SearchBar({ placeholder = 'Поиск…', onInput, value = '' } = {}) {
  const wrap = document.createElement('div');
  wrap.className = 'searchbar';
  wrap.appendChild(icon('search', 18));
  const input = document.createElement('input');
  input.type = 'search';
  input.placeholder = placeholder;
  input.value = value;
  input.setAttribute('aria-label', placeholder);
  let t = null;
  input.addEventListener('input', () => {
    clearTimeout(t);
    t = setTimeout(() => onInput?.(input.value.trim()), 180);
  });
  wrap.appendChild(input);
  return { el: wrap, input };
}
