/**
 * components/ui/adv.js — продвинутые кирпичики v0.5:
 * Segmented (сегмент-контрол коэффициента), Ring (кольцо прогресса SVG),
 * ProgressBar, Chip (фильтр-переключатель), Stepper (+/− у количества),
 * LineRow-хелперы для смет. Все — чистые функции (props) → { el, set*(...) }.
 */

import { icon } from './icon.js';
import { haptics } from '../../services/tg.js';

// --- Segmented ----------------------------------------------------------------

/**
 * Segmented({ options: [{value, label}], value, onChange, small })
 * Сегмент-контрол в стиле iOS/TMA 2026: скользящий индикатор, хаптика select.
 */
export function Segmented({ options = [], value, onChange, small = false } = {}) {
  const el = document.createElement('div');
  el.className = `seg${small ? ' seg-sm' : ''}`;
  el.setAttribute('role', 'radiogroup');
  const btns = new Map();

  const indicator = document.createElement('span');
  indicator.className = 'seg-indicator';

  for (const opt of options) {
    const b = document.createElement('button');
    b.type = 'button';
    b.className = 'seg-btn';
    b.setAttribute('role', 'radio');
    b.textContent = opt.label;
    b.addEventListener('click', () => {
      if (btns.get(value)?.el === b) return;
      haptics.select();
      set(opt.value);
      onChange?.(opt.value);
    });
    btns.set(opt.value, { el: b, label: opt.label });
    el.appendChild(b);
  }
  el.appendChild(indicator);

  function set(v) {
    value = v;
    for (const [val, { el: b }] of btns) {
      b.classList.toggle('active', val === v);
      b.setAttribute('aria-checked', String(val === v));
    }
    const active = btns.get(v);
    if (active) {
      requestAnimationFrame(() => {
        indicator.style.width = `${active.el.offsetWidth}px`;
        indicator.style.transform = `translateX(${active.el.offsetLeft}px)`;
      });
    }
  }
  set(value);

  return { el, set };
}

// --- Ring (кольцо прогресса) ------------------------------------------------------

/**
 * Ring({ value: 0..1, size, stroke, label, sub, tone })
 * SVG-донат: деньги оплачены/выполнено, прогресс сметы. Анимируется CSS.
 */
export function Ring({ value = 0, size = 92, stroke = 9, label = '', sub = '', tone = 'var(--accent)' } = {}) {
  const r = (size - stroke) / 2;
  const c = 2 * Math.PI * r;
  const el = document.createElement('div');
  el.className = 'ring';
  el.style.cssText = `width:${size}px;height:${size}px;`;
  const pct = Math.max(0, Math.min(1, value));
  el.innerHTML = `
<svg width="${size}" height="${size}" viewBox="0 0 ${size} ${size}" role="img" aria-label="${Math.round(pct * 100)}%">
  <circle cx="${size / 2}" cy="${size / 2}" r="${r}" fill="none" stroke="var(--border)" stroke-width="${stroke}"/>
  <circle class="ring-val" cx="${size / 2}" cy="${size / 2}" r="${r}" fill="none" stroke="${tone}"
    stroke-width="${stroke}" stroke-linecap="round"
    stroke-dasharray="${c}" stroke-dashoffset="${c}" transform="rotate(-90 ${size / 2} ${size / 2})"/>
</svg>
<div class="ring-center">${label ? `<div class="ring-label num">${label}</div>` : ''}${sub ? `<div class="ring-sub">${sub}</div>` : ''}</div>`;
  requestAnimationFrame(() => requestAnimationFrame(() => {
    el.querySelector('.ring-val')?.style.setProperty('stroke-dashoffset', String(c * (1 - pct)));
  }));
  return {
    el,
    set(v) {
      const p = Math.max(0, Math.min(1, v));
      el.querySelector('.ring-val')?.style.setProperty('stroke-dashoffset', String(c * (1 - p)));
    },
  };
}

// --- ProgressBar --------------------------------------------------------------------

export function ProgressBar({ value = 0, tone = 'var(--accent)' } = {}) {
  const el = document.createElement('div');
  el.className = 'progress';
  const fill = document.createElement('div');
  fill.className = 'progress-fill';
  fill.style.background = tone;
  el.appendChild(fill);
  const set = (v) => {
    fill.style.width = `${Math.max(0, Math.min(1, v)) * 100}%`;
  };
  requestAnimationFrame(() => requestAnimationFrame(() => set(value)));
  return { el, set };
}

