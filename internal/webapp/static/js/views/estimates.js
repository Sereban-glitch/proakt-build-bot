/**
 * views/estimates.js — список смет (v0.5): карточки с деньгами, статусом,
 * прогрессом закрытия актами и долей скрытых работ. FAB — новая смета.
 */

import { api } from '../services/api.js';
import { money, moneyShort, dateShort } from '../services/format.js';
import { Card, Empty, skeletons } from '../components/ui/primitives.js';
import { Header } from '../components/layout/chrome.js';
import { ProgressBar } from '../components/ui/adv.js';
import { icon } from '../components/ui/icon.js';
import { toast, openSheet, Field } from '../components/ui/feedback.js';
import { haptics } from '../services/tg.js';

const STATUS_LABEL = { draft: 'черновик', sent: 'отправлена', approved: 'согласована', done: 'закрыта' };

export function statusBadge(status) {
  const s = document.createElement('span');
  s.className = `est-status ${status}`;
  s.textContent = STATUS_LABEL[status] || status;
  return s;
}

export function view({ root }) {
  const el = document.createElement('div');
  el.appendChild(Header({ title: 'Сметы', subtitle: 'план работ и деньги до ремонта' }));

  // вход в «админку» шаблонов — техцикл одним тапом (вне скролла,
  // чтобы replaceChildren при загрузке не стирал кнопку)
  const tplRow = document.createElement('div');
  tplRow.style.cssText = 'display:flex;justify-content:flex-end;padding:2px 16px 0;';
  const tplBtn = document.createElement('button');
  tplBtn.className = 'btn btn-ghost';
  tplBtn.style.cssText = 'font-size:12.5px;color:var(--info);';
  tplBtn.appendChild(icon('template', 15));
  const tplLbl = document.createElement('span');
  tplLbl.textContent = 'Шаблоны работ';
  tplBtn.appendChild(tplLbl);
  tplBtn.addEventListener('click', () => { haptics.tap(); location.hash = '#/templates'; });
  tplRow.appendChild(tplBtn);
  el.appendChild(tplRow);

  const content = document.createElement('main');
  content.className = 'view';
  content.appendChild(skeletons(3, 'card-sk'));
  el.appendChild(content);

  // FAB: новая смета
  const fabWrap = document.createElement('div');
  fabWrap.className = 'fab-wrap';
  const fab = document.createElement('button');
  fab.className = 'fab';
  fab.setAttribute('aria-label', 'Новая смета');
  fab.appendChild(icon('plus', 24));
  fab.addEventListener('click', () => { haptics.medium(); openCreateSheet(content); });
  fabWrap.appendChild(fab);
  el.appendChild(fabWrap);

  root.appendChild(el);

  api.estimates().then((list) => {
    content.replaceChildren();
    if (!list.length) {
      content.appendChild(Empty({
        iconName: 'estimate',
        title: 'Смет ещё нет',
        sub: 'Смета — план работ до ремонта: позиции из прайса, скрытая подготовка, коэффициент сложности. Заказчику — ссылка с фото.',
        action: (() => {
          const b = document.createElement('button');
          b.className = 'btn btn-primary';
          b.textContent = '+ Первая смета';
          b.addEventListener('click', () => openCreateSheet(content));
          return b;
        })(),
      }));
      return;
    }
    for (const est of list) content.appendChild(estCard(est));
  }).catch((e) => {
    content.replaceChildren();
    content.appendChild(Empty({ iconName: 'alert', title: 'Сметы не загрузились', sub: e.message }));
  });

  function estCard(est) {
    const card = Card({ className: 'est-card section', clickable: true, onClick: () => { location.hash = `#/estimates/${est.id}`; } });

    const top = document.createElement('div');
    top.className = 'est-top';
    const ic = document.createElement('span');
    ic.className = 'est-ic';
    ic.appendChild(icon('estimate', 20));
    const nameBox = document.createElement('div');
    nameBox.style.cssText = 'min-width:0;flex:1;';
    const nm = document.createElement('div');
    nm.className = 'est-name ellipsis';
    nm.textContent = est.title;
    const ob = document.createElement('div');
    ob.className = 'est-obj ellipsis';
    ob.textContent = `${est.object_name} · ${dateShort(est.created_at)}`;
    nameBox.append(nm, ob);
    top.append(ic, nameBox, statusBadge(est.status));
    card.appendChild(top);

    const moneyRow = document.createElement('div');
    moneyRow.className = 'est-money';
    const total = document.createElement('span');
    total.className = 'est-total';
    total.textContent = moneyShort(est.total);
    const parts = document.createElement('span');
    parts.className = 'est-parts';
    parts.textContent = `${est.lines} поз. · скрытых ${moneyShort(est.hidden)} (${est.hidden_share}%)`;
    moneyRow.append(total, parts);
    card.appendChild(moneyRow);

    const bar = ProgressBar({ value: est.total > 0 ? est.done_sum / est.total : 0 });
    card.appendChild(bar.el);
    const sub = document.createElement('div');
    sub.className = 'est-parts';
    sub.textContent = est.total > 0
      ? `закрыто актами ${moneyShort(est.done_sum)} из ${moneyShort(est.total)}`
      : 'позиций пока нет — добавьте из прайса или шаблона';
    card.appendChild(sub);
    return card;
  }

  function openCreateSheet() {
    let objectsCache = null;
    const titleF = Field({ label: 'Название сметы', placeholder: 'Малярные работы под покраску', value: '' });
    const noteF = Field({ label: 'Заметка (необязательно)', placeholder: 'потолки 300 см, сжатые сроки…' });

    const objBox = document.createElement('div');
    objBox.className = 'field';
    const objLabel = document.createElement('label');
    objLabel.textContent = 'Объект';
    objBox.appendChild(objLabel);
    const sel = document.createElement('select');
    sel.className = 'select';
    sel.innerHTML = '<option>Загружаю объекты…</option>';
    objBox.appendChild(sel);

    // коэффициент
    const coeffBox = document.createElement('div');
    coeffBox.className = 'field';
    const coeffLabel = document.createElement('label');
    coeffLabel.textContent = 'Коэффициент сложности';
    const coeffHint = document.createElement('div');
    coeffHint.className = 'coeff-note';
    coeffHint.textContent = '280 см → 1.10 · 290 см → 1.20 · 300 см → 1.30 · потолки → 1.30 · лестницы → 2.0';
    const coeffSel = document.createElement('select');
    coeffSel.className = 'select';
    for (const c of ['1', '1.1', '1.2', '1.3', '1.5', '2']) {
      const o = document.createElement('option');
      o.value = c;
      o.textContent = c === '1' ? '1.00 — без коэффициента' : `× ${c}`;
      coeffSel.appendChild(o);
    }
    coeffBox.append(coeffLabel, coeffSel, coeffHint);

    api.objects().then((objs) => {
      objectsCache = objs;
      sel.innerHTML = '';
      if (!objs.length) {
        sel.innerHTML = '<option value="">Сначала создайте объект во вкладке «Объекты»</option>';
        return;
      }
      for (const o of objs) {
        const opt = document.createElement('option');
        opt.value = o.id;
        opt.textContent = o.name;
        sel.appendChild(opt);
      }
    });

    const create = document.createElement('button');
    create.className = 'btn btn-primary btn-block';
    create.textContent = 'Создать смету';
    create.addEventListener('click', async () => {
      const objectId = Number(sel.value);
      if (!objectsCache?.length || !objectId) {
        toast('Сначала создайте объект', { tone: 'danger', icon: '🏠' });
        return;
      }
      const title = titleF.input.value.trim();
      if (title.length < 2) {
        toast('Дайте смете название', { tone: 'danger', icon: '✍️' });
        return;
      }
      create.setLoading?.(true);
      create.disabled = true;
      try {
        const est = await api.createEstimate(objectId, title, Number(coeffSel.value), noteF.input.value.trim());
        haptics.success();
        toast('Смета создана — добавляйте позиции', { icon: '🧾' });
        sheet.close();
        location.hash = `#/estimates/${est.id}`;
      } catch (e) {
        create.disabled = false;
        toast(e.message, { tone: 'danger' });
      }
    });

    const sheet = openSheet({
      title: '🧾 Новая смета',
      body: [titleF.el, objBox, coeffBox, noteF.el],
      footer: [create],
    });
  }

  return { el };
}
