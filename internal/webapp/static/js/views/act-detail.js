/**
 * views/act-detail.js — карточка акта: позиции, суммы, оплаты (добавить/
 * удалить — v0.3.8), фото акта, удаление всего акта (v0.3.6).
 */

import { api } from '../services/api.js';
import { money, dateShort, qtyLine } from '../services/format.js';
import { Card, Stat, Empty, Button, actBadge, skeletons } from '../components/ui/primitives.js';
import { Header } from '../components/layout/chrome.js';
import { Field, openSheet, toast, confirmSheet } from '../components/ui/feedback.js';
import { haptics } from '../services/tg.js';
import { icon } from '../components/ui/icon.js';

export function view({ root, params, navigate }) {
  const id = Number(params.id);
  const el = document.createElement('div');
  el.appendChild(Header({ title: 'Акт', subtitle: 'загружаю…' }));
  const content = document.createElement('main');
  content.className = 'view';
  content.appendChild(skeletons(3, 'card-sk'));
  el.appendChild(content);
  root.appendChild(el);

  function head(a) {
    return Header({
      title: `Акт №${a.act_no}`,
      subtitle: `${a.object_name} · ${dateShort(a.date)}`,
      back: () => history.back(),
    });
  }

  api.act(id).then((d) => {
    const a = d.act;
    content.replaceChildren();
    el.querySelector('.app-header')?.replaceWith(head(a));
    // --- деньги акта ---
    const grid = document.createElement('div');
    grid.className = 'stat-grid wide-first';
    const balance = a.total - a.paid;
    grid.appendChild(Stat({
      label: 'Сумма акта',
      value: money(a.total),
      sub: `оплачено ${money(a.paid)} · остаток ${money(balance)}`,
      iconName: 'wallet',
      hero: true,
    }));
    const statusStat = document.createElement('div');
    statusStat.className = 'stat';
    statusStat.style.cssText = 'grid-column:1/-1;display:flex;align-items:center;justify-content:space-between;';
    const lbl = document.createElement('span');
    lbl.className = 'stat-label';
    lbl.textContent = 'Статус';
    statusStat.appendChild(lbl);
    statusStat.appendChild(actBadge(a.total, a.paid));
    grid.appendChild(statusStat);
    content.appendChild(grid);

    // --- позиции ---
    const linesCard = Card({ className: 'section' });
    const lt = document.createElement('div');
    lt.className = 'card-title';
    lt.textContent = 'Позиции';
    linesCard.appendChild(lt);
    if (!d.lines.length) {
      linesCard.appendChild(Empty({ iconName: 'acts', title: 'Позиций нет', sub: 'Акт создан пустым — суммы будут нулевыми' }));
    } else {
      const table = document.createElement('table');
      table.className = 'lines';
      table.innerHTML = `<thead><tr><th>№</th><th>Наименование</th><th class="t-right">Кол-во</th><th class="t-right">Цена</th><th class="t-right">Сумма</th></tr></thead>`;
      const tbody = document.createElement('tbody');
      for (const l of d.lines) {
        const tr = document.createElement('tr');
        const tdPos = document.createElement('td');
        tdPos.className = 'muted';
        tdPos.textContent = String(l.pos);
        const tdName = document.createElement('td');
        tdName.textContent = l.name;
        const tdQty = document.createElement('td');
        tdQty.className = 't-right';
        tdQty.textContent = qtyLine(l.qty, l.unit);
        const tdPrice = document.createElement('td');
        tdPrice.className = 't-right';
        tdPrice.textContent = l.price ? money(l.price, { withUnit: false }) : '—';
        const tdSum = document.createElement('td');
        tdSum.className = 't-right t-sum';
        tdSum.textContent = money(l.sum, { withUnit: false });
        tr.append(tdPos, tdName, tdQty, tdPrice, tdSum);
        tbody.appendChild(tr);
      }
      table.appendChild(tbody);
      linesCard.appendChild(table);
      const totalRow = document.createElement('div');
      totalRow.className = 'row-between';
      totalRow.style.cssText = 'margin-top:12px;font-weight:800;';
      totalRow.innerHTML = `<span class="muted">Итого</span><span class="num">${money(a.total)}</span>`;
      linesCard.appendChild(totalRow);
    }
    content.appendChild(linesCard);

    // --- оплаты ---
    const payCard = Card({ className: 'section' });
    const payTitle = document.createElement('div');
    payTitle.className = 'card-title row-between';
    payTitle.innerHTML = `<span>Оплаты</span><span class="num">${money(a.paid)}</span>`;
    payCard.appendChild(payTitle);
    const payList = document.createElement('div');
    payList.className = 'list';
    if (!d.payments?.length) {
      const none = document.createElement('div');
      none.className = 'hint-text';
      none.textContent = 'Оплат ещё не было — добавь первую кнопку ниже';
      payCard.appendChild(none);
    } else {
      for (const p of d.payments) {
        const row = document.createElement('div');
        row.className = 'cell';
        const ic = document.createElement('span');
        ic.className = 'cell-ic';
        ic.appendChild(icon('wallet', 20));
        row.appendChild(ic);
        const main = document.createElement('div');
        main.className = 'grow';
        main.innerHTML = `<div class="cell-title num">${money(p.amount)}</div><div class="cell-sub">${dateShort(p.at)}${p.note ? ' · ' + p.note : ''}</div>`;
        row.appendChild(main);
        const del = document.createElement('button');
        del.className = 'icon-btn';
        del.setAttribute('aria-label', 'Удалить оплату');
        del.appendChild(icon('trash', 16));
        del.addEventListener('click', async () => {
          haptics.warn();
          const ok = await confirmSheet({
            title: 'Убрать оплату?',
            text: `Уйдёт ${money(p.amount)} от ${dateShort(p.at)} — долг по акту сразу вернётся к правде.`,
            confirmLabel: 'Убрать оплату',
            icon: '🗑',
          });
          if (!ok) return;
          try {
            await api.deletePayment(p.id);
            haptics.success();
            toast('Оплату убрал', { icon: '🗑' });
            rerender();
          } catch (err) {
            toast(err.message, { tone: 'danger' });
          }
        });
        row.appendChild(del);
        payList.appendChild(row);
      }
      payCard.appendChild(payList);
    }
    const addPay = Button({
      label: 'Отметить оплату', iconName: 'plus', variant: 'primary', block: true,
      onClick: () => openPaySheet(),
    });
    addPay.el.style.marginTop = '14px';
    payCard.appendChild(addPay.el);
    content.appendChild(payCard);

    // --- фото акта ---
    const phCard = Card({ className: 'section' });
    const phT = document.createElement('div');
    phT.className = 'card-title';
    phT.textContent = 'Фото акта';
    phCard.appendChild(phT);
    if (!d.photos?.length) {
      phCard.appendChild(Empty({ iconName: 'camera', title: 'Фото нет', sub: 'Пришли снимок в чат и привяжи к этому акту' }));
    } else {
      const gridPh = document.createElement('div');
      gridPh.className = 'photo-grid';
      for (const p of d.photos) {
        const cell = document.createElement('div');
        cell.className = 'ph';
        const img = document.createElement('img');
        img.loading = 'lazy';
        img.alt = p.caption || 'фото акта';
        img.src = api.photoURL(p);
        cell.appendChild(img);
        gridPh.appendChild(cell);
      }
      phCard.appendChild(gridPh);
    }
    content.appendChild(phCard);

    // --- удаление акта (v0.3.6) ---
    const delAct = Button({
      label: 'Удалить акт целиком', iconName: 'trash', variant: 'danger', block: true,
      onClick: async () => {
        const ok = await confirmSheet({
          title: `Удалить акт №${a.act_no}?`,
          text: `Уйдут ${d.lines.length} поз. и оплаты (${money(a.paid)}). Фото отвяжутся от акта, но останутся у объекта.`,
          confirmLabel: 'Удалить акт',
          icon: '🗑',
        });
        if (!ok) return;
        try {
          await api.deleteAct(a.id);
          haptics.success();
          toast(`Акт №${a.act_no} удалён`, { icon: '🗑' });
          navigate('#/acts');
        } catch (err) {
          toast(err.message, { tone: 'danger' });
        }
      },
    });
    const delWrap = document.createElement('div');
    delWrap.className = 'section';
    delWrap.appendChild(delAct.el);
    content.appendChild(delWrap);

    // --- лист добавления оплаты ---
    function openPaySheet() {
      const hint = document.createElement('p');
      hint.className = 'muted';
      hint.style.cssText = 'font-size:13.5px;margin-bottom:14px;';
      hint.textContent = `Акт №${a.act_no} · ${a.object_name}. Оплачено ${money(a.paid)} из ${money(a.total)} — остаток ${money(balance)}.`;
      const amount = Field({ label: 'Сумма, грн', inputmode: 'decimal', placeholder: '10000' });
      const save = Button({
        label: 'Записать оплату', block: true,
        onClick: async () => {
          const val = parseFloat(amount.input.value.replace(',', '.').replace(/\s/g, ''));
          if (!(val > 0)) {
            haptics.error();
            toast('Введи сумму числом, например 5000', { tone: 'danger' });
            return;
          }
          save.setLoading(true);
          try {
            await api.addPayment(a.id, val);
            sheet.close();
            haptics.success();
            toast(val >= balance - 0.009 ? 'Акт закрыт полностью 🎉' : 'Оплата записана', { icon: '💵' });
            rerender();
          } catch (err) {
            haptics.error();
            toast(err.message, { tone: 'danger' });
          } finally {
            save.setLoading(false);
          }
        },
      });
      const sheet = openSheet({ title: 'Оплата по акту', body: [hint, amount.el], footer: [save.el] });
      setTimeout(() => amount.input.focus(), 350);
    }
  }).catch((e) => {
    content.replaceChildren(Empty({ iconName: 'alert', title: 'Акт не открылся', sub: e.message }));
  });

  function rerender() {
    // простая перезагрузка данных без смены маршрута
    import('./act-detail.js').then((m) => {
      content.replaceChildren();
      const fresh = m.view({ root: el.parentNode ?? root, params, navigate });
      el.replaceWith(fresh.el);
    });
  }

  return { el };
}