// --- Chip -----------------------------------------------------------------------------

/**
 * Chip({ label, active, onToggle, tone }) — фильтр «скрытые/видимые/все».
 */
export function Chip({ label, active = false, onToggle, tone = '' } = {}) {
  const b = document.createElement('button');
  b.type = 'button';
  b.className = `chip${active ? ` active ${tone}` : ''}`;
  b.textContent = label;
  b.setAttribute('aria-pressed', String(active));
  b.addEventListener('click', () => {
    haptics.tap();
    onToggle?.(b.classList.toggle('active'));
  });
  return {
    el: b,
    set(active2) {
      b.classList.toggle('active', Boolean(active2));
      b.setAttribute('aria-pressed', String(active2));
    },
  };
}

// --- Stepper -----------------------------------------------------------------------------

/**
 * Stepper({ value, step, min, max, onChange, suffix }) — компактный +/-,
 * длинный тап ускоряет (удобно набирать 45 м²).
 */
export function Stepper({ value = 0, step = 1, min = 0, max = 1e7, onChange, suffix = '' } = {}) {
  const el = document.createElement('div');
  el.className = 'stepper';
  const minus = document.createElement('button');
  minus.type = 'button';
  minus.className = 'stepper-btn';
  minus.setAttribute('aria-label', 'Меньше');
  minus.appendChild(icon('down', 16));
  const input = document.createElement('input');
  input.type = 'text';
  input.inputMode = 'decimal';
  input.className = 'stepper-val num';
  const plus = document.createElement('button');
  plus.type = 'button';
  plus.className = 'stepper-btn';
  plus.setAttribute('aria-label', 'Больше');
  plus.appendChild(icon('up', 16));
  el.append(minus, input, plus);

  let v = value;
  const fmt = (x) => String(+parseFloat(x.toFixed(4)));
  input.value = fmt(v) + (suffix ? ` ${suffix}` : '');

  const commit = () => {
    const n = parseFloat(input.value.replace(',', '.').replace(/[^\d.-]/g, ''));
    if (!Number.isNaN(n)) v = Math.max(min, Math.min(max, n));
    input.value = fmt(v) + (suffix ? ` ${suffix}` : '');
    onChange?.(v);
  };
  input.addEventListener('blur', commit);
  input.addEventListener('keydown', (e) => { if (e.key === 'Enter') { e.preventDefault(); input.blur(); } });
  input.addEventListener('focus', () => { input.value = fmt(v); input.select(); });

  const bump = (dir) => {
    v = Math.max(min, Math.min(max, v + dir * step));
    input.value = fmt(v) + (suffix ? ` ${suffix}` : '');
    haptics.tap();
    onChange?.(v);
  };
  // удержание: первый тап шаг, дальше ускорение
  function bindHold(btn, dir) {
    let t1 = null; let t2 = null;
    const stop = () => { clearTimeout(t1); clearInterval(t2); t1 = t2 = null; };
    btn.addEventListener('pointerdown', () => {
      t1 = setTimeout(() => { t2 = setInterval(() => bump(dir), 90); }, 420);
    });
    for (const ev of ['pointerup', 'pointerleave', 'pointercancel']) btn.addEventListener(ev, stop);
  }
  bindHold(minus, -1);
  bindHold(plus, +1);

  return { el, set(x) { v = x; input.value = fmt(v) + (suffix ? ` ${suffix}` : ''); }, get value() { return v; } };
}

// --- SectionSplit (заголовок-разделитель) ---------------------------------------------------

export function SectionTitle({ text, right } = {}) {
  const el = document.createElement('div');
  el.className = 'section-title row-between';
  const l = document.createElement('span');
  l.textContent = text;
  el.appendChild(l);
  if (right) el.appendChild(right.el ?? right);
  return el;
}

// --- MoneyDiff (анимация «было → стало») ------------------------------------------------------

/** Плавная подсветка изменения числа: жёлтая вспышка на карточке. */
export function flash(el) {
  el.classList.remove('flash');
  void el.offsetWidth; // reflow для рестарта анимации
  el.classList.add('flash');
}
