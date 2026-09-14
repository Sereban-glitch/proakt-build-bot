/**
 * views/dashboard.js — главный экран v2: деньги под контролем.
 * Hero-метрика с кольцом оплаты, упущенная выгода, сметы в работе,
 * долги, последние акты.
 */

import { api } from '../services/api.js';
import { money, moneyShort, pluralN, dateShort } from '../services/format.js';
import { Stat, Card, Cell, actBadge, Empty, skeletons } from '../components/ui/primitives.js';
import { Header } from '../components/layout/chrome.js';
import { Ring } from '../components/ui/adv.js';
import { icon } from '../components/ui/icon.js';
import { toast } from '../components/ui/feedback.js';

export function view({ root, navigate }) {
  const el = document.createElement('div');
  el.appendChild(Header({ title: 'ПрорАКТ', subtitle: 'деньги под контролем' }));

  const content = document.createElement('main');
  content.className = 'view';
  content.appendChild(skeletons(5, 'card-sk'));
  el.appendChild(content);
  root.appendChild(el);

  api.dashboard().then((d) => {
    content.replaceChildren();

    // --- hero: выполнено + кольцо оплаты ---
    const hero = document.createElement('div');
    hero.className = 'card est-hero section';
    const paidRatio = d.stats.total > 0 ? d.stats.paid / d.stats.total : 0;
    const ring = Ring({
      value: paidRatio,
      size: 88,
      stroke: 8,
      label: `${Math.round(paidRatio * 100)}%`,
      sub: 'оплачено',
      tone: 'var(--ok)',
    });
    hero.appendChild(ring.el);

    const hv = document.createElement('div');
    hv.className = 'est-hv';
    const big = document.createElement('div');
    big.className = 'big num';
    big.textContent = moneyShort(d.stats.total);
    const sub1 = document.createElement('div');
    sub1.className = 'sub';
    sub1.textContent = `выполнено · оплачено ${moneyShort(d.stats.paid)}`;
    const sub2 = document.createElement('div');
    sub2.className = 'sub';
    const actsWord = pluralN(d.stats.acts, 'акт', 'акта', 'актов');
    sub2.textContent = `${d.stats.acts} ${actsWord.split('\u202F')[1] || actsWord} · объектов ${d.stats.objects}`;
    hv.append(big, sub1, sub2);
    hero.appendChild(hv);
    content.appendChild(hero);

    // --- плитки: сметы / фото ---
    const grid = document.createElement('div');
    grid.className = 'stat-grid';
    const mkTile = (label, value, iconName, hash) => {
      const t = document.createElement('div');
      t.className = 'stat';
      t.style.cursor = 'pointer';
      t.setAttribute('role', 'button');
      t.tabIndex = 0;
      const ic = document.createElement('span');
      ic.className = 'stat-icon';
      ic.appendChild(icon(iconName, 20));
      t.appendChild(ic);
      const l = document.createElement('div');
      l.className = 'stat-label';
      l.textContent = label;
      t.appendChild(l);
      const v = document.createElement('div');
      v.className = 'stat-value num';
      v.textContent = value;
      t.appendChild(v);
      const go = () => { location.hash = hash; };
      t.addEventListener('click', go);
      t.addEventListener('keydown', (e) => { if (e.key === 'Enter') go(); });
      return t;
    };
    grid.appendChild(mkTile('Скрытых работ, фото', String(d.stats.photos), 'camera', '#/objects'));
    grid.appendChild(mkTile('Позиций в прайсе', String(d.price_count), 'price', '#/price'));
    content.appendChild(grid);

    // --- v0.6: онбординг «3 шага до первого акта» (для тех, кто не читал инструкций) ---
    if (d.stats.objects === 0 || d.price_count === 0 || d.stats.acts === 0) {
      const onb = Card({ className: 'section onboarding' });
      const ot = document.createElement('div');
      ot.className = 'card-title';
      ot.textContent = '🚀 Старт за 3 шага';
      onb.appendChild(ot);
      const steps = [
        { done: d.price_count > 0, t: '1. Прайс — цены один раз', s: 'Вкладка «Прайс» → импорт файла или вручную. Дальше цены подставляются сами.', h: '#/price' },
        { done: d.stats.objects > 0, t: '2. Объект — квартира/дом', s: 'Вкладка «Объекты» → «+». Название и заказчик — 20 секунд.', h: '#/objects' },
        { done: d.stats.acts > 0, t: '3. Первая смета или акт', s: 'Диктуй голосом на объекте или вводи строкой — Excel соберётся сам.', h: '#/estimates' },
      ];
      for (const st of steps) {
        const row = document.createElement('div');
        row.className = 'onb-step' + (st.done ? ' done' : '');
        const mark = document.createElement('span');
        mark.className = 'onb-mark';
        mark.textContent = st.done ? '✓' : '';
        const box = document.createElement('div');
        const t = document.createElement('div');
        t.className = 'onb-t';
        t.textContent = st.t;
        const s = document.createElement('div');
        s.className = 'onb-s';
        s.textContent = st.done ? 'Готово' : st.s;
        box.append(t, s);
        row.append(mark, box);
        if (!st.done) {
          row.style.cursor = 'pointer';
          row.addEventListener('click', () => { location.hash = st.h; });
        }
        onb.appendChild(row);
      }
      content.appendChild(onb);
    }

    // --- упущенная выгода (логика отчёта бота) ---
    const debt = d.stats.total - d.stats.paid;
    const unpaid = debt > 0.009;
    const zero = d.stats.zero_acts > 0;
    if (unpaid || zero) {
      const banner = document.createElement('div');
      banner.className = 'banner section';
      const ic = document.createElement('span');
      ic.className = 'banner-ic';
      ic.appendChild(icon('alert', 20));
      banner.appendChild(ic);
      const box = document.createElement('div');
      box.style.cssText = 'display:flex;flex-direction:column;gap:4px;';
      const b1 = document.createElement('b');
      b1.textContent = 'Упущенная выгода';
      box.appendChild(b1);
      if (unpaid) {
        const p1 = document.createElement('span');
        p1.textContent = `не оплачено ${money(debt)} — напомни заказчику`;
        box.appendChild(p1);
      }
      if (zero) {
        const p2 = document.createElement('span');
        p2.textContent = `${d.stats.zero_acts} ${d.stats.zero_acts === 1 ? 'акт с суммой 0' : 'акта с суммой 0'} — работы есть, деньги не выставлены`;
        box.appendChild(p2);
      }
      banner.appendChild(box);
      banner.style.cursor = 'pointer';
      banner.addEventListener('click', () => { location.hash = '#/report'; });
      content.appendChild(banner);
    }

    // --- долги ---
    const debtsCard = Card({ className: 'section' });
    const debtsTitle = document.createElement('div');
    debtsTitle.className = 'card-title row-between';
    debtsTitle.innerHTML = '<span>Долги</span>';
    debtsCard.appendChild(debtsTitle);
    if (!d.debts?.length) {
      const okRow = document.createElement('div');
      okRow.className = 'row';
      okRow.style.gap = '10px';
      okRow.appendChild(icon('check', 20));
      okRow.appendChild(Object.assign(document.createElement('span'), {
        textContent: 'Долгов нет — всё оплачено', className: 'muted',
      }));
      debtsCard.appendChild(okRow);
    } else {
      const list = document.createElement('div');
      list.className = 'list';
      for (const a of d.debts) {
        list.appendChild(Cell({
          icon: 'acts',
          title: `Акт №${a.act_no} · ${a.object_name}`,
          sub: dateShort(a.date),
          value: money(a.total - a.paid),
          note: `из ${money(a.total)}`,
          onClick: () => navigate('#/acts/' + a.id),
        }));
      }
      debtsCard.appendChild(list);
    }
    content.appendChild(debtsCard);

    // --- последние акты ---
    const recentCard = Card({ className: 'section' });
    const rt = document.createElement('div');
    rt.className = 'card-title';
    rt.textContent = 'Последние акты';
    recentCard.appendChild(rt);
    const rlist = document.createElement('div');
    rlist.className = 'list';
    if (!d.recent_acts?.length) {
      recentCard.appendChild(Empty({
        iconName: 'acts',
        title: 'Актов ещё нет',
        sub: 'Первый акт создай в чате: «📋 Новый акт» — голосом или строкой',
      }));
    } else {
      for (const a of d.recent_acts) {
        rlist.appendChild(Cell({
          icon: 'file',
          title: `Акт №${a.act_no} · ${a.object_name}`,
          sub: dateShort(a.date),
          badge: actBadge(a.total, a.paid),
          value: money(a.total),
          onClick: () => navigate('#/acts/' + a.id),
        }));
      }
    }
    recentCard.appendChild(rlist);
    content.appendChild(recentCard);
  }).catch((e) => {
    content.replaceChildren();
    content.appendChild(Empty({ iconName: 'alert', title: 'Данные не загрузились', sub: e.message }));
    if (e.status === 401) toast(e.message, { tone: 'danger' });
  });

  return { el };
}
