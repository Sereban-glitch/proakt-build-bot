/**
 * views/report.js — «Отчёт»: сводка за всё время + упущенная выгода
 * (повторяет логику showReport бота v0.3.7) + долги списком.
 */

import { api } from '../services/api.js';
import { money, pluralN } from '../services/format.js';
import { Card, Stat, Empty, Cell, skeletons } from '../components/ui/primitives.js';
import { Header } from '../components/layout/chrome.js';
import { icon } from '../components/ui/icon.js';
import { dateShort } from '../services/format.js';

export function view({ root, navigate }) {
  const el = document.createElement('div');
  el.appendChild(Header({ title: 'Отчёт', subtitle: 'сводка за всё время' }));

  const content = document.createElement('main');
  content.className = 'view';
  content.appendChild(skeletons(4, 'card-sk'));
  el.appendChild(content);
  root.appendChild(el);

  api.dashboard().then((d) => {
    content.replaceChildren();
    const st = d.stats;
    const debt = st.total - st.paid;

    // --- сводка ---
    const grid = document.createElement('div');
    grid.className = 'stat-grid wide-first';
    grid.appendChild(Stat({
      label: 'Всего выполнено',
      value: money(st.total),
      sub: `оплачено ${money(st.paid)} · остаток ${money(debt)}`,
      iconName: 'wallet',
      hero: true,
    }));
    grid.appendChild(Stat({ label: 'Объектов', value: String(st.objects), iconName: 'objects' }));
    grid.appendChild(Stat({ label: 'Актов', value: String(st.acts), iconName: 'acts' }));
    grid.appendChild(Stat({ label: 'Фото', value: String(st.photos), iconName: 'camera' }));
    grid.appendChild(Stat({ label: 'Прайс', value: String(d.price_count), sub: 'позиций', iconName: 'price' }));
    content.appendChild(grid);

    // --- упущенная выгода ---
    const missed = document.createElement('div');
    missed.className = 'section';
    if (debt > 0.009 || st.zero_acts > 0) {
      const card = Card();
      const t = document.createElement('div');
      t.className = 'card-title';
      t.style.color = 'var(--warn)';
      t.textContent = 'Упущенная выгода — деньги мимо кармана';
      card.appendChild(t);
      if (debt > 0.009) {
        const b = document.createElement('div');
        b.className = 'banner';
        b.style.marginBottom = st.zero_acts > 0 ? '10px' : '0';
        const ic = document.createElement('span');
        ic.className = 'banner-ic';
        ic.appendChild(icon('alert', 18));
        b.appendChild(ic);
        b.appendChild(Object.assign(document.createElement('span'), {
          textContent: `Не оплачено ${money(debt)} (${pluralN(st.unpaid_acts, 'акт', 'акта', 'актов')}) — напомни заказчику`,
        }));
        card.appendChild(b);
      }
      if (st.zero_acts > 0) {
        const b2 = document.createElement('div');
        b2.className = 'banner danger';
        const ic2 = document.createElement('span');
        ic2.className = 'banner-ic';
        ic2.appendChild(icon('alert', 18));
        b2.appendChild(ic2);
        b2.appendChild(Object.assign(document.createElement('span'), {
          textContent: `${pluralN(st.zero_acts, 'Акт', 'акта', 'актов')} с суммой 0 — работы сделаны, деньги не выставлены: оцени по прайсу`,
        }));
        card.appendChild(b2);
      }
      missed.appendChild(card);
    } else {
      const okCard = Card();
      const okRow = document.createElement('div');
      okRow.className = 'row';
      okRow.style.gap = '10px';
      const ic = document.createElement('span');
      ic.appendChild(icon('check', 22));
      okRow.appendChild(ic);
      okRow.appendChild(Object.assign(document.createElement('span'), {
        textContent: 'Долгов нет, все акты с ценой — деньги под контролем!',
      }));
      okCard.appendChild(okRow);
      missed.appendChild(okCard);
    }
    content.appendChild(missed);

    // --- долги списком ---
    const debtsCard = Card({ className: 'section' });
    const dt = document.createElement('div');
    dt.className = 'card-title';
    dt.textContent = 'Должники';
    debtsCard.appendChild(dt);
    if (!d.debts?.length) {
      debtsCard.appendChild(Empty({ iconName: 'check', title: 'Все долги закрыты', sub: 'Здесь появятся акты с остатком к оплате' }));
    } else {
      const list = document.createElement('div');
      list.className = 'list';
      for (const a of d.debts) {
        list.appendChild(Cell({
          icon: 'wallet',
          title: `Акт №${a.act_no} · ${a.object_name}`,
          sub: `${a.customer || 'без заказчика'} · ${dateShort(a.date)}`,
          value: money(a.total - a.paid),
          note: `из ${money(a.total)}`,
          onClick: () => navigate('#/acts/' + a.id),
        }));
      }
      debtsCard.appendChild(list);
    }
    content.appendChild(debtsCard);
  }).catch((e) => {
    content.replaceChildren(Empty({ iconName: 'alert', title: 'Отчёт не собрался', sub: e.message }));
  });

  return { el };
}
