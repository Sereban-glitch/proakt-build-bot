/**
 * components/ui/feedback.js — то, что «плавает» над контентом:
 * тосты, нижний лист (bottom sheet) с формами, подтверждения.
 */

import { haptics } from '../../services/tg.js';

// --- Toast -------------------------------------------------------------

function toastRoot() {
  let root = document.querySelector('.toasts');
  if (!root) {
    root = document.createElement('div');
    root.className = 'toasts';
    document.body.appendChild(root);
  }
  return root;
}

export function toast(message, { tone = 'ok', icon = '' } = {}) {
  const root = toastRoot();
  const el = document.createElement('div');
  el.className = `toast ${tone}`;
  el.setAttribute('role', 'status');
  if (icon) {
    const i = document.createElement('span');
    i.textContent = icon;
    el.appendChild(i);
  }
  el.appendChild(document.createTextNode(message));
  root.appendChild(el);
  if (tone === 'danger') haptics.error();
  setTimeout(() => {
    el.classList.add('leaving');
    setTimeout(() => el.remove(), 260);
  }, 2600);
}

// --- Sheet (нижний лист) -------------------------------------------------

/**
 * openSheet({ title, body: HTMLElement[], footer: HTMLElement[] })
 * Возвращает { el, close() }. Формы строятся вызывающим кодом,
 * лист знает только про показ/скрытие и фокус.
 */
export function openSheet({ title, body, footer } = {}) {
  const backdrop = document.createElement('div');
  backdrop.className = 'sheet-backdrop';
  backdrop.setAttribute('role', 'dialog');
  backdrop.setAttribute('aria-modal', 'true');

  const sheet = document.createElement('div');
  sheet.className = 'sheet';

  const grab = document.createElement('div');
  grab.className = 'sheet-grabber';
  sheet.appendChild(grab);

  if (title) {
    const h = document.createElement('h3');
    h.textContent = title;
    sheet.appendChild(h);
  }
  for (const node of body ?? []) sheet.appendChild(node.el ?? node);
  if (footer?.length) {
    const div = document.createElement('div');
    div.style.cssText = 'display:flex;flex-direction:column;gap:10px;margin-top:18px;';
    for (const node of footer) div.appendChild(node.el ?? node);
    sheet.appendChild(div);
  }
  backdrop.appendChild(sheet);
  document.body.appendChild(backdrop);
  requestAnimationFrame(() => requestAnimationFrame(() => backdrop.classList.add('open')));

  const close = () => {
    backdrop.classList.remove('open');
    haptics.medium();
    setTimeout(() => backdrop.remove(), 300);
  };
  backdrop.addEventListener('click', (e) => { if (e.target === backdrop) close(); });
  const onKey = (e) => { if (e.key === 'Escape') close(); };
  document.addEventListener('keydown', onKey);
  const origRemove = () => document.removeEventListener('keydown', onKey);
  const origClose = close;
  return {
    el: sheet,
    close() { origClose(); setTimeout(origRemove, 320); },
  };
}

// --- Field ------------------------------------------------------------------

export function Field({ label, type = 'text', placeholder, value = '', inputmode, autocomplete = 'off' } = {}) {
  const wrap = document.createElement('div');
  wrap.className = 'field';
  const l = document.createElement('label');
  l.textContent = label;
  wrap.appendChild(l);
  const input = document.createElement('input');
  input.type = type;
  input.placeholder = placeholder || '';
  input.value = value;
  input.autocomplete = autocomplete;
  if (inputmode) input.inputMode = inputmode;
  wrap.appendChild(input);
  return { el: wrap, input };
}

// --- ConfirmSheet ---------------------------------------------------------------

/**
 * Подтверждение в стиле 2026: нижний лист вместо нативного confirm.
 * Возвращает Promise<boolean>.
 */
export function confirmSheet({ title, text, confirmLabel = 'Да, подтверждаю', danger = true, icon = '' }) {
  return new Promise((resolve) => {
    const textEl = document.createElement('p');
    textEl.className = 'muted';
    textEl.style.cssText = 'font-size:14.5px;line-height:1.5;';
    textEl.textContent = text;
    const yes = document.createElement('button');
    yes.className = `btn btn-block ${danger ? 'btn-danger' : 'btn-primary'}`;
    yes.textContent = confirmLabel;
    yes.addEventListener('click', () => { haptics.warn(); sheet.close(); resolve(true); });
    const no = document.createElement('button');
    no.className = 'btn btn-block btn-secondary';
    no.textContent = 'Отмена';
    no.addEventListener('click', () => { sheet.close(); resolve(false); });
    const sheet = openSheet({
      title: (icon ? icon + '  ' : '') + title,
      body: [textEl],
      footer: [yes, no],
    });
  });
}
